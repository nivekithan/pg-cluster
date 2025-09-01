import {
  ApiException,
  CoreV1Api,
  type CoreV1ApiCreateNamespacedPersistentVolumeClaimRequest,
  type KubeConfig,
} from "@kubernetes/client-node";
import { Data, Effect } from "effect";
import { UnknownException } from "effect/Cause";

export const createNamespacedPvc = Effect.fn("createNamespacedPvc")(function* ({
  kc,
  ...options
}: {
  kc: KubeConfig;
} & CoreV1ApiCreateNamespacedPersistentVolumeClaimRequest) {
  const coreApi = kc.makeApiClient(CoreV1Api);

  const pvc = yield* Effect.tryPromise({
    try: async () => coreApi.createNamespacedPersistentVolumeClaim(options),
    catch: (err) => {
      if (err instanceof ApiException && err.code === 409) {
        return new PvcAlreadyExistsError({
          namespace: options.namespace,
          name: options.body.metadata?.name,
        });
      }

      return new UnknownException(err);
    },
  });

  return pvc;
});

export const readNamespacedPvc = Effect.fn("readNamespacedPvc")(function* ({
  kc,
  name,
  namespace,
}: {
  kc: KubeConfig;
  name: string;
  namespace: string;
}) {
  const coreApi = kc.makeApiClient(CoreV1Api);

  const pvc = yield* Effect.tryPromise({
    try: async () =>
      coreApi.readNamespacedPersistentVolumeClaim({ name: name, namespace }),
    catch: (err) => {
      if (err instanceof ApiException && err.code === 404) {
        return new PvcNotFoundError({ name, namespace });
      }

      return new UnknownException(err);
    },
  });

  return pvc;
});

export class PvcNotFoundError extends Data.TaggedError("pvcNotFound")<{
  name: string;
  namespace: string;
}> {}

export class PvcAlreadyExistsError extends Data.TaggedError(
  "pvcAlreadyExists",
)<{
  name?: string;
  namespace: string;
}> {}
