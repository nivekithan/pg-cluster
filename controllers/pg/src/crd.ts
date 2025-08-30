import type { V1CustomResourceDefinition } from "@kubernetes/client-node";
import { convertSync } from "@openapi-contrib/json-schema-to-openapi-schema";
import z, { ZodObject } from "zod";

const pgCrdSpec = z.object({
  spec: z.object({
    storage: z.string(),
  }),
});

function geneateOpenapiSchema(spec: ZodObject) {
  const openapiSchema = convertSync(
    z.toJSONSchema(spec, {
      override(spec) {
        delete spec.jsonSchema.additionalProperties;
      },
    }),
  );

  return openapiSchema;
}

export const pgCrd: V1CustomResourceDefinition = {
  apiVersion: "apiextensions.k8s.io/v1",
  kind: "CustomResourceDefinition",
  metadata: {
    name: "postgres.kube.nivekithan.com",
  },
  spec: {
    group: "kube.nivekithan.com",
    names: {
      kind: "Postgres",
      plural: "postgres",
      singular: "postgres",
      shortNames: ["pg"],
      listKind: "PostgresList",
    },
    scope: "Namespaced",
    versions: [
      {
        name: "v1alpha1",
        served: true,
        storage: true,
        schema: {
          openAPIV3Schema: geneateOpenapiSchema(pgCrdSpec),
        },
      },
    ],
  },
};
