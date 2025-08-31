import {
  AppsV1Api,
  CustomObjectsApi,
  KubeConfig,
  makeInformer,
  V1Deployment,
} from "@kubernetes/client-node";
import { pgCrd, reconcileCrd, type ReconcileCrdArgs } from "./crd.ts";
import { logger } from "./logger.ts";
import { Effect, Queue } from "effect";
import type { Crd } from "./lib/crd.ts";
import { listAllDeployments } from "./lib/deloyment.ts";

const main = Effect.fn("main")(function* () {
  const kc = new KubeConfig();
  kc.loadFromDefault();

  const pgCrdApi = pgCrd.getApi(kc);
  const reconcileQueue = yield* Queue.unbounded<ReconcileCrdArgs>();

  Effect.runPromise(
    Effect.gen(function* () {
      while (true) {
        logger.debug("Picking up items from the reconcile queue");

        const item = yield* Queue.take(reconcileQueue);
        yield* reconcileCrd(item);
      }
    }),
  );

  const postgresInformer = makeInformer(kc, pgCrd.apiPath(), async () => {
    try {
      const res = await Effect.runPromise(pgCrdApi.listForAllNamesapce);
      return res;
    } catch (err) {
      logger.error({ err });
      throw err;
    }
  });

  postgresInformer.on("add", (event) => {
    if (!event.metadata?.name || !event.metadata?.namespace) return;

    Effect.runFork(
      Queue.offer(reconcileQueue, {
        kc,
        logger,
        name: event.metadata?.name,
        namespace: event.metadata?.namespace,
        pgCrdApi,
      }),
    );
  });

  postgresInformer.on("update", (event) => {
    if (!event.metadata?.name || !event.metadata?.namespace) return;

    Effect.runFork(
      Queue.offer(reconcileQueue, {
        kc,
        logger,
        name: event.metadata?.name,
        namespace: event.metadata?.namespace,
        pgCrdApi,
      }),
    );
  });

  postgresInformer.on("delete", (event) => {
    if (!event.metadata?.name || !event.metadata?.namespace) return;

    Effect.runFork(
      Queue.offer(reconcileQueue, {
        kc,
        logger,
        name: event.metadata?.name,
        namespace: event.metadata?.namespace,
        pgCrdApi,
      }),
    );
  });

  postgresInformer.on("change", (event) => {
    if (!event.metadata?.name || !event.metadata?.namespace) return;
    Effect.runFork(
      Queue.offer(reconcileQueue, {
        kc,
        logger,
        name: event.metadata?.name,
        namespace: event.metadata?.namespace,
        pgCrdApi,
      }),
    );
  });

  yield* Effect.tryPromise(() => postgresInformer.start());

  const deploymentsInformer = makeInformer(
    kc,
    "/apis/apps/v1/deployments",
    async () => {
      try {
        const res = await Effect.runPromise(listAllDeployments({ kc }));
        return res;
      } catch (err) {
        logger.error({ err });
        throw err;
      }
    },
  );

  function identifyControllerManagedDeployment(event: V1Deployment) {
    logger.debug({ event, eventCause: "identifyControllerManagedDeployment" });

    const ownerRef = event.metadata?.ownerReferences?.find(
      (ref) =>
        ref.apiVersion === `${pgCrd.group}/${pgCrd.version}` &&
        ref.kind === pgCrd.kind &&
        ref.controller,
    );

    if (!ownerRef) {
      return;
    }

    const name = ownerRef.name;
    const namespace = event.metadata?.namespace;

    if (!name || !namespace) {
      return;
    }

    Effect.runFork(
      Queue.offer(reconcileQueue, {
        kc,
        logger,
        name: name,
        namespace: namespace,
        pgCrdApi,
      }),
    );
  }

  deploymentsInformer.on("add", identifyControllerManagedDeployment);
  deploymentsInformer.on("update", identifyControllerManagedDeployment);
  deploymentsInformer.on("delete", identifyControllerManagedDeployment);
  deploymentsInformer.on("change", identifyControllerManagedDeployment);

  yield* Effect.tryPromise(() => deploymentsInformer.start());
});

await Effect.runPromise(main());
