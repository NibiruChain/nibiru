#!/usr/bin/env bun
/**
 * OpenAPI/Swagger Generator for Nibiru gRPC-Gateway
 *
 * Overview
 * - Discovers proto service entrypoints (query.proto / service.proto).
 * - Runs `buf generate` with `proto/buf-gen-swagger.yaml` per file.
 * - Writes per-service Swagger JSON to `dist/openapi/*.swagger.json`.
 * - Infers the pinned Cosmos SDK version from go.mod and ensures proto deps.
 *
 * Prerequisites
 * - Tools on PATH: `go`, `buf`, `jq`, `bun`.
 * - Protoc plugins available (declared in go.mod `tool` block):
 *   - protoc-gen-go, protoc-gen-go-grpc
 *   - protoc-gen-grpc-gateway, protoc-gen-swagger (openapiv2)
 *   - protoc-gen-openapiv2, protoc-gen-gocosmos
 *
 * Quick Start
 *   $ just gen-proto-openapi
 *   # or
 *   $ bun proto/buf-gen-swagger.ts
 *
 * Inputs & Config
 * - Proto root:       ./proto
 * - Buf config:       ./proto/buf.yaml    (Cosmos deps; version pins)
 * - Buf template:     ./proto/buf-gen-swagger.yaml (swagger plugin options)
 * - Output root:      ./dist/openapi
 * - Temporary file:   ./dist/openapi/api.swagger.json (renamed per target)
 *
 * Outputs
 * - Per-service JSON: dist/openapi/<pkg>.<version>.<name>.swagger.json
 *   e.g. `nibiru.oracle.v1.query.swagger.json`
 *
 * What This Script Does
 * 1) Validates required tools.
 * 2) Reads go.mod to locate the pinned Cosmos SDK version.
 * 3) `go get` ensures cosmos-sdk / cosmos-proto proto availability.
 * 4) Finds all `query.proto`/`service.proto` under `./proto`.
 * 5) Runs `buf generate` per file and moves the merged output to a unique name.
 *
 * Extending
 * - To include additional entrypoints, tweak `getProtoServiceFiles` or
 *   update `buf-gen-swagger.yaml` options (e.g., tags, enums_as_ints).
 **/
import { join, relative } from "path"
import { bash, type BashOut } from "@uniquedivine/bash"
import { newClog } from "@uniquedivine/jiyuu"
import Bun from "bun"

type CosmosSdkInfo = {
	cosmosSdkGhPath: string
	nibiruCosmosSdkVersion: string
	cosmosSdkProtoDir: string
}

const cfg = (() => {
	// Bun exposes import.meta.dir for module directory
	const moduleDir = (import.meta as unknown as { dir?: string }).dir ?? "."
	const dirNibiruRepo = join(moduleDir, "..")
	const nibiru = (relPath: string) => join(dirNibiruRepo, relPath)
	const obj = {
		dirNibiruRepo,
		dirNibiruProto: nibiru("proto"),
		outPath: nibiru("dist"),
		outOpenapi: nibiru("dist/openapi"),
		bufYaml: nibiru("proto/buf.yaml"),
		bufGenYaml: nibiru("proto/buf-gen-swagger.yaml"),
	}
	return obj
})()

const moduleUrl: string = ((import.meta as unknown as { url?: string })?.url ?? "buf-gen-swagger.ts")
const { clog, cerr, clogCmd } = newClog(
	moduleUrl.includes("/") ? moduleUrl.split("/").pop()! : moduleUrl,
)

const requireTools = (): void => {
	const tools = ["go", "buf", "jq", "bun"]

	for (const [_, tool] of tools.entries()) {
		if (Bun.which(tool) == null) {
			throw new Error(`Tool "${tool}" is missing and not on the $PATH`)
		}
	}
}

// ---------------------------------------------------

/** Runs a command with the working directory as the Nibiru repo. */
const runAtPath = async (cmd: string, path: string): Promise<BashOut> => {
	const wrapped = `( cd "${path}" && ${cmd} )`
	clogCmd(wrapped)
	return await bash(wrapped)
}

async function getCosmosSdkInfo(): Promise<CosmosSdkInfo> {
	// repo + proto roots inferred from your cfg
	const protoDir = cfg.dirNibiruProto

	if (!Bun.file(protoDir).exists()) {
		throw new Error(`proto dir not found: ${protoDir}`)
	}

	// run the go query from inside the repo root (matches bash)
	const cmd = `( cd "${cfg.dirNibiruRepo}" && go list -f '{{ .Dir }}' -m github.com/cosmos/cosmos-sdk )`
	clogCmd(cmd)
	const { stdout } = await bash(cmd)

	const cosmosSdkGhPath = stdout.trim()
	const atIdx = cosmosSdkGhPath.lastIndexOf("@")
	const nibiruCosmosSdkVersion =
		atIdx >= 0 ? cosmosSdkGhPath.slice(atIdx + 1) : ""
	const cosmosSdkProtoDir = join(cosmosSdkGhPath, "proto")

	if (!nibiruCosmosSdkVersion) {
		throw new Error(
			`Could not parse Cosmos SDK version from: ${cosmosSdkGhPath}`,
		)
	}

	// trace for sanity
	clog("protoDir:         ", protoDir)
	clog("cosmosSdkGhPath:  ", cosmosSdkGhPath)
	clog("sdk version:      ", nibiruCosmosSdkVersion)
	clog("nibiruRepoPath:   ", cfg.dirNibiruRepo)
	clog("\n%o", { cfg })

	return {
		cosmosSdkGhPath,
		nibiruCosmosSdkVersion,
		cosmosSdkProtoDir,
	}
}

export type GoModEditJSON = {
	Module?: { Path: string }
	Go?: string

	Require?: Array<{
		Path: string
		Version?: string
		Indirect?: boolean
	}> | null

	Exclude?: Array<{
		Path: string
		Version?: string
	}> | null

	Replace?: Array<{
		Old: { Path: string; Version?: string }
		New: { Path: string; Version?: string }
	}> | null

	Retract?: Array<
		| string
		| {
			Low: string
			High?: string
			Reason?: string
		}
	> | null

	Tool?: Array<{
		Path: string
		Version?: string
	}> | null
}

const goGetCosmosProto = async (sdkInfo: CosmosSdkInfo): Promise<void> => {
	clog("Grabbing cosmos-sdk proto file locations from disk")

	const { stdout } = await bash(
		`(cd "${cfg.dirNibiruRepo}" && go mod edit -json)`,
	)
	const gomod: GoModEditJSON = JSON.parse(stdout)

	// Check replace: gogo → regen
	const hasGogoRegenReplace = (gomod.Replace ?? []).some(
		(r) =>
			r.Old?.Path === "github.com/gogo/protobuf" &&
			r.New?.Path === "github.com/regen-network/protobuf",
	)

	if (!hasGogoRegenReplace) {
		throw new Error(
			"Expected replace github.com/gogo/protobuf => github.com/regen-network/protobuf",
		)
	}

	clog("get protos for: cosmos-sdk, cosmos-proto")
	const cmds = [
		`go get "github.com/cosmos/cosmos-sdk@${sdkInfo.nibiruCosmosSdkVersion}"`,
		`go get github.com/cosmos/cosmos-proto`,
	]
	for (const cmd of cmds) {
		clogCmd(cmd)
		await runAtPath(cmd, cfg.dirNibiruRepo)
	}
}

// Helper: flatten path separators → dots (dir/dir/file.proto → dir.dir.file.proto)
const flatFromRel = (rel: string) => rel.replaceAll("/", ".")

// Limit Cosmos SDK generation to selected services only
const COSMOS_ALLOWLIST_REL: string[] = [
	"cosmos/bank/v1beta1/query.proto",
	"cosmos/auth/v1beta1/query.proto",
	"cosmos/tx/v1beta1/service.proto",
]

const resolveExistingPaths = async (
	rootDir: string,
	relPaths: string[],
): Promise<string[]> => {
	const out: string[] = []
	for (const rel of relPaths) {
		const abs = join(rootDir, rel)
		if (await Bun.file(abs).exists()) {
			out.push(abs)
		}
	}
	return out
}

const protoGen = async (sdkInfo: CosmosSdkInfo) => {
	await bash(`mkdir -p ${cfg.outOpenapi}`)
	const nibiruRoot = cfg.dirNibiruProto
	const cosmosRoot = sdkInfo.cosmosSdkProtoDir

	const nibiruFiles = await getProtoServiceFiles(nibiruRoot)
	const cosmosFiles = await resolveExistingPaths(cosmosRoot, COSMOS_ALLOWLIST_REL)
	const protoFiles = Array.from(new Set([...nibiruFiles, ...cosmosFiles]))
	clog(
		`Found ${protoFiles.length} proto service files (nibiru: ${nibiruFiles.length}, cosmos-allowlisted: ${cosmosFiles.length})`,
	)
	clog("Cosmos allowlist:")
	COSMOS_ALLOWLIST_REL.forEach((p) => clog(`  ${join(cosmosRoot, p)}`))
	clog("Resolved files:")
	protoFiles.forEach((p) => clog(`  ${p}`))

	for (const abs of protoFiles) {
		// Make module-relative path for --path
		// rel:       e.g., nibiru/oracle/v1/query.proto
		// flatNoExt: e.g., nibiru.oracle.v1.query
		// tmpOut:        , plugin always writes here
		// dstOut:    e.g., nibiru.oracle.v1.query.swagger.json
		const roots = [nibiruRoot, cosmosRoot]
		const baseRoot = roots.find((r) => abs.startsWith(r)) ?? nibiruRoot
		const rel = relative(baseRoot, abs)
		const flatNoExt = flatFromRel(rel).replace(/\.proto$/, "")
		const tmpOut = join(cfg.outOpenapi, "api.swagger.json")
		const dstOut = join(cfg.outOpenapi, `${flatNoExt}.swagger.json`)

		// Clean any previous tmp
		if (await Bun.file(tmpOut).exists()) {
			await Bun.file(tmpOut).delete()
		}

		// Show exactly what will run
		const cmd = [
			"buf generate .",
			`--path "${rel}"`,
			`--template "${cfg.bufGenYaml}"`,
			`--config "${cfg.bufYaml}"`,
			`-o "${cfg.outPath}"`,
		].join(" ")
		clogCmd(cmd)

		// Run from module root, important to keep --path in context
		await runAtPath(cmd, baseRoot)

		// Move the single generated file to unique, flat name
		if (!(await Bun.file(tmpOut).exists())) {
			cerr(`WARN: expected ${tmpOut} not found for ${rel}`)
			continue
		}
		await bash(`mv "${tmpOut}" "${dstOut}"`)
		clog(`Moved -> ${dstOut}`)

		// Quick listing to validate
		await bash(`ls -lah "${cfg.outOpenapi}" | sed 's/^/    /'`)
	}

	clog("Done. All per-file outputs in:", cfg.outOpenapi)
	await bash(
		`find "${cfg.outOpenapi}" -maxdepth 1 -type f -name '*.swagger.json' -print`,
	)
}

const getProtoServiceFiles = async (
	...protoDirs: string[]
): Promise<string[]> => {
	const results: string[] = []
	for (const dir of protoDirs) {
		const out = (
			await bash(`
find "${dir}" \
  -type f \\( -name 'query.proto' -o -name 'service.proto' \\) -print
`)
		).stdout
			.split("\n")
			.filter(
				(fname) =>
					fname.endsWith("query.proto") || fname.endsWith("service.proto"),
			)
		results.push(...out)
	}
	// de-dup in case paths overlap
	return Array.from(new Set(results)).filter(Boolean)
}

// ---------------------------------------------------

const main = async (): Promise<void> => {
	requireTools()
	const sdkInfo = await getCosmosSdkInfo()
	await goGetCosmosProto(sdkInfo)
	clog("%o", { sdkInfo })

	await protoGen(sdkInfo)
}

main()
