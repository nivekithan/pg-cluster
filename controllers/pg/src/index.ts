import {
  AppsV1Api,
  CustomObjectsApi,
  KubeConfig,
  makeInformer,
} from "@kubernetes/client-node";
import { pgCrd, reconcileCrd } from "./crd.ts";
import { logger } from "./logger.ts";
import { Effect } from "effect";

const kc = new KubeConfig();
kc.loadFromDefault();

const pgCrdApi = pgCrd.getApi(kc);

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

  Effect.runPromise(
    reconcileCrd({
      name: event.metadata?.name,
      namespace: event.metadata?.namespace,
      pgCrdApi,
      kc,
      logger,
    }),
  );
});

podsInformer.on("update", (event) => {
  if (!event.metadata?.name || !event.metadata?.namespace) return;

  Effect.runPromise(
    reconcileCrd({
      name: event.metadata?.name,
      namespace: event.metadata?.namespace,
      pgCrdApi,
      kc,
      logger,
    }),
  );
});

podsInformer.on("delete", (event) => {
  if (!event.metadata?.name || !event.metadata?.namespace) return;

  Effect.runPromise(
    reconcileCrd({
      name: event.metadata?.name,
      namespace: event.metadata?.namespace,
      pgCrdApi,
      kc,
      logger,
    }),
  );
});

podsInformer.on("change", (event) => {
  if (!event.metadata?.name || !event.metadata?.namespace) return;

  Effect.runPromise(
    reconcileCrd({
      name: event.metadata?.name,
      namespace: event.metadata?.namespace,
      pgCrdApi,
      kc,
      logger,
    }),
  );
});

await podsInformer.start();
