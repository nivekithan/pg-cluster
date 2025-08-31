import { pgCrd } from "../src/crd.ts";
import yaml from "yaml";
import { writeFile } from "fs/promises";

const yamlContent = pgCrd.toYaml();

await writeFile("./generated_manifests/crd.yaml", yamlContent);
