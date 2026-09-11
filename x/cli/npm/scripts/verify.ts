import { access, stat } from "node:fs/promises";
import { constants } from "node:fs";
import { join } from "node:path";
import { selectedTargets, targets, version } from "./shared";

const root = join(import.meta.dir, "..");
const releaseVersion = version();
const selected = selectedTargets();
const cli = await Bun.file(join(root, "packages", "cli", "package.json")).json();

if (cli.version !== releaseVersion || !cli.bin?.nibid) throw new Error("umbrella package version or bin is invalid");
for (const [name, dependencyVersion] of Object.entries(cli.optionalDependencies)) {
  if (dependencyVersion !== releaseVersion) throw new Error(`${name} is not lockstep with ${releaseVersion}`);
}
if (Object.keys(cli.optionalDependencies).length !== Object.keys(targets).length) throw new Error("umbrella optionalDependencies is incomplete");

for (const target of selected) {
  const manifest = await Bun.file(join(root, "packages", target, "package.json")).json();
  const expected = targets[target];
  if (manifest.version !== releaseVersion) throw new Error(`${target} is not lockstep with ${releaseVersion}`);
  if (manifest.os?.[0] !== expected.os || manifest.cpu?.[0] !== expected.cpu) throw new Error(`${target} os/cpu metadata is invalid`);
  if (manifest.bin) throw new Error(`${target} must not declare a public bin`);
  const binary = join(root, "packages", target, "bin", "nibid");
  await access(binary, constants.X_OK);
  if ((await stat(binary)).size < 1_000_000) throw new Error(`${target} binary is implausibly small`);
}

console.log(`verified ${selected.join(", ")} for ${releaseVersion}`);
