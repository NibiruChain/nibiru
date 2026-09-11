import { $ } from "bun";
import { chmod, cp, mkdtemp, mkdir, readdir, rm, writeFile } from "node:fs/promises";
import { join } from "node:path";
import { tmpdir } from "node:os";
import { selectedTargets, targets, version } from "./shared";

const root = join(import.meta.dir, "..");
const artifactDir = process.env.NIBID_ARTIFACT_DIR ?? join(root, "artifacts");
const releaseVersion = version();
const selected = selectedTargets();

for (const target of selected) {
  const artifact = join(artifactDir, `nibid_${releaseVersion}_${targets[target].artifact}.tar.gz`);
  const stagingDir = await mkdtemp(join(tmpdir(), "nibid-npm-"));
  try {
    await $`tar -xzf ${artifact} -C ${stagingDir}`.quiet();
    const entries = await readdir(stagingDir, { recursive: true });
    const binaryPath = entries.map(String).find((entry) => entry === "nibid");
    if (!binaryPath) throw new Error(`${artifact} does not contain nibid`);

    const destination = join(root, "packages", target, "bin", "nibid");
    await mkdir(join(root, "packages", target, "bin"), { recursive: true });
    await cp(join(stagingDir, binaryPath), destination);
    await chmod(destination, 0o755);
  } finally {
    await rm(stagingDir, { recursive: true, force: true });
  }
}

for (const packageDir of ["cli", ...Object.keys(targets)]) {
  const manifestPath = join(root, "packages", packageDir, "package.json");
  const manifest = await Bun.file(manifestPath).json();
  manifest.version = releaseVersion;
  if (packageDir === "cli") {
    for (const packageName of Object.keys(manifest.optionalDependencies)) {
      manifest.optionalDependencies[packageName] = releaseVersion;
    }
  }
  await writeFile(manifestPath, `${JSON.stringify(manifest, null, 2)}\n`);
}
