import { mkdir, mkdtemp, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { describe, expect, test } from "bun:test";

import {
  buildPublishDryRunPlan,
  computeNextLibwasmvmReleaseTag,
  normalizeReleaseTag,
  parseLibwasmvmReleaseTag,
  publishArtifacts,
  sortLibwasmvmReleaseTags,
  validateExistingArtifacts,
} from "./releaseArtifacts";

const writeRequiredArtifacts = async (artifactsDir: string): Promise<void> => {
  await mkdir(artifactsDir, { recursive: true });
  await Promise.all(
    [
      "libwasmvm_muslc.x86_64.a",
      "libwasmvm_muslc.aarch64.a",
      "libwasmvm.x86_64.so",
      "libwasmvm.aarch64.so",
      "libwasmvm.dylib",
      "libwasmvmstatic_darwin.a",
    ].map((fileName) => Bun.write(join(artifactsDir, fileName), fileName)),
  );
};

describe("release tags", () => {
  test("accepts only the new tag format for publishing", () => {
    expect(normalizeReleaseTag("lib/wasmvm/v1.12.0")).toBe(
      "lib/wasmvm/v1.12.0",
    );
    expect(() => normalizeReleaseTag("lib/wasmvm-ffi/v1.12.0")).toThrow(
      "Invalid release target",
    );
    expect(() => normalizeReleaseTag("lib/wasmvm/v1.12")).toThrow(
      "Invalid release target",
    );
  });

  test("uses legacy tags as version history while suggesting the new prefix", () => {
    expect(
      computeNextLibwasmvmReleaseTag(["lib/wasmvm-ffi/v1.11.0"]),
    ).toBe("lib/wasmvm/v1.12.0");
    expect(
      sortLibwasmvmReleaseTags([
        "lib/wasmvm/v1.12.0",
        "lib/wasmvm-ffi/v1.11.0",
      ]),
    ).toEqual(["lib/wasmvm-ffi/v1.11.0", "lib/wasmvm/v1.12.0"]);
    expect(parseLibwasmvmReleaseTag("unrelated/v1.12.0")).toBeUndefined();
  });
});

describe("release assets", () => {
  test("requires every downloaded release artifact", async () => {
    const artifactsDir = await mkdtemp(join(tmpdir(), "libwasmvm-artifacts-"));
    try {
      await writeRequiredArtifacts(artifactsDir);
      await expect(validateExistingArtifacts(artifactsDir)).resolves.toBeUndefined();
      await rm(join(artifactsDir, "libwasmvm.dylib"));
      await expect(validateExistingArtifacts(artifactsDir)).rejects.toThrow(
        "Missing required artifact file",
      );
    } finally {
      await rm(artifactsDir, { recursive: true, force: true });
    }
  });
});

describe("publishing", () => {
  test("plans a release upload without creating or pushing a tag", () => {
    const plan = buildPublishDryRunPlan(
      "main",
      "abc123",
      "lib/wasmvm/v1.12.0",
      "tmp-artifacts",
    );

    expect(plan.commands).toEqual([
      "test -d tmp-artifacts",
      'gh release create lib/wasmvm/v1.12.0 tmp-artifacts/libwasmvm_muslc.x86_64.a tmp-artifacts/libwasmvm_muslc.aarch64.a tmp-artifacts/libwasmvm.x86_64.so tmp-artifacts/libwasmvm.aarch64.so tmp-artifacts/libwasmvm.dylib tmp-artifacts/libwasmvmstatic_darwin.a --repo NibiruChain/nibiru --title "lib/wasmvm/v1.12.0" --notes-file <release-body.md>',
    ]);
    expect(plan.commands.join("\n")).not.toContain("git tag");
    expect(plan.commands.join("\n")).not.toContain("git push");
  });

  test("creates a release for the existing pushed tag without mutating tags", async () => {
    const tempDir = await mkdtemp(join(tmpdir(), "libwasmvm-artifacts-"));
    const artifactsDir = join(tempDir, "release-artifacts");
    const commands: string[] = [];
    await writeRequiredArtifacts(artifactsDir);

    try {
      await publishArtifacts("lib/wasmvm/v1.12.0", {
        run: true,
        artifactsDir,
        runner: async (command) => {
          commands.push(command);
          if (command === "git branch --show-current") return "main\n";
          if (command === "git rev-parse HEAD") return "abc123\n";
          if (command === "command -v gh >/dev/null 2>&1") return "";
          if (command === "git rev-parse 'lib/wasmvm/v1.12.0^{commit}'") {
            return "abc123\n";
          }
          if (command.includes("git log -1 --format=%s")) {
            return "Update wasm runtime\n";
          }
          if (command.startsWith("gh release create")) return "";
          throw new Error(`Unexpected command: ${command}`);
        },
      });

      expect(commands.some((command) => command.startsWith("gh release create"))).toBe(
        true,
      );
      expect(commands.some((command) => command.startsWith("git tag"))).toBe(false);
      expect(commands.some((command) => command.startsWith("git push"))).toBe(false);
    } finally {
      await rm(tempDir, { recursive: true, force: true });
    }
  });
});
