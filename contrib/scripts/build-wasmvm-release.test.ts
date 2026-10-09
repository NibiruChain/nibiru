import { expect, test } from "bun:test";
import { createHash } from "node:crypto";
import {
  mkdirSync,
  mkdtempSync,
  readFileSync,
  readdirSync,
  rmSync,
  writeFileSync,
} from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

test("release cache downloads, reuses, repairs, and rejects corrupt archives", () => {
  const fixtureDir = mkdtempSync(join(tmpdir(), "wasmvm-release-cache-"));
  try {
    const archive = "verified archive\n";
    const libDir = join(fixtureDir, "cache/wasmvm/v1.13.1/lib/linux_arm64");
    const library = join(libDir, "libwasmvm_muslc.a");
    mkdirSync(join(fixtureDir, "scripts"));
    writeFileSync(join(fixtureDir, "archive"), archive);
    const digest = createHash("sha256").update(archive).digest("hex");
    writeFileSync(join(fixtureDir, "scripts/wasmvm-checksums.txt"),
      `${digest}  libwasmvm_muslc.aarch64.a\n`);

    // Only the helper invocation and mock downloader use Bash. All assertions
    // run in Bun, with one fixture and no runtime compilation or network access.
    const runHelper = () => Bun.spawnSync(["bash", "-c", String.raw`
      set -Eeuo pipefail
      source "$CACHE_HELPER"
      SCRIPT_DIR="$CACHE_FIXTURE/scripts"
      wget() {
        cp "$CACHE_FIXTURE/archive" "$3"
        printf '%s\n' "$4" >> "$CACHE_FIXTURE/downloads"
      }
      ensure_wasmvm_lib "$CACHE_FIXTURE/cache" linux arm64 v1.13.1
    `], {
      env: {
        ...process.env,
        CACHE_HELPER: join(import.meta.dir, "build-nibiru.sh"),
        CACHE_FIXTURE: fixtureDir,
      },
      stdout: "pipe",
      stderr: "pipe",
    });
    const ensureLibrary = () => {
      const result = runHelper();
      if (result.exitCode !== 0) {
        throw new Error(`cache helper failed (${result.exitCode}): ${result.stderr.toString()}`);
      }
    };
    const downloads = () =>
      readFileSync(join(fixtureDir, "downloads"), "utf8").trim().split("\n");
    const url = "https://github.com/NibiruChain/nibiru/releases/download/lib/wasmvm/v1.13.1/libwasmvm_muslc.aarch64.a";

    // 1. Download and verify an empty cache.
    ensureLibrary();
    expect(readFileSync(library, "utf8")).toBe(archive);
    expect(downloads()).toEqual([url]);

    // 2. Reuse the verified cache without downloading.
    ensureLibrary();
    expect(readFileSync(library, "utf8")).toBe(archive);
    expect(downloads()).toEqual([url]);

    // 3. Replace a corrupt cached archive.
    writeFileSync(library, "corrupt cache\n");
    ensureLibrary();
    expect(readFileSync(library, "utf8")).toBe(archive);
    expect(downloads()).toEqual([url, url]);

    // 4. Reject a corrupt download without replacing the cache, and clean up.
    writeFileSync(library, "corrupt cache\n");
    writeFileSync(join(fixtureDir, "archive"), "corrupt download\n");
    const result = runHelper();
    expect(result.exitCode).not.toBe(0);
    expect(result.stderr.toString()).toContain("WasmVM checksum mismatch");
    expect(readFileSync(library, "utf8")).toBe("corrupt cache\n");
    expect(downloads()).toEqual([url, url, url]);
    expect(readdirSync(libDir).filter(name => name.startsWith("download."))).toEqual([]);
  } finally {
    rmSync(fixtureDir, { recursive: true, force: true });
  }
});
