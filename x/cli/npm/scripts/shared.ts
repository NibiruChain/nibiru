export const targets = {
  "linux-x64": { os: "linux", cpu: "x64", artifact: "linux_amd64" },
  "linux-arm64": { os: "linux", cpu: "arm64", artifact: "linux_arm64" },
  "darwin-x64": { os: "darwin", cpu: "x64", artifact: "darwin_amd64" },
  "darwin-arm64": { os: "darwin", cpu: "arm64", artifact: "darwin_arm64" },
} as const;

export type Target = keyof typeof targets;

export function version(): string {
  const value = process.env.NIBID_NPM_VERSION;
  if (!value || value.startsWith("v") || !/^\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?$/.test(value)) {
    throw new Error("set NIBID_NPM_VERSION to an npm version without a leading v, for example 2.19.0");
  }
  return value;
}

export function selectedTargets(): Target[] {
  const requested = process.env.NIBID_NPM_TARGETS?.split(",").filter(Boolean) ?? Object.keys(targets);
  for (const target of requested) {
    if (!(target in targets)) throw new Error(`unknown NIBID_NPM_TARGETS entry: ${target}`);
  }
  return requested as Target[];
}
