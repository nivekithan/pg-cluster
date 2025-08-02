import * as k8s from "@kubernetes/client-node";
import { logger } from "./logger.ts";
import { SimpleWebServerCrd } from "./simpleWebServer.ts";

const kubeConfig = new k8s.KubeConfig();
kubeConfig.loadFromDefault();

async function watchWebServers() {
  const watch = new k8s.Watch(kubeConfig);

  const watchPath = `/apis/${SimpleWebServerCrd.spec.group}/${SimpleWebServerCrd.spec.versions[0].name}/${SimpleWebServerCrd.spec.names.plural}`;

  const handleAdded = async (obj: any) => {
    logger.info(
      {
        event: "ADDED",
        obj,
      },
      `WebServer ${obj.metadata.namespace}/${obj.metadata.name} created with port ${obj.spec.port}`
    );
  };

  const handleModified = async (obj: any) => {
    logger.info(
      {
        event: "MODIFIED",
        obj,
      },
      `WebServer ${obj.metadata.namespace}/${obj.metadata.name} updated to port ${obj.spec.port}`
    );
  };

  const handleDeleted = async (obj: any) => {
    logger.info(
      {
        event: "DELETED",
        obj,
      },
      `WebServer ${obj.metadata.namespace}/${obj.metadata.name} deleted`
    );
  };

  const handleError = (err: any) => {
    logger.error({ error: err }, "Watch error occurred");
    setTimeout(() => {
      logger.info("Reconnecting watch...");
      watchWebServers();
    }, 5000);
  };

  logger.info("Starting WebServer watch...");

  watch.watch(
    watchPath,
    {},
    async (type: string, obj: any) => {
      switch (type) {
        case "ADDED":
          await handleAdded(obj);
          break;
        case "MODIFIED":
          await handleModified(obj);
          break;
        case "DELETED":
          await handleDeleted(obj);
          break;
        default:
          logger.warn({ type, obj }, `Unknown watch event type: ${type}`);
      }
    },
    handleError
  );
}

async function ensureCRDExists() {
  const apiExtensionsApi = kubeConfig.makeApiClient(k8s.ApiextensionsV1Api);

  try {
    await apiExtensionsApi.readCustomResourceDefinition({
      name: SimpleWebServerCrd.metadata.name,
    });
    logger.info("WebServer CRD already exists");
  } catch (error) {
    throw new Error("WebServer CRD does not exist");
  }
}

async function startOperator() {
  try {
    logger.info("Starting WebServer Kubernetes Operator...");

    // Ensure CRD exists
    await ensureCRDExists();

    // Start watching for WebServer resources
    await watchWebServers();

    logger.info("WebServer Operator started successfully");
  } catch (error) {
    logger.error({ error }, "Failed to start WebServer Operator");
    process.exit(1);
  }
}

// Handle graceful shutdown
process.on("SIGINT", () => {
  logger.info("Received SIGINT, shutting down gracefully...");
  process.exit(0);
});

process.on("SIGTERM", () => {
  logger.info("Received SIGTERM, shutting down gracefully...");
  process.exit(0);
});

// Start the operator
startOperator();
