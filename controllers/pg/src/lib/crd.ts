import { CustomObjectsApi, type KubeConfig } from "@kubernetes/client-node";
import { convertSync } from "@openapi-contrib/json-schema-to-openapi-schema";
import { Effect } from "effect";
import { EffectPrototype } from "effect/Effectable";
import { func } from "effect/FastCheck";
import type { Logger } from "pino";
import yaml from "yaml";
import { z, type ZodObject } from "zod";

export class Crd<T extends ZodObject> {
  readonly group: string;
  readonly kind: string;
  readonly spec: T;
  readonly version = "v1alpha1";
  #logger: Logger;

  constructor({
    group,
    kind,
    spec,
    logger,
  }: {
    group: string;
    kind: string;
    spec: T;
    logger: Logger;
  }) {
    this.group = group;
    this.kind = kind;
    this.spec = spec;
    this.#logger = logger.child({ group, kind, class: "Crd" });
  }

  getApi(kc: KubeConfig) {
    const customObjectsApi = kc.makeApiClient(CustomObjectsApi);

    return {
      listForAllNamesapce: Effect.gen(this, function* () {
        const res = yield* Effect.tryPromise(async () => {
          return customObjectsApi.listCustomObjectForAllNamespaces({
            group: this.group,
            plural: this.#names().plural,
            version: this.version,
          });
        });

        this.#logger.debug({ query: "listForAllNamesapce", res });

        const finalResult = yield* Effect.try(() => {
          return z
            .looseObject({ items: z.array(this.#getCrSchema()) })
            .parse(res);
        });

        this.#logger.info({ query: "listForAllNamesapce", finalResult });

        return finalResult;
      }),

      getNamespacedObject: ({
        name,
        namespace,
      }: {
        name: string;
        namespace: string;
      }) => {
        return Effect.gen(this, function* () {
          const res = yield* Effect.tryPromise(() =>
            customObjectsApi.getNamespacedCustomObject({
              group: this.group,
              version: this.version,
              namespace: namespace,
              plural: this.kind.toLowerCase(),
              name: name,
            }),
          );

          this.#logger.debug({ query: "getNamespacedObject", res });

          const finalResult = yield* Effect.try(() =>
            this.#getCrSchema().parse(res),
          );

          return finalResult;
        });
      },
    };
  }

  toJson() {
    return {
      apiVersion: "apiextensions.k8s.io/v1",
      kind: "CustomResourceDefinition",
      metadata: {
        name: `${this.group}.${this.kind}`,
      },
      spec: {
        group: this.group,
        names: {
          kind: this.kind,
          plural: this.kind.toLowerCase(),
          singular: this.kind.toLowerCase(),
          listKind: `${this.kind}List`,
        },
        scope: "Namespaced",
        versions: [
          {
            name: this.version,
            served: true,
            storage: true,
            schema: {
              openAPIV3Schema: geneateOpenapiSchema(this.spec),
            },
          },
        ],
      },
    };
  }

  toYaml() {
    const jsonValue = this.toJson();

    return yaml.stringify(jsonValue);
  }

  apiPath() {
    return `/apis/${this.group}/${this.version}/${this.#names().singular}`;
  }

  #names() {
    return {
      kind: this.kind,
      plural: this.kind.toLowerCase(),
      singular: this.kind.toLowerCase(),
      listKind: `${this.kind}List`,
    };
  }

  #getCrSchema() {
    return z
      .looseObject({
        apiVersion: z.literal(`${this.group}/${this.version}`),
        kind: z.literal(this.kind),
        metadata: z.looseObject({
          annotations: z.record(z.string(), z.string()).optional(),
          name: z.string(),
          namespace: z.string(),
          resourceVersion: z.string(),
          uid: z.string(),
        }),
      })
      .merge(this.spec);
  }
}

function geneateOpenapiSchema<T extends ZodObject>(spec: T) {
  const openapiSchema = convertSync(
    z.toJSONSchema(spec, {
      override(spec) {
        delete spec.jsonSchema.additionalProperties;
      },
    }),
  );

  return openapiSchema;
}
