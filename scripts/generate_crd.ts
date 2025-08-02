import { writeFile } from "fs/promises";
import { SimpleWebServerCrd } from "../src/simpleWebServer.ts";
import yaml from "js-yaml";

const yamlContent = yaml.dump(SimpleWebServerCrd);

await writeFile("crds/simpleWebServer.yaml", yamlContent);
