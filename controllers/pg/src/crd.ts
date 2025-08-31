import z, { ZodObject } from "zod";
import { Crd } from "./lib/crd.ts";
import { logger } from "./logger.ts";
import { Effect, Queue } from "effect";
import {
  createNamespacedDeployment,
  readNamespacedDeployment,
} from "./lib/deloyment.ts";
import type { KubeConfig } from "@kubernetes/client-node";
import type { Logger } from "pino";

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

export const reconcileCrd = Effect.fn("reconcileCrd")(function* ({
  name,
  namespace,
  pgCrdApi,
  kc,
  logger,
}: {
  name: string;
  namespace: string;
  pgCrdApi: ReturnType<Crd<typeof pgCrdSpec>["getApi"]>;
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

  const existingDeployment = yield* readNamespacedDeployment({
    kc,
    name: "busybox",
    namespace,
  }).pipe(Effect.catchTag("DeploymentNotFound", () => Effect.succeed(null)));

  childLogger.info({
    action: "CHECKING_EXISTING_DEPLOYMENT_RESULT",
    args: { name: "busybox", namespace },
    deployment: existingDeployment,
  });

  if (!existingDeployment) {
    childLogger.info({ action: "CREATING_BUSYBOX_DEPLOYMENT" });
    const busyboxDeployment = yield* createNamespacedDeployment({
      kc,
      namespace: namespace,
      body: {
        apiVersion: "apps/v1",
        kind: "Deployment",
        metadata: {
          name: "busybox",
          namespace: namespace,
        },
        spec: {
          replicas: 1,
          selector: {
            matchLabels: {
              app: "busybox",
            },
          },
          template: {
            metadata: {
              labels: {
                app: "busybox",
              },
            },
            spec: {
              containers: [
                {
                  name: "busybox",
                  image: "busybox",
                  command: ["sh", "-c", "echo Hello Kubernetes! && sleep 3600"],
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
  }

  childLogger.info({ action: "RECONCILE_LOOP_COMPLETED" });
});
