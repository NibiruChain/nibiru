import { afterEach, describe, expect, spyOn, test } from "bun:test";
import { createHash } from "node:crypto";
import { mkdir, mkdtemp, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import {
  assetsForRelease,
  createProgram,
  normalizeDistributionVersion,
  packageVersionIsUnpublished,
  parseVersion,
  publishPreflight,
  publishRelease,
  type CommandRunner,
  type FetchResult,
  type PreparedRelease,
  TARGETS,
  readCachedReleases,
  selectReleaseTag,
  type Release,
  type ReleaseSummary,
} from "./nibiru-dist";

const summaries: ReleaseSummary[] = [
  { tagName: "v2.18.1", name: "v2.18.1", isDraft: false, url: "https://github.com/NibiruChain/nibiru/releases/tag/v2.18.1" },
  { tagName: "hotfix/v2.19.0", name: "v2.19.0", isDraft: false, url: "https://github.com/NibiruChain/nibiru/releases/tag/hotfix%2Fv2.19.0" },
];

describe("selectReleaseTag", () => {
  test("keeps an exact tag", () => {
    expect(selectReleaseTag("hotfix/v2.19.0", summaries)).toBe("hotfix/v2.19.0");
  });

  test("accepts a GitHub release URL", () => {
    expect(selectReleaseTag("https://github.com/NibiruChain/nibiru/releases/tag/hotfix%2Fv2.19.0", summaries)).toBe("hotfix/v2.19.0");
  });

  test("rejects an ambiguous version", () => {
    expect(() => selectReleaseTag("2.19.0", [...summaries, { ...summaries[0], tagName: "v2.19.0", name: "v2.19.0" }])).toThrow("multiple GitHub releases");
  });
});

describe("assetsForRelease", () => {
  test("selects the four npm platform archives and checksum manifest", () => {
    const release: Release = {
      ...summaries[1],
      id: "release-id",
      isPrerelease: false,
      assets: [
        "checksums.txt",
        "linux_amd64.tar.gz",
        "linux_arm64.tar.gz",
        "darwin_amd64.tar.gz",
        "darwin_arm64.tar.gz",
        "darwin_all.tar.gz",
      ].map((suffix) => ({ name: `nibid_2.19.0_${suffix}` })),
    };
    expect(assetsForRelease(release)).toEqual({
      version: "2.19.0",
      names: [
        "nibid_2.19.0_checksums.txt",
        "nibid_2.19.0_linux_amd64.tar.gz",
        "nibid_2.19.0_linux_arm64.tar.gz",
        "nibid_2.19.0_darwin_amd64.tar.gz",
        "nibid_2.19.0_darwin_arm64.tar.gz",
      ],
    });
  });
});

describe("publish arguments", () => {
  test("uses Commander commands and generated help", () => {
    const program = createProgram();
    expect(program.commands.map((command) => command.name())).toEqual(["list", "get", "prepare", "publish"]);
    expect(program.helpInformation()).toContain("Usage: bun run main.ts [options] [command]");
    expect(program.helpInformation()).toContain("Show locally cached release assets");
  });

  test("allows npm prerelease suffixes that describe one binary version", () => {
    expect(parseVersion("2.19.0")).toEqual({ version: "2.19.0", core: "2.19.0" });
    expect(parseVersion("v2.19.0-rc.1")).toEqual({ version: "2.19.0-rc.1", core: "2.19.0" });
    expect(parseVersion("2.19.0-npm.1")).toEqual({ version: "2.19.0-npm.1", core: "2.19.0" });
  });

  test("rejects npm channel names and missing publish inputs", () => {
    expect(() => normalizeDistributionVersion("latest")).toThrow("--dist-ver");
    expect(() => normalizeDistributionVersion("next")).toThrow("--dist-ver");
    expect(() => normalizeDistributionVersion("2.19.0+build.1")).toThrow("--dist-ver");
  });

  test("accepts both Bun missing-package responses before publishing", () => {
    expect(packageVersionIsUnpublished({ code: 1, stdout: "", stderr: "404 Not Found" })).toBe(true);
    expect(packageVersionIsUnpublished({ code: 1, stdout: "", stderr: "No version of package satisfying 2.1.0 found" })).toBe(true);
    expect(packageVersionIsUnpublished({ code: 1, stdout: "", stderr: "authentication required" })).toBe(false);
  });
});

describe("readCachedReleases", () => {
  test("reads and verifies a local release without GitHub access", async () => {
    const cacheRoot = await mkdtemp(join(tmpdir(), "nibid-cache-"));
    const directory = join(cacheRoot, "hotfix__v2.19.0--release-id");
    const archiveName = "nibid_2.19.0_linux_amd64.tar.gz";
    const checksumName = "nibid_2.19.0_checksums.txt";
    const archive = "native binary bytes";
    await mkdir(directory);
    await writeFile(join(directory, archiveName), archive);
    await writeFile(join(directory, checksumName), `${createHash("sha256").update(archive).digest("hex")}  ${archiveName}\n`);
    await writeFile(join(directory, "release.json"), JSON.stringify({
      id: "release-id",
      tagName: "hotfix/v2.19.0",
      version: "2.19.0",
      assetNames: [checksumName, archiveName],
    }));

    const entries = await readCachedReleases(cacheRoot, true);
    expect(entries).toHaveLength(1);
    expect(entries[0].state).toBe("verified");
    expect(entries[0].totalBytes).toBeGreaterThan(archive.length);
  });
});


describe("publish workflow", () => {
  const temporaryDirectories: string[] = [];
  const logs: string[] = [];
  let restoreLog: (() => void) | undefined;

  afterEach(async () => {
    restoreLog?.();
    restoreLog = undefined;
    logs.length = 0;
    for (const path of temporaryDirectories.splice(0)) await rm(path, { recursive: true, force: true });
  });

  function captureLogs() {
    const log = spyOn(console, "log").mockImplementation((message) => { logs.push(String(message)); });
    restoreLog = () => log.mockRestore();
  }

  async function temporary() {
    const path = await mkdtemp(join(tmpdir(), "clidist-workflow-"));
    temporaryDirectories.push(path);
    return path;
  }

  function runner(calls: string[], override?: CommandRunner): CommandRunner {
    return async (program, args, cwd) => {
      calls.push([program, ...args].join(" "));
      if (override) return override(program, args, cwd);
      if (args.includes("--help")) return { code: 0, stdout: "--access --destination", stderr: "" };
      if (args.includes("view")) return { code: 1, stdout: "", stderr: "404 Not Found" };
      return { code: 0, stdout: "test-user", stderr: "" };
    };
  }

  async function operations() {
    const workspace = await temporary();
    for (const target of [...Object.keys(TARGETS), "cli"]) {
      await mkdir(join(workspace, target));
      await writeFile(join(workspace, target, "package.json"), JSON.stringify({
        name: `@nibiruchain/nibiru${target === "cli" ? "" : `-${target}`}`,
        version: "2.19.0",
      }));
    }
    const fetched: FetchResult = {
      release: { id: "test-release", tagName: "v2.19.0", name: "v2.19.0", isDraft: false, isPrerelease: false, assets: [] },
      version: "2.19.0", directory: workspace, cached: true,
    };
    const prepared: PreparedRelease = { ...fetched, distributionVersion: "2.19.0", workspace, output: workspace, tarballs: [] };
    return { fetched, prepared };
  }

  test("failed authentication stops before registry checks or release work", async () => {
    captureLogs();
    const calls: string[] = [];
    const run = runner(calls, async (_program, args) => args.includes("whoami")
      ? { code: 1, stdout: "", stderr: "secret-auth-output" }
      : { code: 0, stdout: "--access --destination", stderr: "" });
    await expect(publishRelease("v2.19.0", "2.19.0", true, run)).rejects.toThrow("Bun could not verify npm authentication");
    expect(calls.at(-1)).toBe("bun pm whoami");
    expect(calls.some((call) => /release|view|pack --destination|publish --access/.test(call))).toBe(false);
    expect(logs.join("\n")).not.toContain("secret-auth-output");
    expect(logs[0]).toContain("Publish npm version 2.19.0 from GitHub release v2.19.0");
    expect(logs[1]).toContain("[Phase 0/3]");
  });

  test("missing tools and unsupported Bun fail before fetching", async () => {
    captureLogs();
    await expect(publishRelease("v2.19.0", "2.19.0", false,
      runner([], async () => ({ code: 1, stdout: "", stderr: "tool unavailable" })),
    )).rejects.toThrow("tool unavailable");
    await expect(publishPreflight(false,
      runner([], async () => ({ code: 0, stdout: "old help", stderr: "" })),
    )).rejects.toThrow("required publish and pack options");
  });

  test("explains XDG mismatch without reading or printing credentials", async () => {
    captureLogs();
    const home = await temporary();
    const xdg = join(home, "config");
    await writeFile(join(home, ".npmrc"), "SECRET_NOT_FOR_LOGS");
    const run = runner([], async (_program, args) => ({
      code: args.includes("whoami") ? 1 : 0, stdout: "--access --destination", stderr: "",
    }));
    await expect(publishPreflight(true, run, { home, xdg })).rejects.toThrow("env -u XDG_CONFIG_HOME bun pm whoami");
    expect(logs.join("\n")).not.toContain("SECRET_NOT_FOR_LOGS");
    await expect(publishPreflight(true, run, { home: xdg })).rejects.toThrow("bunx npm login");
    await mkdir(xdg);
    await writeFile(join(xdg, ".npmrc"), "OTHER_SECRET");
    await expect(publishPreflight(true, run, { home, xdg })).rejects.toThrow("bunx npm login");
  });

  test("checks collisions before fetching, including in dry runs", async () => {
    captureLogs();
    const calls: string[] = [];
    const run = runner(calls, async () => ({ code: 0, stdout: "--access --destination", stderr: "" }));
    await expect(publishRelease("v2.19.0", "2.19.0", false, run)).rejects.toThrow("is already published");
    expect(calls.at(-1)).toBe("bun pm view @nibiruchain/nibiru-linux-x64@2.19.0 version");
    expect(calls.some((call) => call.startsWith("gh release"))).toBe(false);
  });

  test("registry failures stop before release work instead of being treated as missing versions", async () => {
    captureLogs();
    const calls: string[] = [];
    const run = runner(calls, async (_program, args) => args.includes("view")
      ? { code: 1, stdout: "", stderr: "registry unavailable" }
      : { code: 0, stdout: "--access --destination", stderr: "" });
    await expect(publishRelease("v2.19.0", "2.19.0", false, run)).rejects.toThrow("registry unavailable");
    expect(calls.some((call) => call.startsWith("gh release"))).toBe(false);
  });

  test("an empty identity does not pass authentication", async () => {
    captureLogs();
    const run = runner([], async (_program, args) => ({
      code: 0, stdout: args.includes("whoami") ? "" : "--access --destination", stderr: "",
    }));
    await expect(publishPreflight(true, run, { home: await temporary() })).rejects.toThrow("Bun could not verify npm authentication");
  });

  test("mismatched binary versions never reach preparation or publication", async () => {
    captureLogs();
    const { fetched } = await operations();
    await expect(publishRelease("v2.19.0", "2.19.1", false, runner([]), {
      fetch: async () => fetched,
      prepare: async () => { throw new Error("unexpected preparation"); },
      publish: async () => { throw new Error("unexpected upload"); },
    })).rejects.toThrow("does not match --dist-ver 2.19.1");
  });

  test("dry run checks five versions once and never authenticates or uploads", async () => {
    captureLogs();
    const calls: string[] = [];
    const { fetched, prepared } = await operations();
    await publishRelease("v2.19.0", "v2.19.0", false, runner(calls), {
      fetch: async () => { calls.push("fetch"); return fetched; },
      prepare: async () => { calls.push("prepare"); return prepared; },
      publish: async () => { throw new Error("unexpected upload"); },
    });
    expect(calls.filter((call) => call.startsWith("bun pm view"))).toHaveLength(5);
    expect(calls.indexOf("fetch")).toBeGreaterThan(calls.findLastIndex((call) => call.startsWith("bun pm view")));
    expect(calls.at(-1)).toBe("prepare");
    expect(calls).not.toContain("bun pm whoami");
    expect(logs.join("\n")).toContain("[Phase 3/3] Previewing npm publish commands; nothing will be uploaded");
    expect(logs.filter((log) => log.startsWith("[Dry run]"))).toHaveLength(5);
    expect(logs.at(-1)).toContain("/cli && bun publish --access public");
  });

  test("successful uploads preserve platform order and interactive publisher arguments", async () => {
    captureLogs();
    const calls: string[] = [];
    const { fetched, prepared } = await operations();
    const uploads: string[] = [];
    await publishRelease("v2.19.0", "2.19.0", true, runner(calls), {
      fetch: async () => fetched, prepare: async () => prepared,
      publish: async (directory, args) => {
        uploads.push(directory.split("/").at(-1)!);
        expect(args).toEqual(["publish", "--access", "public"]);
      },
    });
    expect(uploads).toEqual([...Object.keys(TARGETS), "cli"]);
    expect(calls.filter((call) => call.startsWith("bun pm view"))).toHaveLength(5);
    expect(logs.join("\n")).toContain("[Phase 3/3] Publishing all npm packages with bun publish");
  });

  test("upload failure reports partial publication and stops later packages", async () => {
    captureLogs();
    const { fetched, prepared } = await operations();
    const uploads: string[] = [];
    const result = publishRelease("v2.19.0", "2.19.0", true, runner([]), {
      fetch: async () => fetched, prepare: async () => prepared,
      publish: async (directory) => {
        uploads.push(directory.split("/").at(-1)!);
        if (uploads.length === 2) throw new Error("bun publish failed with exit code 1");
      },
    });
    await expect(result).rejects.toThrow("Failed to publish @nibiruchain/nibiru-linux-arm64@2.19.0: bun publish failed with exit code 1");
    await expect(result).rejects.toThrow("Successfully published packages: @nibiruchain/nibiru-linux-x64@2.19.0");
    expect(uploads).toEqual(["linux-x64", "linux-arm64"]);
  });
});
