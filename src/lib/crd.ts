import { logger } from "../logger.ts";
import { convert } from "@openapi-contrib/json-schema-to-openapi-schema";
import { z } from "zod/v4";
import type * as k8s from "@kubernetes/client-node";

export async function generateCustomResourceDefination<T extends z.ZodObject>({
  group,
  name,
  scope,
  schema,
}: {
  group: string;
  name: {
    plural: string;
    singular: string;
    kind: string;
    shortNames?: Array<string>;
  };
  scope: "Namespaced" | "Cluster";
  schema: T;
}): Promise<k8s.V1CustomResourceDefinition> {
  const jsonSchema = z.toJSONSchema(schema, { target: "draft-4" });

  logger.debug({ type: "jsonSchema", jsonSchema });

  const openApiSchema = await convert(jsonSchema, { dereference: false });

  logger.debug({ type: "openApiSchema", openApiSchema });

  return {
    apiVersion: "apiextensions.k8s.io/v1",
    kind: "CustomResourceDefinition",
    metadata: {
      name: `${name.plural}.${group}`,
    },
    spec: {
      group: group,
      scope: scope,
      names: {
        plural: name.plural,
        singular: name.singular,
        kind: name.kind,
        shortNames: name.shortNames ?? [],
      },
      versions: [
        {
          name: "v1",
          served: true,
          storage: true,
          schema: {
            openAPIV3Schema: openApiSchema,
          },
        },
      ],
    },
  };
}
