import z, { ZodObject } from "zod";
import { Crd } from "./lib/crd.ts";
import { logger } from "./logger.ts";

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
