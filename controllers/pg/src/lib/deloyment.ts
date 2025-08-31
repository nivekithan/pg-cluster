import {
  ApiException,
  AppsV1Api,
  type AppsV1ApiCreateNamespacedDeploymentRequest,
  type AppsV1ApiReadNamespacedDeploymentRequest,
  type KubeConfig,
} from "@kubernetes/client-node";
import { Data, Effect } from "effect";
import { UnknownException } from "effect/Cause";

export const readNamespacedDeployment = Effect.fn("readNamespacedDeployment")(
  function* ({
    kc,
    ...options
  }: { kc: KubeConfig } & AppsV1ApiReadNamespacedDeploymentRequest) {
    const appsApi = kc.makeApiClient(AppsV1Api);

    const checkExistingDeployment = yield* Effect.tryPromise({
      try: async (a) => {
        return appsApi.readNamespacedDeployment(options);
      },

      catch: (err) => {
        if (err instanceof Error && err.message.includes("404")) {
          return new DeploymentNotFoundError({
            name: options.name,
            namespace: options.namespace,
          });
        }

        return new UnknownException(err);
      },
    });

    return checkExistingDeployment;
  },
);

export const createNamespacedDeployment = Effect.fn(
  "createNamespacedDeployment",
)(function* ({
  kc,
  ...options
}: {
  kc: KubeConfig;
} & AppsV1ApiCreateNamespacedDeploymentRequest) {
  const appsApi = kc.makeApiClient(AppsV1Api);

  const createDeployment = yield* Effect.tryPromise({
    try: async (a) => {
      return appsApi.createNamespacedDeployment(options);
    },

    catch: (err) => {
      if (err instanceof ApiException && err.code === 409) {
        return new DeploymentAlreadyExist({
          name: options.body.metadata?.name,
          namespace: options.namespace,
        });
      }

      return new UnknownException(err);
    },
  });

  return createDeployment;
});

export class DeploymentNotFoundError extends Data.TaggedError(
  "DeploymentNotFound",
)<{ name: string; namespace: string }> {}

export class DeploymentAlreadyExist extends Data.TaggedError(
  "DeploymentAlreadyExist",
)<{ name?: string; namespace: string }> {}
