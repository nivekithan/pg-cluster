import z, { ZodObject } from "zod";
import { Crd } from "./lib/crd.ts";
import { logger } from "./logger.ts";
import { Effect, Queue } from "effect";
import {
  createNamespacedDeployment,
  readNamespacedDeployment,
} from "./lib/deloyment.ts";
import { ResourceUsage, type KubeConfig } from "@kubernetes/client-node";
import type { Logger } from "pino";
import { Images } from "./images.ts";
import { createNamespacedPvc, readNamespacedPvc } from "./lib/pvc.ts";

const pgCrdSpec = z.object({
  spec: z.object({
    storage: z.string(),
  }),
});

export const pgCrd = new Crd({
  group: "kube.nivekithan.com",
  kind: "Postgres",
  spec: pgCrdSpec,
  logger: logger,
});

export type ReconcileCrdArgs = Parameters<typeof reconcileCrd>[0];

export type PgCrdApi = ReturnType<Crd<typeof pgCrdSpec>["getApi"]>;

export const reconcileCrd = Effect.fn("reconcileCrd")(function* ({
  name,
  namespace,
  pgCrdApi,
  kc,
  logger,
}: {
  name: string;
  namespace: string;
  pgCrdApi: PgCrdApi;
  kc: KubeConfig;
  logger: Logger;
}) {
  const childLogger = logger.child({ postgresCrd: { name, namespace } });

  childLogger.info({ action: "RECONCILE_LOOP_STARTED" });

  const postgres = yield* pgCrdApi.getNamespacedObject({
    name,
    namespace,
  });

  childLogger.debug({ msg: "Fetched Postgres object", postgres });

  childLogger.info({
    action: "CHECKING_EXISTING_DEPLOYMENT",
    args: { name: "busybox", namespace },
  });

  yield* ensurePostgresDeployment({
    kc,
    logger,
    pgCrdApi,
    postgresCrd: { name, namespace },
  });

  childLogger.info({ action: "RECONCILE_LOOP_COMPLETED" });
});

const ensurePostgresDeployment = Effect.fn("ensurePostgresDeployment")(
  function* ({
    kc,
    postgresCrd: { name, namespace },
    logger,
    pgCrdApi,
  }: {
    postgresCrd: { name: string; namespace: string };
    kc: KubeConfig;
    logger: Logger;
    pgCrdApi: PgCrdApi;
  }) {
    const childLogger = logger.child({});

    childLogger.info({ action: "RECONCILE_LOOP_STARTED" });

    const postgres = yield* pgCrdApi.getNamespacedObject({
      name,
      namespace,
    });

    childLogger.debug({ msg: "Fetched Postgres object", postgres });

    childLogger.info({
      action: "CHECKING_EXISTING_DEPLOYMENT",
      args: { name: "busybox", namespace },
    });

    const deploymentName = `${name}-db-server`;
    const existingDeployment = yield* readNamespacedDeployment({
      kc,
      name: deploymentName,
      namespace,
    }).pipe(Effect.catchTag("DeploymentNotFound", () => Effect.succeed(null)));

    childLogger.info({
      action: "CHECKING_EXISTING_DEPLOYMENT_RESULT",
      args: { name: deploymentName, namespace },
      deployment: existingDeployment,
    });

    if (existingDeployment) {
      return;
    }

    childLogger.info({ action: "CREATING_BUSYBOX_DEPLOYMENT" });

    const postgresDeployment = yield* createNamespacedDeployment({
      kc,
      namespace: namespace,
      body: {
        apiVersion: "apps/v1",
        kind: "Deployment",

        metadata: {
          name: deploymentName,
          namespace: namespace,
          ownerReferences: [
            {
              apiVersion: postgres.apiVersion,
              kind: postgres.kind,
              name: postgres.metadata.name,
              uid: postgres.metadata.uid,
              blockOwnerDeletion: false,
              controller: true,
            },
          ],
        },

        spec: {
          replicas: 1,

          selector: {
            matchLabels: {
              app: deploymentName,
            },
          },
          template: {
            metadata: {
              labels: {
                app: deploymentName,
              },
            },
            spec: {
              volumes: [
                {
                  name: "postgres-socket",
                  emptyDir: {
                    sizeLimit: "16Mi",
                  },
                },
              ],

              containers: [
                {
                  name: "db-server",
                  image: Images.postgres,
                  env: [
                    {
                      name: "POSTGRES_PASSWORD",
                      value: "password",
                    },
                  ],

                  volumeMounts: [
                    {
                      name: "postgres-socket",
                      mountPath: "/tmp/postgres",
                    },
                  ],
                },
              ],
            },
          },
        },
      },
    });

    childLogger.info({
      action: "CREATED_POSTGRES_DEPLOYMENT",
      deployment: postgresDeployment,
    });
  },
);
