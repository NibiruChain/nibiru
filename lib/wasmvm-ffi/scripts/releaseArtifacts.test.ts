import { mkdir, mkdtemp, rm } from "node:fs/promises"
import { tmpdir } from "node:os"
import { join } from "node:path"
import { describe, expect, test } from "bun:test"

import {
  assertArtifactCommitAfterLatestRelease,
  buildPublishDryRunPlan,
  computeNextLibwasmvmReleaseTag,
  findLatestLibwasmvmReleaseTag,
  findLatestLibwasmvmReleaseTagFromGhOutput,
  normalizeReleaseTag,
  parseGhReleaseListTags,
  parseLibwasmvmReleaseTag,
  publishArtifacts,
  quoteShellArg,
  REAL_FILTERED_GH_RELEASE_LIST_OUTPUT,
  renderReleaseMetadataMarkdown,
  sortLibwasmvmReleaseTags,
  validateExistingArtifacts,
} from "./releaseArtifacts"

const RELEASE_LIST_WITH_LIBWASMVM_RELEASES = JSON.stringify([
  { tagName: "v1.5.9" },
  { tagName: "v1.6.0" },
  { tagName: "libwasmvm/v9.9.9" },
  { tagName: "v1.10.0" },
])

const writeRequiredArtifacts = async (artifactsDir: string): Promise<void> => {
  await mkdir(artifactsDir, { recursive: true })
  await Bun.write(
    Bun.file(join(artifactsDir, "libwasmvm_muslc.x86_64.a")),
    "musl x86",
  )
  await Bun.write(
    Bun.file(join(artifactsDir, "libwasmvm_muslc.aarch64.a")),
    "musl arm",
  )
  await Bun.write(
    Bun.file(join(artifactsDir, "libwasmvm.x86_64.so")),
    "glibc x86",
  )
  await Bun.write(
    Bun.file(join(artifactsDir, "libwasmvm.aarch64.so")),
    "glibc arm",
  )
  await Bun.write(
    Bun.file(join(artifactsDir, "libwasmvm.dylib")),
    "darwin universal dylib",
  )
  await Bun.write(
    Bun.file(join(artifactsDir, "libwasmvmstatic_darwin.a")),
    "darwin universal static",
  )
}

describe("normalizeReleaseTag", () => {
  test("keeps a plain semantic version tag unchanged", () => {
    expect(normalizeReleaseTag("v1.6.0")).toBe("v1.6.0")
  })

  test("normalizes a GitHub Release URL", () => {
    const url = "https://github.com/NibiruChain/go-wasmvm/releases/tag/v1.6.0"

    expect(normalizeReleaseTag(url)).toBe("v1.6.0")
  })

  test("trims surrounding whitespace", () => {
    expect(normalizeReleaseTag("  v1.6.0  ")).toBe("v1.6.0")
  })

  test("rejects unsupported release targets", () => {
    expect(() => normalizeReleaseTag("v1.6")).toThrow("Invalid release target")
    expect(() => normalizeReleaseTag("libwasmvm/v1.6.0")).toThrow(
      "Invalid release target",
    )
    expect(() =>
      normalizeReleaseTag(
        "https://github.com/CosmWasm/wasmvm/releases/tag/v1.6.0",
      ),
    ).toThrow("Invalid release target")
  })
})

describe("parseLibwasmvmReleaseTag", () => {
  test("parses a plain semantic version release tag", () => {
    expect(parseLibwasmvmReleaseTag("v1.6.0")).toEqual({
      tag: "v1.6.0",
      major: 1,
      minor: 6,
      patch: 0,
    })
  })

  test("ignores unrelated or prerelease tags", () => {
    expect(parseLibwasmvmReleaseTag("libwasmvm/v1.6.0")).toBeUndefined()
    expect(parseLibwasmvmReleaseTag("v1.6.0-rc.1")).toBeUndefined()
    expect(parseLibwasmvmReleaseTag("foo")).toBeUndefined()
  })
})

describe("sortLibwasmvmReleaseTags", () => {
  test("filters unrelated tags and sorts by semantic version", () => {
    expect(
      sortLibwasmvmReleaseTags([
        "v1.10.0",
        "libwasmvm/v9.9.9",
        "v1.6.2",
        "v1.6.10",
        "v1.6.0-rc.1",
      ]),
    ).toEqual(["v1.6.2", "v1.6.10", "v1.10.0"])
  })
})

describe("findLatestLibwasmvmReleaseTag", () => {
  test("returns the highest plain semantic version tag", () => {
    expect(
      findLatestLibwasmvmReleaseTag(["v1.6.0", "v1.7.0", "foo/v99.0.0"]),
    ).toBe("v1.7.0")
  })

  test("returns undefined when no plain semantic version tags exist", () => {
    expect(findLatestLibwasmvmReleaseTag(["libwasmvm/v1.6.0"])).toBeUndefined()
  })
})

describe("computeNextLibwasmvmReleaseTag", () => {
  test("starts the Nibiru release line at v1.6.0", () => {
    expect(computeNextLibwasmvmReleaseTag([])).toBe("v1.6.0")
  })

  test("bumps from the upstream v1.5.9 line to v1.6.0", () => {
    expect(computeNextLibwasmvmReleaseTag(["v1.5.9"])).toBe("v1.6.0")
  })

  test("computes patch, minor, and major bumps", () => {
    const tags = ["v1.6.0"]

    expect(computeNextLibwasmvmReleaseTag(tags, "patch")).toBe("v1.6.1")
    expect(computeNextLibwasmvmReleaseTag(tags, "minor")).toBe("v1.7.0")
    expect(computeNextLibwasmvmReleaseTag(tags, "major")).toBe("v2.0.0")
  })

  test("rejects invalid bump types", () => {
    expect(() => computeNextLibwasmvmReleaseTag(["v1.6.0"], "banana")).toThrow(
      "Invalid bump type",
    )
  })
})

describe("parseGhReleaseListTags", () => {
  test("parses the captured real filtered gh release list output", () => {
    expect(parseGhReleaseListTags(REAL_FILTERED_GH_RELEASE_LIST_OUTPUT)).toEqual(
      [],
    )
  })

  test("extracts tag names from gh release JSON output", () => {
    expect(parseGhReleaseListTags(RELEASE_LIST_WITH_LIBWASMVM_RELEASES)).toEqual(
      ["v1.5.9", "v1.6.0", "libwasmvm/v9.9.9", "v1.10.0"],
    )
  })

  test("rejects non-array gh release JSON output", () => {
    expect(() => parseGhReleaseListTags("{}")).toThrow(
      "Expected gh release list JSON output",
    )
  })
})

describe("findLatestLibwasmvmReleaseTagFromGhOutput", () => {
  test("returns undefined for the captured real output with no releases", () => {
    expect(
      findLatestLibwasmvmReleaseTagFromGhOutput(
        REAL_FILTERED_GH_RELEASE_LIST_OUTPUT,
      ),
    ).toBeUndefined()
  })

  test("finds the latest libwasmvm release from filtered gh output", () => {
    expect(
      findLatestLibwasmvmReleaseTagFromGhOutput(
        RELEASE_LIST_WITH_LIBWASMVM_RELEASES,
      ),
    ).toBe("v1.10.0")
  })
})

describe("quoteShellArg", () => {
  test("quotes shell arguments with embedded single quotes", () => {
    expect(quoteShellArg("v1.6.0")).toBe("'v1.6.0'")
    expect(quoteShellArg("don't")).toBe("'don'\\''t'")
  })
})

describe("validateExistingArtifacts", () => {
  test("requires all release files", async () => {
    const artifactsDir = await mkdtemp(join(tmpdir(), "libwasmvm-artifacts-"))
    await writeRequiredArtifacts(artifactsDir)

    await expect(validateExistingArtifacts(artifactsDir)).resolves.toBeUndefined()

    await rm(artifactsDir, { recursive: true, force: true })
  })

  test("rejects missing artifact files", async () => {
    const artifactsDir = await mkdtemp(join(tmpdir(), "libwasmvm-artifacts-"))
    await writeRequiredArtifacts(artifactsDir)
    await rm(join(artifactsDir, "libwasmvm.aarch64.so"))

    await expect(validateExistingArtifacts(artifactsDir)).rejects.toThrow(
      "Missing required artifact file",
    )

    await rm(artifactsDir, { recursive: true, force: true })
  })
})

describe("buildPublishDryRunPlan", () => {
  test("builds a non-mutating publish plan", () => {
    const plan = buildPublishDryRunPlan(
      "feature",
      "abc123",
      ["v1.5.9"],
      "minor",
      "tmp-artifacts",
    )

    expect(plan).toEqual({
      bumpType: "minor",
      branch: "feature",
      commitSha: "abc123",
      nextTag: "v1.6.0",
      commands: [
        "test -d tmp-artifacts",
        'git tag --no-sign -a v1.6.0 abc123 -m "libwasmvm v1.6.0"',
        "git push origin v1.6.0",
        "gh release create v1.6.0 " +
          "tmp-artifacts/libwasmvm_muslc.x86_64.a " +
          "tmp-artifacts/libwasmvm_muslc.aarch64.a " +
          "tmp-artifacts/libwasmvm.x86_64.so " +
          "tmp-artifacts/libwasmvm.aarch64.so " +
          "tmp-artifacts/libwasmvm.dylib " +
          "tmp-artifacts/libwasmvmstatic_darwin.a " +
          '--repo NibiruChain/go-wasmvm --title "libwasmvm v1.6.0" ' +
          "--notes-file <release-body.md>",
      ],
    })
  })
})

describe("renderReleaseMetadataMarkdown", () => {
  test("renders release metadata and release assets", () => {
    const releaseUrl = "https://github.com/NibiruChain/go-wasmvm/releases/tag/v1.6.0"
    const markdown = renderReleaseMetadataMarkdown({
      tag: "v1.6.0",
      commitSha: "abc123",
      commitSubject: "Update wasm runtime",
      releaseUrl,
      workflowRunUrl:
        "https://github.com/NibiruChain/go-wasmvm/actions/runs/1",
    })

    expect(markdown).toContain(`- Release tag: [\`v1.6.0\`](${releaseUrl})`)
    expect(markdown).toContain("- Source commit: `abc123` Update wasm runtime")
    expect(markdown).toContain("- `libwasmvm_muslc.x86_64.a`")
    expect(markdown).toContain("- `libwasmvm.aarch64.so`")
    expect(markdown).toContain("- `libwasmvm.dylib`")
    expect(markdown).toContain("- `libwasmvmstatic_darwin.a`")
    expect(markdown).not.toContain("bindings.h")
  })
})

describe("assertArtifactCommitAfterLatestRelease", () => {
  test("passes when no stable releases exist yet", async () => {
    await assertArtifactCommitAfterLatestRelease("abc123", async (command) => {
      if (command.startsWith("gh release list ")) {
        return "[]"
      }
      throw new Error(`Unexpected command: ${command}`)
    })
  })

  test("passes when the latest release commit is an ancestor", async () => {
    const commands: string[] = []
    await assertArtifactCommitAfterLatestRelease("new123", async (command) => {
      commands.push(command)
      if (command.startsWith("gh release list ")) {
        return JSON.stringify([{ tagName: "v1.6.0" }])
      }
      if (command.includes("git rev-parse ")) return "old123\n"
      if (command.includes("git merge-base --is-ancestor")) return ""
      throw new Error(`Unexpected command: ${command}`)
    })

    expect(commands.some((command) => command.includes("merge-base"))).toBe(true)
  })
})

describe("publishArtifacts", () => {
  test("publishes existing artifacts", async () => {
    const tempDir = await mkdtemp(join(tmpdir(), "libwasmvm-artifacts-"))
    const artifactsDir = join(tempDir, "release-artifacts")
    const commands: string[] = []

    await writeRequiredArtifacts(artifactsDir)

    await publishArtifacts("minor", {
      run: true,
      artifactsDir,
      runner: async (command) => {
        commands.push(command)
        if (command === "git branch --show-current") return "main\n"
        if (command === "git rev-parse HEAD") return "abc123\n"
        if (command === `git tag --list "v*.*.*"`) return "v1.5.9\n"

        const emptyExact = ["command -v gh >/dev/null 2>&1"]
        const emptyPrefixes = [
          "git tag --no-sign",
          "git push origin",
          "gh release create",
        ]

        if (
          emptyExact.includes(command) ||
          emptyPrefixes.some((prefix) => command.startsWith(prefix))
        ) {
          return ""
        }

        if (command.includes("git tag --points-at 'abc123'")) return ""
        if (command.startsWith("gh release list ")) {
          return JSON.stringify([{ tagName: "v1.5.9" }])
        }
        if (command.includes("git rev-parse ")) return "old123\n"
        if (command.includes("git merge-base --is-ancestor")) return ""
        if (command.includes("git log -1 --format=%s")) {
          return "Update wasm runtime\n"
        }

        throw new Error(`Unexpected command: ${command}`)
      },
    })

    expect(commands.some((command) => command.startsWith("gh release create"))).toBe(
      true,
    )
    expect(
      commands.some((command) =>
        command.includes(`${artifactsDir}/libwasmvm_muslc.x86_64.a`),
      ),
    ).toBe(true)

    await rm(tempDir, { recursive: true, force: true })
  })

  test("skips when the artifact commit already has a release", async () => {
    const tempDir = await mkdtemp(join(tmpdir(), "libwasmvm-artifacts-"))
    const artifactsDir = join(tempDir, "release-artifacts")
    const commands: string[] = []

    await writeRequiredArtifacts(artifactsDir)

    await publishArtifacts("minor", {
      run: true,
      artifactsDir,
      runner: async (command) => {
        commands.push(command)
        if (command === "git branch --show-current") return "main\n"
        if (command === "git rev-parse HEAD") return "abc123\n"
        if (command === `git tag --list "v*.*.*"`) return "v1.6.0\n"
        if (command === "command -v gh >/dev/null 2>&1") return ""
        if (command.includes("git tag --points-at 'abc123'")) return "v1.6.0\n"
        if (command.includes("gh release view 'v1.6.0'")) {
          return JSON.stringify({ tagName: "v1.6.0" })
        }
        throw new Error(`Unexpected command: ${command}`)
      },
    })

    expect(commands.some((command) => command.startsWith("gh release create"))).toBe(
      false,
    )

    await rm(tempDir, { recursive: true, force: true })
  })

  test("fails when the artifact commit has a tag without a release", async () => {
    const tempDir = await mkdtemp(join(tmpdir(), "libwasmvm-artifacts-"))
    const artifactsDir = join(tempDir, "release-artifacts")

    await writeRequiredArtifacts(artifactsDir)

    await expect(
      publishArtifacts("minor", {
        run: true,
        artifactsDir,
        runner: async (command) => {
          if (command === "git branch --show-current") return "main\n"
          if (command === "git rev-parse HEAD") return "abc123\n"
          if (command === `git tag --list "v*.*.*"`) return "v1.6.0\n"
          if (command === "command -v gh >/dev/null 2>&1") return ""
          if (command.includes("git tag --points-at 'abc123'")) return "v1.6.0\n"
          if (command.includes("gh release view 'v1.6.0'")) {
            throw new Error("release not found")
          }
          throw new Error(`Unexpected command: ${command}`)
        },
      }),
    ).rejects.toThrow("does not have a GitHub Release")

    await rm(tempDir, { recursive: true, force: true })
  })
})
