import {
  AppsV1Api,
  CustomObjectsApi,
  KubeConfig,
  makeInformer,
} from "@kubernetes/client-node";
import { pgCrd } from "./crd.ts";
import { logger } from "./logger.ts";
import { Effect } from "effect";

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

async function reconcileCrd({
  name,
  namespace,
}: {
  name: string;
  namespace: string;
}) {
  const childLogger = logger.child({ postgresCrd: { name, namespace } });

  childLogger.info({ action: "RECONCILE_LOOP_STARTED" });

  const postgres = await pgCrdApi.getNamespacedObject({
    name,
    namespace,
  });

  childLogger.debug({ msg: "Fetched Postgres object", postgres });

  childLogger.info({
    action: "CHECKING_EXISTING_DEPLOYMENT",
    args: { name: "busybox", namespace },
  });

  const checkExistingDeployment = await appsApi
    .readNamespacedDeployment({
      name: "busybox",
      namespace,
    })
    .catch((err) => {
      if (err instanceof Error && err.message.includes("404")) {
        return null;
      }

      throw err;
    });

  childLogger.info({
    action: "FOUND_EXISTING_DEPLOYMENT",
    args: { name: "busybox", namespace },
    deployment: checkExistingDeployment,
  });

  if (!checkExistingDeployment) {
    childLogger.info({ action: "CREATING_BUSYBOX_DEPLOYMENT" });
    const busyboxDeployment = await appsApi.createNamespacedDeployment({
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
}

podsInformer.on("add", (event) => {
  if (!event.metadata?.name || !event.metadata?.namespace) return;

  reconcileCrd({
    name: event.metadata?.name,
    namespace: event.metadata?.namespace,
  });
});

podsInformer.on("update", (event) => {
  if (!event.metadata?.name || !event.metadata?.namespace) return;

  reconcileCrd({
    name: event.metadata?.name,
    namespace: event.metadata?.namespace,
  });
});

podsInformer.on("delete", (event) => {
  if (!event.metadata?.name || !event.metadata?.namespace) return;

  reconcileCrd({
    name: event.metadata?.name,
    namespace: event.metadata?.namespace,
  });
});

podsInformer.on("change", (event) => {
  if (!event.metadata?.name || !event.metadata?.namespace) return;

  reconcileCrd({
    name: event.metadata?.name,
    namespace: event.metadata?.namespace,
  });
});

await podsInformer.start();
