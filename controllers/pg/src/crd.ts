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

    const socketPvc = yield* ensureSocketPvc({
      kc,
      logger: childLogger,
      pgCrdApi,
      postgresCrd: { name, namespace },
    });

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

    const busyboxDeployment = yield* createNamespacedDeployment({
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
                  persistentVolumeClaim: {
                    claimName: getSocketPvcName(name),
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
      action: "CREATED_BUSYBOX_DEPLOYMENT",
      deployment: busyboxDeployment,
    });
  },
);

const getSocketPvcName = (name: string) => `${name}-socket-pvc`;

const ensureSocketPvc = Effect.fn("ensureSocketPvc")(function* ({
  kc,
  logger,
  pgCrdApi,
  postgresCrd: { name, namespace },
}: {
  postgresCrd: { name: string; namespace: string };
  kc: KubeConfig;
  logger: Logger;
  pgCrdApi: PgCrdApi;
}) {
  const pvcName = getSocketPvcName(name);

  const childLogger = logger.child({ pvcName });

  childLogger.info({ action: "CHECKING_FOR_EXISTING_PVC" });

  const pvc = yield* readNamespacedPvc({ kc, name: pvcName, namespace }).pipe(
    Effect.catchTag("pvcNotFound", () => Effect.succeed(null)),
  );

  childLogger.info({ action: "CHECK_EXISTING_PVC_RESULT", pvc });

  if (pvc) {
    childLogger.info({ action: "PVC_ALREADY_EXISTS", pvc });
    return pvc;
  }

  childLogger.info({ action: "CREATING_NEW_PVC" });

  const socketPvc = yield* createNamespacedPvc({
    kc,
    namespace,
    body: {
      metadata: {
        name: pvcName,
        namespace,
      },
      spec: {
        accessModes: ["ReadWriteOnce"],
        resources: {
          requests: {
            storage: "16Mi",
          },
        },
      },
    },
  });

  childLogger.info({ action: "CREATED_NEW_PVC", pvc: socketPvc });

  return socketPvc;
});
