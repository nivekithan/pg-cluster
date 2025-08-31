import {
  AppsV1Api,
  CustomObjectsApi,
  KubeConfig,
  makeInformer,
} from "@kubernetes/client-node";
import { pgCrd, reconcileCrd, type ReconcileCrdArgs } from "./crd.ts";
import { logger } from "./logger.ts";
import { Effect, Queue } from "effect";
import type { Crd } from "./lib/crd.ts";

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

  const podsInformer = makeInformer(kc, pgCrd.apiPath(), async () => {
    try {
      const res = await Effect.runPromise(pgCrdApi.listForAllNamesapce);
      return res;
    } catch (err) {
      logger.error({ err });
      throw err;
    }
  });

  podsInformer.on("add", (event) => {
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

  podsInformer.on("update", (event) => {
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

  podsInformer.on("delete", (event) => {
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

  podsInformer.on("change", (event) => {
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

  yield* Effect.tryPromise(() => podsInformer.start());
});

await Effect.runPromise(main());
