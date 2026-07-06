#!/usr/bin/env bun
import { spawn } from "node:child_process"
import { mkdtemp, rm, stat } from "node:fs/promises"
import { tmpdir } from "node:os"
import { join } from "node:path"
import { Command } from "commander"

const DEFAULT_BUMP_TYPE = "minor"
const FIRST_RELEASE_TAG = "v1.6.0"
const RELEASE_VERSION_PATTERN = "v[0-9]+\\.[0-9]+\\.[0-9]+"
const GITHUB_REPO = "NibiruChain/go-wasmvm"
const LOCAL_ARTIFACTS_DIR = "release-artifacts"
const RELEASE_BRANCH = "main"
const BUMP_TYPES = ["patch", "minor", "major"] as const
const REQUIRED_RELEASE_ARTIFACTS = [
  "libwasmvm_muslc.x86_64.a",
  "libwasmvm_muslc.aarch64.a",
  "libwasmvm.x86_64.so",
  "libwasmvm.aarch64.so",
  "libwasmvm.dylib",
  "libwasmvmstatic_darwin.a",
] as const

type BumpType = (typeof BUMP_TYPES)[number]
type CommandRunner = (command: string) => Promise<string>

export interface LibwasmvmReleaseVersion {
  tag: string
  major: number
  minor: number
  patch: number
}

export interface PublishDryRunPlan {
  bumpType: BumpType
  branch: string
  commitSha: string
  nextTag: string
  commands: string[]
}

export interface PublishReleaseMetadata {
  tag: string
  commitSha: string
  commitSubject: string
  releaseUrl: string
  workflowRunUrl?: string
}

export const REAL_FILTERED_GH_RELEASE_LIST_OUTPUT = `[]`

export const runShellCommand = async (command: string): Promise<string> => {
  console.log(`$ ${command}`)

  return new Promise((resolve, reject) => {
    const child = spawn(command, {
      shell: true,
      stdio: ["ignore", "pipe", "pipe"],
    })
    let stdout = ""
    let stderr = ""

    child.stdout.setEncoding("utf8")
    child.stderr.setEncoding("utf8")
    child.stdout.on("data", (chunk) => {
      stdout += chunk
    })
    child.stderr.on("data", (chunk) => {
      stderr += chunk
    })
    child.on("error", reject)
    child.on("close", (code) => {
      if (stderr.trim() !== "") {
        console.error(stderr.trim())
      }
      if (code !== 0) {
        reject(
          new Error(
            `Command failed with exit code ${code}: ` +
              `${stderr || stdout || command}`,
          ),
        )
        return
      }
      resolve(stdout)
    })
  })
}

export const quoteShellArg = (value: string): string => {
  return `'${value.replaceAll("'", "'\\''")}'`
}

export const normalizeBumpType = (value: string | undefined): BumpType => {
  if (value === undefined) {
    return DEFAULT_BUMP_TYPE
  }

  if (BUMP_TYPES.includes(value as BumpType)) {
    return value as BumpType
  }

  throw new Error(
    `Invalid bump type "${value}". Use one of: ${BUMP_TYPES.join(", ")}`,
  )
}

export const normalizeReleaseTag = (value: string): string => {
  const input = value.trim()
  const versionRegex = new RegExp(`^${RELEASE_VERSION_PATTERN}$`)

  if (versionRegex.test(input)) {
    return input
  }

  try {
    const url = new URL(input)
    const releasePathPrefix = "/NibiruChain/go-wasmvm/releases/tag/"

    if (url.hostname !== "github.com") {
      throw new Error("URL is not a github.com release URL")
    }

    if (!url.pathname.startsWith(releasePathPrefix)) {
      throw new Error("URL path is not a go-wasmvm release tag path")
    }

    const tag = decodeURIComponent(url.pathname.slice(releasePathPrefix.length))
    if (versionRegex.test(tag)) {
      return tag
    }
  } catch {
    // Fall through to the shared error below.
  }

  throw new Error(
    `Invalid release target "${value}". Use vX.Y.Z or a go-wasmvm GitHub Release URL.`,
  )
}

export const parseLibwasmvmReleaseTag = (
  tag: string,
): LibwasmvmReleaseVersion | undefined => {
  const match = tag.match(
    /^v(?<major>[0-9]+)\.(?<minor>[0-9]+)\.(?<patch>[0-9]+)$/,
  )

  if (match?.groups === undefined) {
    return undefined
  }

  return {
    tag,
    major: Number(match.groups.major),
    minor: Number(match.groups.minor),
    patch: Number(match.groups.patch),
  }
}

const compareLibwasmvmReleaseVersions = (
  left: LibwasmvmReleaseVersion,
  right: LibwasmvmReleaseVersion,
): number => {
  return (
    left.major - right.major ||
    left.minor - right.minor ||
    left.patch - right.patch
  )
}

export const sortLibwasmvmReleaseTags = (tags: string[]): string[] => {
  return tags
    .map(parseLibwasmvmReleaseTag)
    .filter(
      (version): version is LibwasmvmReleaseVersion => version !== undefined,
    )
    .sort(compareLibwasmvmReleaseVersions)
    .map((version) => version.tag)
}

export const findLatestLibwasmvmReleaseTag = (
  tags: string[],
): string | undefined => {
  return sortLibwasmvmReleaseTags(tags).at(-1)
}

export const computeNextLibwasmvmReleaseTag = (
  tags: string[],
  bumpType?: string,
): string => {
  const normalizedBumpType = normalizeBumpType(bumpType)
  const latestTag = findLatestLibwasmvmReleaseTag(tags)
  const latestVersion =
    latestTag === undefined ? undefined : parseLibwasmvmReleaseTag(latestTag)

  if (latestVersion === undefined) {
    return FIRST_RELEASE_TAG
  }

  if (normalizedBumpType === "patch") {
    return `v${latestVersion.major}.${latestVersion.minor}.${
      latestVersion.patch + 1
    }`
  }

  if (normalizedBumpType === "minor") {
    return `v${latestVersion.major}.${latestVersion.minor + 1}.0`
  }

  return `v${latestVersion.major + 1}.0.0`
}

export const parseGhReleaseListTags = (output: string): string[] => {
  const parsed = JSON.parse(output) as Array<{ tagName: string }>
  if (!Array.isArray(parsed)) {
    throw new Error("Expected gh release list JSON output to be an array")
  }

  return parsed.map((release) => release.tagName)
}

export const findLatestLibwasmvmReleaseTagFromGhOutput = (
  output: string,
): string | undefined => {
  return findLatestLibwasmvmReleaseTag(parseGhReleaseListTags(output))
}

export const getLocalGitTags = async (
  runner: CommandRunner = runShellCommand,
): Promise<string[]> => {
  const stdout = await runner(`git tag --list "v*.*.*"`)
  return stdout
    .split("\n")
    .map((line) => line.trim())
    .filter((line) => line.length > 0)
}

export const getLatestStableReleaseTag = async (
  runner: CommandRunner = runShellCommand,
): Promise<string> => {
  const stdout = await runner(
    `gh release list --repo ${GITHUB_REPO} --exclude-drafts --exclude-pre-releases --limit 100 --json tagName`,
  )
  const latestTag = findLatestLibwasmvmReleaseTagFromGhOutput(stdout)

  if (latestTag === undefined) {
    throw new Error(`No stable ${GITHUB_REPO} releases found`)
  }

  return latestTag
}

export const ensureGhCli = async (
  runner: CommandRunner = runShellCommand,
): Promise<void> => {
  try {
    await runner("command -v gh >/dev/null 2>&1")
  } catch {
    throw new Error(
      "GitHub CLI `gh` is required. Install it from https://cli.github.com/ " +
        "and run `gh auth login` if the repository requires authentication.",
    )
  }
}

export const validateExistingArtifacts = async (
  artifactsDir: string,
): Promise<void> => {
  let artifactsStat
  try {
    artifactsStat = await stat(artifactsDir)
  } catch {
    throw new Error(`Missing artifacts directory: ${artifactsDir}`)
  }

  if (!artifactsStat.isDirectory()) {
    throw new Error(`Artifacts path is not a directory: ${artifactsDir}`)
  }

  for (const fileName of REQUIRED_RELEASE_ARTIFACTS) {
    const filePath = join(artifactsDir, fileName)
    if (!(await Bun.file(filePath).exists())) {
      throw new Error(`Missing required artifact file: ${filePath}`)
    }
  }
}

export const getLibwasmvmTagsPointingAtCommit = async (
  commit: string,
  runner: CommandRunner = runShellCommand,
): Promise<string[]> => {
  const stdout = await runner(
    `git tag --points-at ${quoteShellArg(commit)} --list "v*.*.*"`,
  )
  return stdout
    .split("\n")
    .map((line) => line.trim())
    .filter((line) => line.length > 0)
}

export const githubReleaseExists = async (
  tag: string,
  runner: CommandRunner = runShellCommand,
): Promise<boolean> => {
  try {
    await runner(
      `gh release view ${quoteShellArg(tag)} --repo ${GITHUB_REPO} --json tagName`,
    )
    return true
  } catch {
    return false
  }
}

export const findReleasedHeadTag = async (
  headTags: string[],
  runner: CommandRunner = runShellCommand,
): Promise<string | undefined> => {
  for (const tag of sortLibwasmvmReleaseTags(headTags).toReversed()) {
    if (await githubReleaseExists(tag, runner)) {
      return tag
    }
  }

  return undefined
}

export const getTagCommit = async (
  tag: string,
  runner: CommandRunner = runShellCommand,
): Promise<string> => {
  return (
    await runner(`git rev-parse ${quoteShellArg(`${tag}^{commit}`)}`)
  ).trim()
}

export const isAncestorCommit = async (
  ancestorCommit: string,
  descendantCommit: string,
  runner: CommandRunner = runShellCommand,
): Promise<boolean> => {
  try {
    await runner(
      `git merge-base --is-ancestor ${quoteShellArg(
        ancestorCommit,
      )} ${quoteShellArg(descendantCommit)}`,
    )
    return true
  } catch {
    return false
  }
}

export const assertArtifactCommitAfterLatestRelease = async (
  artifactCommit: string,
  runner: CommandRunner = runShellCommand,
): Promise<void> => {
  let latestReleaseTag: string
  try {
    latestReleaseTag = await getLatestStableReleaseTag(runner)
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error)
    if (message.includes(`No stable ${GITHUB_REPO} releases found`)) {
      return
    }
    throw error
  }

  const latestReleaseCommit = await getTagCommit(latestReleaseTag, runner)
  if (latestReleaseCommit === artifactCommit) {
    throw new Error(
      `Artifact commit ${artifactCommit} is already the latest release commit ` +
        `for ${latestReleaseTag}`,
    )
  }

  if (!(await isAncestorCommit(latestReleaseCommit, artifactCommit, runner))) {
    throw new Error(
      `Artifact commit ${artifactCommit} is not after latest release ` +
        `${latestReleaseTag} (${latestReleaseCommit})`,
    )
  }
}

export const buildPublishDryRunPlan = (
  branch: string,
  commitSha: string,
  localTags: string[],
  bumpType?: string,
  artifactsDir = LOCAL_ARTIFACTS_DIR,
): PublishDryRunPlan => {
  const normalizedBumpType = normalizeBumpType(bumpType)
  const nextTag = computeNextLibwasmvmReleaseTag(localTags, normalizedBumpType)
  const artifactArgs = REQUIRED_RELEASE_ARTIFACTS.map(
    (fileName) => `${artifactsDir}/${fileName}`,
  ).join(" ")

  return {
    bumpType: normalizedBumpType,
    branch,
    commitSha,
    nextTag,
    commands: [
      `test -d ${artifactsDir}`,
      `git tag --no-sign -a ${nextTag} ${commitSha} -m "libwasmvm ${nextTag}"`,
      `git push origin ${nextTag}`,
      `gh release create ${nextTag} ${artifactArgs} --repo ${GITHUB_REPO} --title "libwasmvm ${nextTag}" --notes-file <release-body.md>`,
    ],
  }
}

export const printPublishDryRun = (plan: PublishDryRunPlan): void => {
  console.log("Dry run: no release changes were made.")
  console.log(
    "Pass --run to tag the commit, create the GitHub Release, and upload assets.",
  )
  console.log("")
  console.log(`Branch: ${plan.branch}`)
  console.log(`Commit: ${plan.commitSha}`)
  console.log(`Bump: ${plan.bumpType}`)
  console.log(`Next tag: ${plan.nextTag}`)
  console.log("")
  console.log("Commands that would run:")
  for (const command of plan.commands) {
    console.log(`  ${command}`)
  }
}

export const buildWorkflowRunUrl = (): string | undefined => {
  const serverUrl = process.env.GITHUB_SERVER_URL
  const repository = process.env.GITHUB_REPOSITORY
  const runId = process.env.GITHUB_RUN_ID

  if (
    serverUrl === undefined ||
    repository === undefined ||
    runId === undefined
  ) {
    return undefined
  }

  return `${serverUrl}/${repository}/actions/runs/${runId}`
}

export const renderReleaseMetadataMarkdown = (
  metadata: PublishReleaseMetadata,
): string => {
  const lines = [
    "## Release metadata",
    "",
    `- Release tag: [\`${metadata.tag}\`](${metadata.releaseUrl})`,
    `- Source commit: \`${metadata.commitSha.slice(0, 7)}\` ${
      metadata.commitSubject
    }`,
    "",
    "## Release assets",
    "",
    ...REQUIRED_RELEASE_ARTIFACTS.map((fileName) => `- \`${fileName}\``),
  ]

  if (metadata.workflowRunUrl !== undefined) {
    lines.push("", `- Workflow run: ${metadata.workflowRunUrl}`)
  }

  lines.push("")

  return `${lines.join("\n")}\n`
}

export const createGitTagAndRelease = async (
  tag: string,
  commitSha: string,
  releaseBodyPath: string,
  artifactsDir = LOCAL_ARTIFACTS_DIR,
  runner: CommandRunner = runShellCommand,
): Promise<void> => {
  const artifactArgs = REQUIRED_RELEASE_ARTIFACTS.map((fileName) =>
    quoteShellArg(join(artifactsDir, fileName)),
  ).join(" ")

  await runner(
    `git tag --no-sign -a ${quoteShellArg(tag)} ${quoteShellArg(
      commitSha,
    )} -m ${quoteShellArg(`libwasmvm ${tag}`)}`,
  )
  await runner(`git push origin ${quoteShellArg(tag)}`)
  await runner(
    `gh release create ${quoteShellArg(
      tag,
    )} ${artifactArgs} --repo ${GITHUB_REPO} --title ${quoteShellArg(
      `libwasmvm ${tag}`,
    )} --notes-file ${quoteShellArg(releaseBodyPath)}`,
  )
}

export const publishArtifacts = async (
  bumpType?: string,
  options: {
    run?: boolean
    artifactsDir?: string
    runner?: CommandRunner
  } = {},
): Promise<void> => {
  const runner = options.runner ?? runShellCommand
  const artifactsDir = options.artifactsDir ?? LOCAL_ARTIFACTS_DIR
  const normalizedBumpType = normalizeBumpType(bumpType)
  const branch = (await runner("git branch --show-current")).trim()
  const commitSha = (await runner("git rev-parse HEAD")).trim()
  const localTags = await getLocalGitTags(runner)
  const dryRunPlan = buildPublishDryRunPlan(
    branch,
    commitSha,
    localTags,
    normalizedBumpType,
    artifactsDir,
  )

  if (options.run !== true) {
    printPublishDryRun(dryRunPlan)
    return
  }

  if (branch !== RELEASE_BRANCH) {
    throw new Error(
      `Refusing to publish from branch "${branch}". Use ${RELEASE_BRANCH}.`,
    )
  }

  await ensureGhCli(runner)
  await validateExistingArtifacts(artifactsDir)

  const artifactTags = await getLibwasmvmTagsPointingAtCommit(commitSha, runner)
  if (artifactTags.length > 0) {
    const releasedArtifactTag = await findReleasedHeadTag(artifactTags, runner)
    if (releasedArtifactTag !== undefined) {
      console.log(
        `${releasedArtifactTag} already has a GitHub Release; skipping.`,
      )
      return
    }

    const latestArtifactTag = sortLibwasmvmReleaseTags(artifactTags).at(-1)
    throw new Error(
      `${latestArtifactTag} tags artifact commit ${commitSha} ` +
        "but does not have a GitHub Release.",
    )
  }

  await assertArtifactCommitAfterLatestRelease(commitSha, runner)

  if (localTags.includes(dryRunPlan.nextTag)) {
    throw new Error(
      `${dryRunPlan.nextTag} already exists but does not have a release ` +
        "for the current HEAD.",
    )
  }

  const commitSubject = (
    await runner(`git log -1 --format=%s ${quoteShellArg(commitSha)}`)
  ).trim()
  const metadata: PublishReleaseMetadata = {
    tag: dryRunPlan.nextTag,
    commitSha,
    commitSubject,
    releaseUrl: `https://github.com/${GITHUB_REPO}/releases/tag/${dryRunPlan.nextTag}`,
    workflowRunUrl: buildWorkflowRunUrl(),
  }
  const releaseNotesDir = await mkdtemp(
    join(tmpdir(), "libwasmvm-release-notes-"),
  )

  try {
    const releaseBodyPath = join(releaseNotesDir, "release-body.md")
    await Bun.write(
      Bun.file(releaseBodyPath),
      renderReleaseMetadataMarkdown(metadata),
    )
    await createGitTagAndRelease(
      dryRunPlan.nextTag,
      commitSha,
      releaseBodyPath,
      artifactsDir,
      runner,
    )
  } finally {
    await rm(releaseNotesDir, { recursive: true, force: true })
  }
  console.log(`Published ${dryRunPlan.nextTag}`)
}

export const printNextTag = async (bumpType?: string): Promise<void> => {
  const tags = await getLocalGitTags()
  console.log(computeNextLibwasmvmReleaseTag(tags, bumpType))
}

export const testReleaseHelper = async (
  runner: CommandRunner = runShellCommand,
): Promise<void> => {
  await runner("bun test scripts")
}

const runCliAction = async (
  action: () => void | Promise<void>,
): Promise<void> => {
  try {
    await action()
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error)
    console.error(message)
    process.exit(1)
  }
}

const program = new Command()

program
  .name("releaseArtifacts")
  .description("go-wasmvm libwasmvm artifact release helper")
  .helpOption("--help", "display help for command")

program
  .command("publish")
  .description("Publish tested libwasmvm artifacts")
  .argument("[bump]", "patch | minor | major", DEFAULT_BUMP_TYPE)
  .option("--run", "execute the release; without this, print a dry run")
  .action((bumpType: string | undefined, options: { run?: boolean }) => {
    return runCliAction(() => publishArtifacts(bumpType, options))
  })

program
  .command("next-tag")
  .description("Compute the next libwasmvm release tag")
  .argument("[bump]", "patch | minor | major", DEFAULT_BUMP_TYPE)
  .action((bumpType?: string) => {
    return runCliAction(() => printNextTag(bumpType))
  })

program
  .command("test")
  .description("Run release helper tests")
  .action(() => {
    return runCliAction(() => testReleaseHelper())
  })

if (import.meta.main) {
  if (process.argv.length <= 2) {
    program.outputHelp()
    process.exit(0)
  }

  await program.parseAsync()
}
