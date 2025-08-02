import z from "zod/v4";
import { generateCustomResourceDefination } from "./lib/crd.ts";

export const SimpleWebServerCrd = generateCustomResourceDefination({
  group: "nivekithan.com",
  name: {
    kind: "WebServer",
    plural: "webservers",
    singular: "webserver",
  },
  schema: z.object({ port: z.int().min(1) }),
  scope: "Namespaced",
});
