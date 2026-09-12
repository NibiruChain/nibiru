import { describe, expect, test } from "bun:test";
import { createHash } from "node:crypto";
import { mkdir, mkdtemp, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import {
  assetsForRelease,
  createProgram,
  normalizeDistributionVersion,
  parseVersion,
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
    expect(program.helpInformation()).toContain("Usage: nibiru-dist [options] [command]");
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
