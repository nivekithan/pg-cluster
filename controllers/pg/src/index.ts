import {
  AppsV1Api,
  CustomObjectsApi,
  KubeConfig,
  makeInformer,
} from "@kubernetes/client-node";
import { pgCrd } from "./crd.ts";
import { logger } from "./logger.ts";
import { Effect } from "effect";
import {
  createNamespacedDeployment,
  readNamespacedDeployment,
} from "./lib/deloyment.ts";

const kc = new KubeConfig();
kc.loadFromDefault();

const pgCrdApi = pgCrd.getApi(kc);
const appsApi = kc.makeApiClient(AppsV1Api);

const podsInformer = makeInformer(kc, pgCrd.apiPath(), async () => {
  try {
    const res = await Effect.runPromise(pgCrdApi.listForAllNamesapce);
    return res;
  } catch (err) {
    logger.error({ err });
    throw err;
  }
});

const reconcileCrd = Effect.fn("reconcileCrd")(function* ({
  name,
  namespace,
}: {
  name: string;
  namespace: string;
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
    name,
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

podsInformer.on("add", (event) => {
  if (!event.metadata?.name || !event.metadata?.namespace) return;

  Effect.runPromise(
    reconcileCrd({
      name: event.metadata?.name,
      namespace: event.metadata?.namespace,
    }),
  );
});

podsInformer.on("update", (event) => {
  if (!event.metadata?.name || !event.metadata?.namespace) return;

  Effect.runPromise(
    reconcileCrd({
      name: event.metadata?.name,
      namespace: event.metadata?.namespace,
    }),
  );
});

podsInformer.on("delete", (event) => {
  if (!event.metadata?.name || !event.metadata?.namespace) return;

  Effect.runPromise(
    reconcileCrd({
      name: event.metadata?.name,
      namespace: event.metadata?.namespace,
    }),
  );
});

podsInformer.on("change", (event) => {
  if (!event.metadata?.name || !event.metadata?.namespace) return;

  Effect.runPromise(
    reconcileCrd({
      name: event.metadata?.name,
      namespace: event.metadata?.namespace,
    }),
  );
});

await podsInformer.start();
