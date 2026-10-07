import { afterEach, beforeEach, describe, expect, test } from "bun:test";
import { createHash } from "node:crypto";
import {
  mkdirSync, mkdtempSync, readFileSync, readdirSync, rmSync, writeFileSync,
} from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

// Exercise the Bash cache helper with tiny local fixtures. No runtime builds
// or network downloads are needed, and no package installation is required.
const archive = "verified archive\n";
let fixtureDir: string;
const libDir = () => join(fixtureDir, "cache/wasmvm/v1.13.0/lib/linux_arm64");
const library = () => join(libDir(), "libwasmvm_muslc.a");
const downloadCount = () =>
  readFileSync(join(fixtureDir, "downloads"), "utf8").trim().split("\n").length;

const runHelper = () => Bun.spawnSync([
  "bash", "-c", `
    set -Eeuo pipefail
    source "$CACHE_HELPER"
    SCRIPT_DIR="$CACHE_FIXTURE/scripts"
    ensure_wasmvm_lib "$CACHE_FIXTURE/cache" linux arm64 v1.13.0
  `,
], {
  env: {
    ...process.env,
    CACHE_HELPER: join(import.meta.dir, "build-nibiru.sh"),
    CACHE_FIXTURE: fixtureDir,
    PATH: `${join(fixtureDir, "bin")}:${process.env.PATH}`,
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

beforeEach(() => {
  fixtureDir = mkdtempSync(join(tmpdir(), "wasmvm-release-cache-"));
  mkdirSync(join(fixtureDir, "scripts"));
  mkdirSync(join(fixtureDir, "bin"));
  writeFileSync(join(fixtureDir, "archive"), archive);
  const digest = createHash("sha256").update(archive).digest("hex");
  writeFileSync(join(fixtureDir, "scripts/wasmvm-checksums.txt"),
    `${digest}  libwasmvm_muslc.aarch64.a\n`);
  // Replace wget on PATH, while still exercising the production shell helper.
  writeFileSync(join(fixtureDir, "bin/wget"), String.raw`#!/usr/bin/env bun
import assert from "node:assert/strict";
import { appendFileSync, copyFileSync } from "node:fs";
import { join } from "node:path";
const args = process.argv.slice(2);
assert.deepEqual(args.slice(0, 2), ["-q", "-O"]);
assert.equal(args.length, 4);
assert.equal(args[3], "https://github.com/NibiruChain/nibiru/releases/download/lib/wasmvm/v1.13.0/libwasmvm_muslc.aarch64.a");
copyFileSync(join(process.env.CACHE_FIXTURE, "archive"), args[2]);
appendFileSync(join(process.env.CACHE_FIXTURE, "downloads"), "download\n");
`, { mode: 0o755 });
});

afterEach(() => rmSync(fixtureDir, { recursive: true, force: true }));

describe("published WasmVM library cache", () => {
  test("downloads and verifies an archive when the cache is empty", () => {
    ensureLibrary();
    expect(readFileSync(library(), "utf8")).toBe(archive);
    expect(downloadCount()).toBe(1);
  });

  test("reuses a verified archive without another download", () => {
    ensureLibrary();
    ensureLibrary();
    expect(readFileSync(library(), "utf8")).toBe(archive);
    expect(downloadCount()).toBe(1);
  });

  test("replaces a corrupt cached archive", () => {
    ensureLibrary();
    writeFileSync(library(), "corrupt cache\n");
    ensureLibrary();
    expect(readFileSync(library(), "utf8")).toBe(archive);
    expect(downloadCount()).toBe(2);
  });

  test("rejects a checksum mismatch and cleans up the download", () => {
    ensureLibrary();
    writeFileSync(library(), "corrupt cache\n");
    writeFileSync(join(fixtureDir, "archive"), "corrupt download\n");
    const result = runHelper();
    expect(result.exitCode).not.toBe(0);
    expect(result.stderr.toString()).toContain("WasmVM checksum mismatch");
    expect(readFileSync(library(), "utf8")).toBe("corrupt cache\n");
    expect(downloadCount()).toBe(2);
    expect(readdirSync(libDir()).filter(name => name.startsWith("download."))).toEqual([]);
  });
});
