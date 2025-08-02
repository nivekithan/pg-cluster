import * as k8s from "@kubernetes/client-node";
import { generateCustomResourceDefination } from "./lib/crd.ts";
import z from "zod/v4";

const kubeConfig = new k8s.KubeConfig();
kubeConfig.loadFromDefault();

generateCustomResourceDefination({
  group: "nivekithan.com",
  name: {
    kind: "Example",
    plural: "examples",
    singular: "example",
  },
  schema: z.object({ port: z.int() }),
  scope: "Namespaced",
});
