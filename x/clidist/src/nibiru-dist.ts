import { createHash } from "node:crypto";
import {
  access,
  chmod,
  cp,
  mkdir,
  mkdtemp,
  readdir,
  readFile,
  rename,
  rm,
  stat,
  writeFile,
} from "node:fs/promises";
import { constants } from "node:fs";
import { tmpdir } from "node:os";
import { basename, join } from "node:path";
import { Command } from "commander";

export const GITHUB_REPOSITORY = "NibiruChain/nibiru";

/**
 * npm needs one leaf package per supported operating-system and CPU pair. The
 * release also has a universal macOS archive, but it is not an npm target: npm
 * selects packages through this `os` and `cpu` metadata before installation.
 */
export const TARGETS = {
  "linux-x64": { os: "linux", cpu: "x64", artifact: "linux_amd64" },
  "linux-arm64": { os: "linux", cpu: "arm64", artifact: "linux_arm64" },
  "darwin-x64": { os: "darwin", cpu: "x64", artifact: "darwin_amd64" },
  "darwin-arm64": { os: "darwin", cpu: "arm64", artifact: "darwin_arm64" },
} as const;

export type Target = keyof typeof TARGETS;
export type CommandResult = { code: number; stdout: string; stderr: string };
export type CommandRunner = (program: string, args: string[], cwd?: string) => Promise<CommandResult>;

export interface ReleaseSummary {
  tagName: string;
  name: string;
  isDraft: boolean;
  url?: string;
}

export interface ReleaseAsset {
  name: string;
}

export interface Release extends ReleaseSummary {
  id: string;
  isPrerelease: boolean;
  assets: ReleaseAsset[];
}

export interface CachedRelease {
  id: string;
  tagName: string;
  version: string;
  assetNames: string[];
}

export interface ParsedVersion {
  version: string;
  core: string;
}

const root = join(import.meta.dir, "..");
const artifactsRoot = join(root, "artifacts", "releases");
const distRoot = join(root, "dist");
const templatePackages = join(root, "packages");
const repositoryRoot = join(root, "..", "..");
const licensePath = join(repositoryRoot, "LICENSE.md");
const packageRepository = "git+https://github.com/NibiruChain/nibiru.git";
const packageHomepage = "https://nibiru.fi/docs/dev/cli";
const packageBugsUrl = "https://github.com/NibiruChain/nibiru/issues";

export const runCommand: CommandRunner = async (program, args, cwd) => {
  const child = Bun.spawn([program, ...args], {
    cwd,
    stdout: "pipe",
    stderr: "pipe",
  });
  const [code, stdout, stderr] = await Promise.all([
    child.exited,
    new Response(child.stdout).text(),
    new Response(child.stderr).text(),
  ]);
  return { code, stdout, stderr };
};

/**
 * Publishes through the operator's terminal instead of captured pipes. npm may
 * require a browser confirmation or one-time password for a write. Capturing
 * that prompt made an interactive publish appear frozen and left no way to
 * complete the required 2FA challenge.
 */
async function publishInteractively(directory: string, args: string[]): Promise<void> {
  const child = Bun.spawn(["bun", ...args], {
    cwd: directory,
    stdin: "inherit",
    stdout: "inherit",
    stderr: "inherit",
  });
  const code = await child.exited;
  if (code !== 0) throw new Error(`bun ${args.join(" ")} failed with exit code ${code}`);
}

function cacheKey(release: Pick<Release, "id" | "tagName">): string {
  // Tags can be reused by people and release names can collide. The GitHub
  // release ID identifies the immutable release whose checksums we verified.
  const id = Buffer.from(release.id).toString("base64url");
  return `${release.tagName.replaceAll("/", "__")}--${id}`;
}

function failCommand(program: string, args: string[], result: CommandResult): never {
  throw new Error(
    `${program} ${args.join(" ")} failed with exit code ${result.code}: ${
      result.stderr.trim() || result.stdout.trim()
    }`,
  );
}

async function mustRun(
  runner: CommandRunner,
  program: string,
  args: string[],
  cwd?: string,
): Promise<string> {
  const result = await runner(program, args, cwd);
  if (result.code !== 0) failCommand(program, args, result);
  return result.stdout;
}

function releaseTagFromUrl(input: string): string | undefined {
  try {
    const url = new URL(input);
    const prefix = `/${GITHUB_REPOSITORY}/releases/tag/`;
    if (url.hostname !== "github.com" || !url.pathname.startsWith(prefix)) return undefined;
    return decodeURIComponent(url.pathname.slice(prefix.length));
  } catch {
    return undefined;
  }
}

function parseReleaseSummaries(output: string): ReleaseSummary[] {
  const releases = JSON.parse(output) as ReleaseSummary[];
  if (!Array.isArray(releases)) throw new Error("GitHub release list was not an array");
  return releases;
}

export function selectReleaseTag(input: string, releases: ReleaseSummary[]): string {
  const value = input.trim();
  const fromUrl = releaseTagFromUrl(value);
  if (value.startsWith("http") && !fromUrl) {
    throw new Error(`Expected a ${GITHUB_REPOSITORY} GitHub release URL`);
  }
  const requested = fromUrl ?? value;
  const published = releases.filter((release) => !release.isDraft);
  // Prefer an exact tag before treating input as a version. A hotfix tag such
  // as `hotfix/v2.19.0` can share the visible release version with `v2.19.0`.
  const exactTag = published.filter((release) => release.tagName === requested);
  if (exactTag.length === 1) return exactTag[0].tagName;
  if (exactTag.length > 1) throw new Error(`GitHub returned duplicate releases for tag ${requested}`);

  const normalized = requested.replace(/^v/, "");
  const matches = published.filter((release) =>
    [release.name, release.tagName]
      .filter(Boolean)
      .map((value) => value.replace(/^v/, ""))
      .includes(normalized),
  );
  if (matches.length === 1) return matches[0].tagName;
  if (matches.length === 0) throw new Error(`No GitHub release matches ${requested}`);
  throw new Error(
    `${requested} matches multiple GitHub releases (${matches
      .map((release) => release.tagName)
      .join(", ")}). Use an exact tag or release URL.`,
  );
}

export async function resolveRelease(
  input: string,
  runner: CommandRunner = runCommand,
): Promise<Release> {
  console.log(`resolving GitHub release ${input}`);
  // GitHub tags and visible release versions are different namespaces here.
  // Resolve a friendly version only when it maps to one release. Otherwise a
  // human must choose the source explicitly with a tag or release URL.
  const list = await mustRun(runner, "gh", [
    "release",
    "list",
    "--repo",
    GITHUB_REPOSITORY,
    "--limit",
    "1000",
    "--json",
    "tagName,name,isDraft",
  ]);
  const tag = selectReleaseTag(input, parseReleaseSummaries(list));
  const output = await mustRun(runner, "gh", [
    "release",
    "view",
    tag,
    "--repo",
    GITHUB_REPOSITORY,
    "--json",
    "id,tagName,name,isDraft,isPrerelease,url,assets",
  ]);
  const release = JSON.parse(output) as Release;
  if (release.isDraft) throw new Error(`Refusing to use draft release ${release.tagName}`);
  return release;
}

export function assetsForRelease(release: Release): { version: string; names: string[] } {
  const targetSuffixes = Object.values(TARGETS).map((target) => target.artifact);
  const versions = new Map<string, Set<string>>();
  for (const asset of release.assets) {
    const match = asset.name.match(/^nibid_(.+)_(linux_amd64|linux_arm64|darwin_amd64|darwin_arm64)\.tar\.gz$/);
    if (!match || !targetSuffixes.includes(match[2])) continue;
    const names = versions.get(match[1]) ?? new Set<string>();
    names.add(match[2]);
    versions.set(match[1], names);
  }
  // A partial release cannot produce a portable npm package. Do not quietly
  // publish only the platform that happened to be available on this machine.
  const usable = [...versions.entries()].filter(([, suffixes]) => targetSuffixes.every((suffix) => suffixes.has(suffix)));
  if (usable.length !== 1) {
    throw new Error(`Release ${release.tagName} must contain one complete set of npm platform archives`);
  }
  const [version] = usable[0];
  if (!/^\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?$/.test(version)) {
    throw new Error(`Release asset version ${version} is not valid npm semver`);
  }
  const names = [
    `nibid_${version}_checksums.txt`,
    ...targetSuffixes.map((suffix) => `nibid_${version}_${suffix}.tar.gz`),
  ];
  const available = new Set(release.assets.map((asset) => asset.name));
  for (const name of names) {
    if (!available.has(name)) throw new Error(`Release ${release.tagName} is missing ${name}`);
  }
  return { version, names };
}

function parseChecksums(content: string): Map<string, string> {
  const checksums = new Map<string, string>();
  for (const line of content.split("\n")) {
    const match = line.trim().match(/^([a-fA-F0-9]{64})\s+\*?(.+)$/);
    if (match) checksums.set(basename(match[2]), match[1].toLowerCase());
  }
  return checksums;
}

async function verifyFiles(directory: string, assetNames: string[]): Promise<void> {
  // The release checksum file is the authority for downloaded archives. This
  // check applies equally to fresh downloads and local cache entries.
  const checksumName = assetNames.find((name) => name.endsWith("_checksums.txt"));
  if (!checksumName) throw new Error("Missing checksum manifest name");
  const checksums = parseChecksums(await readFile(join(directory, checksumName), "utf8"));
  for (const name of assetNames.filter((name) => name.endsWith(".tar.gz"))) {
    const expected = checksums.get(name);
    if (!expected) throw new Error(`${checksumName} has no checksum for ${name}`);
    const actual = createHash("sha256").update(await readFile(join(directory, name))).digest("hex");
    if (actual !== expected) throw new Error(`Checksum mismatch for ${name}`);
  }
}

async function readValidCache(directory: string, release: Release, version: string, names: string[]): Promise<boolean> {
  try {
    const marker = JSON.parse(await readFile(join(directory, "release.json"), "utf8")) as CachedRelease;
    if (marker.id !== release.id || marker.tagName !== release.tagName || marker.version !== version) return false;
    if (JSON.stringify(marker.assetNames) !== JSON.stringify(names)) return false;
    // A cache hit avoids downloading hundreds of megabytes. It does not avoid
    // checksum verification, because local artifact directories are mutable.
    await verifyFiles(directory, names);
    return true;
  } catch {
    return false;
  }
}

export interface FetchResult {
  release: Release;
  version: string;
  directory: string;
  cached: boolean;
}

export interface CachedArchive {
  name: string;
  path: string;
  size: number | undefined;
  present: boolean;
}

export interface CachedReleaseEntry {
  directory: string;
  release: CachedRelease | undefined;
  archives: CachedArchive[];
  totalBytes: number;
  state: "cached" | "incomplete" | "malformed" | "verified" | "invalid";
  error: string | undefined;
}

/**
 * Reads cache directories without GitHub access. This is intentionally separate
 * from `getArtifacts`: operators need to inspect what is on disk when offline,
 * while release operations need current GitHub metadata before trusting a cache.
 */
export async function readCachedReleases(
  cacheRoot = artifactsRoot,
  verify = false,
): Promise<CachedReleaseEntry[]> {
  let directories: Awaited<ReturnType<typeof readdir>>;
  try {
    directories = await readdir(cacheRoot, { withFileTypes: true });
  } catch (error) {
    if ((error as NodeJS.ErrnoException).code === "ENOENT") return [];
    throw error;
  }

  const entries: CachedReleaseEntry[] = [];
  for (const entry of directories) {
    // A process killed during download can leave its private staging directory.
    // It was never promoted to a cache entry and should not worry an operator.
    if (!entry.isDirectory() || entry.name.startsWith(".download-")) continue;
    const directory = join(cacheRoot, entry.name);
    let release: CachedRelease;
    try {
      release = JSON.parse(await readFile(join(directory, "release.json"), "utf8")) as CachedRelease;
      if (!release.id || !release.tagName || !release.version || !Array.isArray(release.assetNames)) {
        throw new Error("release.json is missing required release metadata");
      }
    } catch (error) {
      entries.push({
        directory,
        release: undefined,
        archives: [],
        totalBytes: 0,
        state: "malformed",
        error: error instanceof Error ? error.message : String(error),
      });
      continue;
    }

    const archives = await Promise.all(
      release.assetNames.map(async (name): Promise<CachedArchive> => {
        const path = join(directory, name);
        try {
          return { name, path, size: (await stat(path)).size, present: true };
        } catch {
          return { name, path, size: undefined, present: false };
        }
      }),
    );
    const totalBytes = archives.reduce((total, archive) => total + (archive.size ?? 0), 0);
    const missing = archives.some((archive) => !archive.present);
    let state: CachedReleaseEntry["state"] = missing ? "incomplete" : "cached";
    let error: string | undefined;
    if (verify && !missing) {
      try {
        await verifyFiles(directory, release.assetNames);
        state = "verified";
      } catch (reason) {
        state = "invalid";
        error = reason instanceof Error ? reason.message : String(reason);
      }
    }
    entries.push({ directory, release, archives, totalBytes, state, error });
  }
  return entries.sort((left, right) => (left.release?.tagName ?? left.directory).localeCompare(right.release?.tagName ?? right.directory));
}

function formatBytes(bytes: number): string {
  if (bytes < 1_000) return `${bytes} B`;
  if (bytes < 1_000_000) return `${(bytes / 1_000).toFixed(1)} kB`;
  return `${(bytes / 1_000_000).toFixed(1)} MB`;
}

export function renderCachedReleases(entries: CachedReleaseEntry[], long = false): string {
  if (entries.length === 0) return "No cached release assets.\n";
  const lines = ["SOURCE\tBINARY\tASSETS\tSIZE\tSTATE\tCACHE"];
  for (const entry of entries) {
    const binaryArchives = entry.archives.filter((archive) => archive.name.endsWith(".tar.gz"));
    const present = binaryArchives.filter((archive) => archive.present).length;
    lines.push([
      entry.release?.tagName ?? "<malformed>",
      entry.release?.version ?? "-",
      `${present}/${binaryArchives.length}`,
      formatBytes(entry.totalBytes),
      entry.state,
      entry.directory,
    ].join("\t"));
    if (long) {
      for (const archive of entry.archives) {
        lines.push(`  ${archive.present ? "present" : "missing"}\t${archive.size === undefined ? "-" : formatBytes(archive.size)}\t${archive.name}`);
      }
      if (entry.error) lines.push(`  error\t${entry.error}`);
    }
  }
  return `${lines.join("\n")}\n`;
}

export async function getArtifacts(input: string, runner: CommandRunner = runCommand): Promise<FetchResult> {
  const release = await resolveRelease(input, runner);
  const { version, names } = assetsForRelease(release);
  const directory = join(artifactsRoot, cacheKey(release));
  console.log(`checking cached assets for ${release.tagName}`);
  if (await readValidCache(directory, release, version, names)) {
    console.log(`using verified cache for ${release.tagName}`);
    return { release, version, directory, cached: true };
  }

  await mkdir(artifactsRoot, { recursive: true });
  const temporary = await mkdtemp(join(artifactsRoot, ".download-"));
  try {
    for (const name of names) {
      console.log(`downloading ${name}`);
      await mustRun(runner, "gh", [
        "release",
        "download",
        release.tagName,
        "--repo",
        GITHUB_REPOSITORY,
        "--dir",
        temporary,
        "--pattern",
        name,
      ]);
    }
    console.log(`verifying checksums for ${release.tagName}`);
    await verifyFiles(temporary, names);
    await writeFile(
      join(temporary, "release.json"),
      `${JSON.stringify({ id: release.id, tagName: release.tagName, version, assetNames: names }, null, 2)}\n`,
    );
    // Replace the cache only after every archive passes verification. A failed
    // download must not destroy a previously valid local release.
    await rm(directory, { recursive: true, force: true });
    await rename(temporary, directory);
  } catch (error) {
    await rm(temporary, { recursive: true, force: true });
    throw error;
  }
  return { release, version, directory, cached: false };
}

async function findBinary(directory: string): Promise<string> {
  const matches: string[] = [];
  async function walk(path: string): Promise<void> {
    for (const entry of await readdir(path, { withFileTypes: true })) {
      const child = join(path, entry.name);
      if (entry.isDirectory()) await walk(child);
      else if (entry.isFile() && entry.name === "nibid") matches.push(child);
    }
  }
  await walk(directory);
  if (matches.length !== 1) throw new Error(`Expected one nibid binary in ${directory}, found ${matches.length}`);
  return matches[0];
}

async function stageBinary(archive: string, destination: string, runner: CommandRunner): Promise<void> {
  const temporary = await mkdtemp(join(tmpdir(), "nibid-npm-"));
  try {
    await mustRun(runner, "tar", ["-xzf", archive, "-C", temporary]);
    // Archives passed checksum verification can still have an unexpected file
    // layout. Require exactly one executable payload before packaging it.
    const binary = await findBinary(temporary);
    await mkdir(join(destination, ".."), { recursive: true });
    await cp(binary, destination);
    await chmod(destination, 0o755);
  } finally {
    await rm(temporary, { recursive: true, force: true });
  }
}

async function setPackageVersions(workspace: string, version: string): Promise<void> {
  for (const packageDir of ["cli", ...Object.keys(TARGETS)]) {
    const path = join(workspace, packageDir, "package.json");
    const manifest = await Bun.file(path).json();
    manifest.version = version;
    if (packageDir === "cli") {
      // Optional dependencies let npm choose one native package by platform.
      // Every leaf must use the umbrella package version or users can install
      // a launcher whose requested binary package does not exist.
      for (const name of Object.keys(manifest.optionalDependencies)) {
        manifest.optionalDependencies[name] = version;
      }
    }
    await writeFile(path, `${JSON.stringify(manifest, null, 2)}\n`);
  }
}

async function addPackageLicenses(workspace: string): Promise<void> {
  for (const packageDir of ["cli", ...Object.keys(TARGETS)]) {
    await cp(licensePath, join(workspace, packageDir, "LICENSE.md"));
  }
}

async function verifyPackageDocuments(directory: string, manifest: Record<string, unknown>): Promise<void> {
  if (manifest.license !== "BSD-2-Clause") throw new Error(`${directory} package license is invalid`);
  const repository = manifest.repository as { url?: string } | undefined;
  if (repository?.url !== packageRepository || manifest.homepage !== packageHomepage) {
    throw new Error(`${directory} package repository metadata is invalid`);
  }
  const bugs = manifest.bugs as { url?: string } | undefined;
  if (bugs?.url !== packageBugsUrl) throw new Error(`${directory} package issue tracker is invalid`);
  await access(join(directory, "README.md"), constants.R_OK);
  await access(join(directory, "LICENSE.md"), constants.R_OK);
}

export async function verifyWorkspace(workspace: string, version: string): Promise<void> {
  const cliDirectory = join(workspace, "cli");
  const cli = await Bun.file(join(cliDirectory, "package.json")).json();
  if (cli.version !== version || !cli.bin?.nibiru) throw new Error("Umbrella package version or bin is invalid");
  await verifyPackageDocuments(cliDirectory, cli);
  if (Object.keys(cli.optionalDependencies).length !== Object.keys(TARGETS).length) {
    throw new Error("Umbrella optionalDependencies is incomplete");
  }
  for (const [name, dependencyVersion] of Object.entries(cli.optionalDependencies)) {
    if (dependencyVersion !== version) throw new Error(`${name} is not lockstep with ${version}`);
  }
  for (const [target, expected] of Object.entries(TARGETS)) {
    const directory = join(workspace, target);
    const manifest = await Bun.file(join(directory, "package.json")).json();
    if (manifest.version !== version || manifest.os?.[0] !== expected.os || manifest.cpu?.[0] !== expected.cpu) {
      throw new Error(`${target} package metadata is invalid`);
    }
    await verifyPackageDocuments(directory, manifest);
    // The umbrella package owns the public `nibiru` command. Leaf packages only
    // provide native files, so their metadata cannot replace the launcher.
    if (manifest.bin) throw new Error(`${target} must not declare a public bin`);
    const binary = join(workspace, target, "bin", "nibid");
    await access(binary, constants.X_OK);
    if ((await stat(binary)).size < 1_000_000) throw new Error(`${target} binary is implausibly small`);
  }
}

export interface PreparedRelease extends FetchResult {
  /** npm package version, which may add a prerelease suffix to `version`. */
  distributionVersion: string;
  output: string;
  workspace: string;
  tarballs: string[];
}

/**
 * Builds an ignored workspace instead of modifying package templates. A source
 * release may be packaged more than once, for example as `2.19.0-rc.1` and
 * `2.19.0-npm.1`, without leaving either version in the working tree.
 */
async function prepareFetchedRelease(
  fetched: FetchResult,
  distributionVersion: string,
  runner: CommandRunner,
): Promise<PreparedRelease> {
  const output = join(distRoot, cacheKey(fetched.release), distributionVersion);
  const workspace = join(output, "workspace");
  console.log(`building ${distributionVersion} packages from ${fetched.release.tagName}`);
  await rm(output, { recursive: true, force: true });
  await mkdir(workspace, { recursive: true });
  await cp(templatePackages, workspace, { recursive: true, filter: (source) => !source.endsWith("/bin/nibid") });
  await setPackageVersions(workspace, distributionVersion);
  await addPackageLicenses(workspace);
  for (const [target, platform] of Object.entries(TARGETS)) {
    console.log(`staging ${target} native binary`);
    await stageBinary(
      join(fetched.directory, `nibid_${fetched.version}_${platform.artifact}.tar.gz`),
      join(workspace, target, "bin", "nibid"),
      runner,
    );
  }
  await verifyWorkspace(workspace, distributionVersion);
  const tarballs: string[] = [];
  for (const packageDir of [...Object.keys(TARGETS), "cli"]) {
    console.log(`packing ${packageDir}`);
    await mustRun(runner, "bun", ["pm", "pack", "--destination", output], join(workspace, packageDir));
  }
  for (const entry of await readdir(output)) {
    if (entry.endsWith(".tgz")) tarballs.push(join(output, entry));
  }
  if (tarballs.length !== Object.keys(TARGETS).length + 1) throw new Error("Expected one tarball for every npm package");
  return { ...fetched, distributionVersion, output, workspace, tarballs: tarballs.sort() };
}

export async function prepareRelease(input: string, runner: CommandRunner = runCommand): Promise<PreparedRelease> {
  const fetched = await getArtifacts(input, runner);
  return prepareFetchedRelease(fetched, fetched.version, runner);
}

function packageDirectories(workspace: string): string[] {
  return [...Object.keys(TARGETS), "cli"].map((name) => join(workspace, name));
}

async function assertUnpublished(workspace: string, version: string, runner: CommandRunner): Promise<void> {
  for (const directory of packageDirectories(workspace)) {
    const manifest = await Bun.file(join(directory, "package.json")).json();
    console.log(`checking npm for ${manifest.name}@${version}`);
    const result = await runner("bun", ["pm", "view", `${manifest.name}@${version}`, "version"]);
    // npm cannot replace a published version. Check every package before the
    // first upload so a known collision does not create a partial release.
    if (result.code === 0) throw new Error(`${manifest.name}@${version} is already published`);
    if (!/404|not found/i.test(`${result.stdout}\n${result.stderr}`)) {
      failCommand("bun", ["pm", "view", `${manifest.name}@${version}`, "version"], result);
    }
  }
}

export async function publishRelease(
  fromGh: string,
  distributionVersion: string,
  run: boolean,
  runner: CommandRunner = runCommand,
): Promise<PreparedRelease> {
  const expectedVersion = normalizeDistributionVersion(distributionVersion);
  const fetched = await getArtifacts(fromGh, runner);
  const sourceVersion = parseVersion(fetched.version);
  const expectedCore = parseVersion(expectedVersion).core;
  if (sourceVersion.core !== expectedCore) {
    throw new Error(
      `GitHub release ${fetched.release.tagName} contains binary version ${fetched.version}, whose SemVer core ${sourceVersion.core} does not match --dist-ver ${expectedVersion}`,
    );
  }
  // The suffix is npm-release metadata, not a claim that the binary changed.
  // `2.19.0-rc.1` may wrap a verified `2.19.0` binary. `2.19.1-rc.1` may not.
  const prepared = await prepareFetchedRelease(fetched, expectedVersion, runner);
  await assertUnpublished(prepared.workspace, prepared.distributionVersion, runner);
  for (const directory of packageDirectories(prepared.workspace)) {
    // Do not expose npm registry channel labels as a release-helper choice.
    // The immutable package version is `--dist-ver`; npm applies its own normal
    // default registry behavior when Bun publishes without `--tag`.
    const args = ["publish", "--access", "public"];
    if (!run) {
      console.log(`[dry run] (cd ${directory} && bun ${args.join(" ")})`);
      continue;
    }
    const manifest = await Bun.file(join(directory, "package.json")).json();
    console.log(`publishing ${manifest.name}@${prepared.distributionVersion}`);
    // Publishing deliberately bypasses the captured command runner so npm can
    // ask the operator to complete the account's write-time 2FA challenge.
    await publishInteractively(directory, args);
  }
  return prepared;
}

export function normalizeDistributionVersion(input: string): string {
  return parseVersion(input).version;
}

/**
 * Parses the npm version shape this helper owns. It deliberately excludes build
 * metadata (`+...`): npm package identity should be one clear publishable
 * version, while prerelease suffixes (`-rc.1`, `-npm.1`) remain useful labels.
 */
export function parseVersion(input: string): ParsedVersion {
  const version = input.trim().replace(/^v/, "");
  const match = version.match(/^(\d+)\.(\d+)\.(\d+)(?:-[0-9A-Za-z.-]+)?$/);
  if (!match) {
    throw new Error(`--dist-ver must be an npm version such as 2.19.0 or v2.19.0, not ${input}`);
  }
  return { version, core: `${match[1]}.${match[2]}.${match[3]}` };
}

interface ListOptions {
  json?: boolean;
  long?: boolean;
  verify?: boolean;
}

interface PublishOptions {
  fromGh: string;
  distVer: string;
  run?: boolean;
}

/**
 * Constructs the operator CLI. Keep release work in the exported functions
 * above; Commander owns only argument validation, help, and presentation.
 */
export function createProgram(): Command {
  const program = new Command()
    .name("bun run main.ts")
    .description("Cache GitHub nibid releases and prepare native CLI distributions.")
    .showHelpAfterError();

  program.action(() => {
    program.help({ error: false });
  });

  program
    .command("list")
    .description("Show locally cached release assets without contacting GitHub")
    .option("--long", "show each expected archive")
    .option("--verify", "hash archives against their cached checksum manifests")
    .option("--json", "write structured cache records as JSON")
    .addHelpText(
      "after",
      "\nExamples:\n  bun run main.ts list\n  bun run main.ts list --verify\n  bun run main.ts list --json\n",
    )
    .action(async (options: ListOptions) => {
      const entries = await readCachedReleases(artifactsRoot, options.verify);
      if (options.json) {
        process.stdout.write(`${JSON.stringify(entries, null, 2)}\n`);
      } else {
        process.stdout.write(renderCachedReleases(entries, options.long));
      }
      if (options.verify && entries.some((entry) => entry.state === "invalid" || entry.state === "incomplete" || entry.state === "malformed")) {
        process.exitCode = 1;
      }
    });

  program
    .command("get")
    .description("Download and checksum-verify release assets, or reuse a valid local cache")
    .argument("<github-release>", "exact GitHub tag, unique release version, or release URL")
    .action(async (input: string) => {
      const fetched = await getArtifacts(input);
      console.log(`${fetched.cached ? "using cached" : "downloaded"} ${fetched.version} at ${fetched.directory}`);
    });

  program
    .command("prepare")
    .description("Create npm package tarballs from cached or downloaded release assets")
    .argument("<github-release>", "exact GitHub tag, unique release version, or release URL")
    .action(async (input: string) => {
      const prepared = await prepareRelease(input);
      console.log(`prepared ${prepared.distributionVersion} at ${prepared.output}`);
    });

  program
    .command("publish")
    .description("Prepare native packages, then publish the umbrella package last")
    .requiredOption("--from-gh <github-release>", "GitHub release that supplies the native binaries")
    .requiredOption("--dist-ver <version>", "npm package version to publish", normalizeDistributionVersion)
    .option("--run", "publish to npm; default prints the ordered publish plan")
    .addHelpText(
      "after",
      "\nExamples:\n  bun run main.ts publish --from-gh hotfix/v2.19.0 --dist-ver 2.19.0-npm.1\n  bun run main.ts publish --from-gh hotfix/v2.19.0 --dist-ver 2.19.0 --run\n",
    )
    .action(async (options: PublishOptions) => {
      const prepared = await publishRelease(options.fromGh, options.distVer, options.run ?? false);
      console.log(`${options.run ? "published" : "dry-run complete for"} ${prepared.distributionVersion}`);
    });

  return program;
}
