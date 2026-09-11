import { $ } from "bun";
import { mkdir, rm } from "node:fs/promises";
import { join } from "node:path";
import { selectedTargets } from "./shared";

const root = join(import.meta.dir, "..");
const dist = join(root, "dist");
await rm(dist, { recursive: true, force: true });
await mkdir(dist, { recursive: true });

for (const packageDir of [...selectedTargets(), "cli"]) {
  await $`bun pm pack --destination ${dist}`.cwd(join(root, "packages", packageDir));
}
