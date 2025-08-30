import { V1CustomResourceDefinition } from "@kubernetes/client-node";
import { convertSync } from "@openapi-contrib/json-schema-to-openapi-schema";

const pgCrd: V1CustomResourceDefinition = {
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
          openAPIV3Schema: {},
        },
      },
    ],
  },
};
