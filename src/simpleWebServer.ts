import z from "zod/v4";
import { generateCustomResourceDefination } from "./lib/crd";

export const SimpleWebServerCrd = await generateCustomResourceDefination({
  group: "nivekithan.com",
  name: {
    kind: "WebServer",
    plural: "webservers",
    singular: "webserver",
  },
  schema: z.object({ port: z.int() }),
  scope: "Namespaced",
});
