#!/usr/bin/env bun
import Bun from "bun";
import { bash as execCommand, type BashOut } from "@uniquedivine/bash";
import { newClog } from "@uniquedivine/jiyuu";
import { join, relative } from "path";

const cfg = (() => {
  const dirNibiruRepo = join(__dirname, "..");
  const nibiru = (relPath: string) => join(dirNibiruRepo, relPath);
  const obj = {
    dirNibiruRepo,
    dirNibiruProto: nibiru("proto"),
    outPath: nibiru("dist"),
    outOpenapi: nibiru("dist/openapi"),
    bufYaml: nibiru("proto/buf.yaml"),
    bufGenYaml: nibiru("proto/buf-gen-swagger.yaml"),
  };
  return obj;
})();

const { clog, cerr, clogCmd } = newClog(
  import.meta.url.includes("/")
    ? import.meta.url.split("/").pop()!
    : import.meta.url,
);

const requireTools = (): void => {
  const tools = ["go", "buf", "jq", "bun"];

  for (const [_, tool] of tools.entries()) {
    if (Bun.which(tool) == null) {
      throw new Error(`Tool "${tool}" is mssing and not on the $PATH`);
    }
  }
};

// ---------------------------------------------------

/** Runs a command with the working directory as the Nibiru repo. */
const runAtPath = async (cmd: string, path: string): Promise<BashOut> => {
  const wrapped = `( cd "${path}" && ${cmd} )`;
  clogCmd(wrapped);
  return await execCommand(wrapped);
};

type CosmosSdkInfo = {
  cosmosSdkGhPath: string;
  nibiruCosmosSdkVersion: string;
};

async function getCosmosSdkInfo(): Promise<CosmosSdkInfo> {
  // repo + proto roots inferred from your cfg
  const protoDir = cfg.dirNibiruProto;

  if (!Bun.file(protoDir).exists()) {
    throw new Error(`proto dir not found: ${protoDir}`);
  }

  // run the go query from inside the repo root (matches bash)
  const cmd = `( cd "${cfg.dirNibiruRepo}" && go list -f '{{ .Dir }}' -m github.com/cosmos/cosmos-sdk )`;
  clogCmd(cmd);
  const { stdout } = await execCommand(cmd);

  const cosmosSdkGhPath = stdout.trim(); // e.g. $HOME/go/pkg/mod/github.com/cosmos/cosmos-sdk@v0.47.4
  const atIdx = cosmosSdkGhPath.lastIndexOf("@");
  const nibiruCosmosSdkVersion =
    atIdx >= 0 ? cosmosSdkGhPath.slice(atIdx + 1) : "";

  if (!nibiruCosmosSdkVersion) {
    throw new Error(
      `Could not parse Cosmos SDK version from: ${cosmosSdkGhPath}`,
    );
  }

  // trace for sanity
  clog("protoDir:         ", protoDir);
  clog("cosmosSdkGhPath:  ", cosmosSdkGhPath);
  clog("sdk version:      ", nibiruCosmosSdkVersion);
  clog("nibiruRepoPath:   ", cfg.dirNibiruRepo);
  clog("\n%o", { cfg });

  return {
    cosmosSdkGhPath,
    nibiruCosmosSdkVersion,
  };
}

export type GoModEditJSON = {
  Module?: { Path: string };
  Go?: string;

  Require?: Array<{
    Path: string;
    Version?: string;
    Indirect?: boolean;
  }> | null;

  Exclude?: Array<{
    Path: string;
    Version?: string;
  }> | null;

  Replace?: Array<{
    Old: { Path: string; Version?: string };
    New: { Path: string; Version?: string };
  }> | null;

  Retract?: Array<
    | string
    | {
      Low: string;
      High?: string;
      Reason?: string;
    }
  > | null;

  Tool?: Array<{
    Path: string;
    Version?: string;
  }> | null;
};

const goGetCosmosProto = async (sdkInfo: CosmosSdkInfo): Promise<void> => {
  clog("Grabbing cosmos-sdk proto file locations from disk");

  const { stdout } = await execCommand(
    `(cd "${cfg.dirNibiruRepo}" && go mod edit -json)`,
  );
  const gomod: GoModEditJSON = JSON.parse(stdout);

  // Check replace: gogo → regen
  const hasGogoRegenReplace = (gomod.Replace ?? []).some(
    (r) =>
      r.Old?.Path === "github.com/gogo/protobuf" &&
      r.New?.Path === "github.com/regen-network/protobuf",
  );

  if (!hasGogoRegenReplace) {
    throw new Error(
      "Expected replace github.com/gogo/protobuf => github.com/regen-network/protobuf",
    );
  }

  clog("get protos for: cosmos-sdk, cosmos-proto");
  const cmds = [
    `go get "github.com/cosmos/cosmos-sdk@${sdkInfo.nibiruCosmosSdkVersion}"`,
    `go get github.com/cosmos/cosmos-proto`,
  ];
  for (const cmd of cmds) {
    clogCmd(cmd);
    await execCommand(cmd);
  }

  // // Find the pinned cosmos-sdk version (from Require or Replace.New)
  // const sdkFromRequire = (gomod.Require ?? []).find(
  //   (r) => r.Path === "github.com/cosmos/cosmos-sdk",
  // )?.Version;

  // const sdkFromReplace = (gomod.Replace ?? []).find(
  //   (r) => r.Old.Path === "github.com/cosmos/cosmos-sdk",
  // )?.New.Version;

  // if (sdkFromReplace) {

  // }
};

// Helper: flatten path separators → dots (dir/dir/file.proto → dir.dir.file.proto)
const flatFromRel = (rel: string) => rel.replaceAll("/", ".");

const protoGen = async () => {
  await execCommand(`mkdir -p ${cfg.outOpenapi}`);
  const protoRoot = cfg.dirNibiruProto; // TODO: add comsmos generation next

  const protoFiles = await getProtoServiceFiles(protoRoot);
  clog(`Found ${protoFiles.length} proto service files in ${protoRoot}`);
  protoFiles.forEach((p) => clog(`  ${p}`));

  for (const abs of protoFiles) {
    // Make module-relative path for --path
    // rel:       e.g., nibiru/oracle/v1/query.proto
    // flatNoExt: e.g., nibiru.oracle.v1.query
    // tmpOut:        , plugin always writes here
    // dstOut:    e.g., nibiru.oracle.v1.query.swagger.json
    const rel = relative(protoRoot, abs);
    const flatNoExt = flatFromRel(rel).replace(/\.proto$/, "");
    const tmpOut = join(cfg.outOpenapi, "api.swagger.json");
    const dstOut = join(cfg.outOpenapi, `${flatNoExt}.swagger.json`);

    // Clean any previous tmp
    if (await Bun.file(tmpOut).exists()) {
      await Bun.file(tmpOut).delete();
    }

    // Show exactly what will run
    const cmd = [
      "buf generate .",
      `--path "${rel}"`,
      `--template "${cfg.bufGenYaml}"`,
      `--config "${cfg.bufYaml}"`,
      `-o "${cfg.outPath}"`,
    ].join(" ");
    clogCmd(cmd);

    // Run from module root, important to keep --path in context
    await runAtPath(cmd, cfg.dirNibiruProto);

    // Move the single generated file to unique, flat name
    if (!(await Bun.file(tmpOut).exists())) {
      cerr(`WARN: expected ${tmpOut} not found for ${rel}`);
      continue;
    }
    execCommand(`mv "${tmpOut}" "${dstOut}"`);
    // renameSync(tmpOut, dstOut);
    clog(`Moved -> ${dstOut}`);

    // Quick listing to validate
    await execCommand(`ls -lah "${cfg.outOpenapi}" | sed 's/^/    /'`);
  }

  clog("Done. All per-file outputs in:", cfg.outOpenapi);
  await execCommand(
    `find "${cfg.outOpenapi}" -maxdepth 1 -type f -name '*.swagger.json' -print`,
  );
};

const getProtoServiceFiles = async (protoDir: string): Promise<string[]> => {
  return (
    await execCommand(`
find "${protoDir}" \
  -type f \\( -name 'query.proto' -o -name 'service.proto' \\) -print
`)
  ).stdout
    .split("\n")
    .filter(
      (fname) =>
        fname.endsWith("query.proto") || fname.endsWith("service.proto"),
    );
};

// ---------------------------------------------------

const main = async (): Promise<void> => {
  requireTools();
  const sdkInfo = await getCosmosSdkInfo();
  await goGetCosmosProto(sdkInfo);
  clog("%o", { sdkInfo });

  const protoFiles = (
    await execCommand(`
find "$HOME/ki/nibi-chain/proto" \
  -type f \\( -name 'query.proto' -o -name 'service.proto' \\) -print
`)
  ).stdout.split("\n");
  clog("%o", { protoFiles });

  await protoGen();
};

main();
