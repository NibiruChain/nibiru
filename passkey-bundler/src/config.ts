import fs from "fs"
import path from "path"
import { z } from "zod"

import { BundlerConfig, LogLevel } from "./types"

const hexAddress = z.string().regex(/^0x[a-fA-F0-9]{40}$/i, "expected 20-byte hex address")
const privateKey = z.string().regex(/^0x[a-fA-F0-9]{64}$/i, "expected 32-byte hex private key")

const baseSchema = z.object({
  mode: z.enum(["dev", "testnet"]).default("dev"),
  rpcUrl: z.string().min(1).default("http://127.0.0.1:8545"),
  entryPoint: hexAddress,
  chainId: z.bigint({ coerce: true }).default(0n),
  bundlerPrivateKey: privateKey,
  beneficiary: hexAddress.optional(),
  port: z.number().int().positive().default(4337),
  metricsPort: z.number().int().positive().optional(),
  maxBodyBytes: z.number().int().positive().default(1_000_000),
  authRequired: z.boolean().default(false),
  rateLimitPerMinute: z.number().int().nonnegative().default(120),
  apiKeys: z.array(z.string()).default([]),
  maxQueue: z.number().int().positive().default(1000),
  queueConcurrency: z.number().int().positive().default(4),
  logLevel: z.enum(["debug", "info", "warn", "error"]).default("info"),
  enablePasskeyHelpers: z.boolean().default(true),
  dbUrl: z.string().min(1).optional(),
  validationEnabled: z.boolean().default(false),
  gasBumpPercent: z.number().nonnegative().default(15),
  gasBumpWei: z.bigint({ coerce: true }).optional(),
  prefundEnabled: z.boolean().default(true),
  maxPrefundWei: z.bigint({ coerce: true }).default(5_000_000_000_000_000_000n), // 5 ETH
  prefundAllowlist: z.array(hexAddress).default([]),
  submissionTimeoutMs: z.number().int().positive().default(45_000),
  finalityBlocks: z.number().int().positive().default(2),
  receiptLimit: z.number().int().positive().default(1000),
  receiptPollIntervalMs: z.number().int().positive().default(5_000),
})

type RawConfig = Partial<z.input<typeof baseSchema>>

export function loadConfig(): BundlerConfig {
  const fromFile = readConfigFile(process.env.BUNDLER_CONFIG)
  const merged: RawConfig = { ...fromFile, ...envOverrides() }
  const parsed = baseSchema.parse(merged) as BundlerConfig

  let config = parsed
  if (config.mode === "testnet") {
    config = {
      ...config,
      authRequired: merged.authRequired ?? true,
      prefundEnabled: merged.prefundEnabled ?? false,
      enablePasskeyHelpers: merged.enablePasskeyHelpers ?? false,
      validationEnabled: merged.validationEnabled ?? true,
    }

    if (!config.dbUrl) {
      throw new Error("Testnet mode requires persistence (set DB_URL to a sqlite path/URL)")
    }
    if (config.authRequired && config.apiKeys.length === 0) {
      throw new Error("Testnet mode requires API keys (set BUNDLER_API_KEYS) or disable authRequired")
    }
    if (config.prefundEnabled && config.prefundAllowlist.length === 0) {
      throw new Error("Testnet mode with prefundEnabled requires PREFUND_ALLOWLIST (comma-separated sender addresses)")
    }
  }

  return config
}

function envOverrides(): RawConfig {
  const toBool = (value?: string) => {
    if (value === undefined) return undefined
    const normalized = value.trim().toLowerCase()
    if (!normalized) return undefined
    if (["1", "true", "yes", "y", "on"].includes(normalized)) return true
    if (["0", "false", "no", "n", "off"].includes(normalized)) return false
    return undefined
  }
  const toBigInt = (value?: string) => (value ? BigInt(value) : undefined)
  const toInt = (value?: string) => (value ? Number.parseInt(value, 10) : undefined)

  const apiKeys = process.env.BUNDLER_API_KEYS
    ? process.env.BUNDLER_API_KEYS.split(",").map((v) => v.trim()).filter(Boolean)
    : undefined

  return {
    mode: (process.env.BUNDLER_MODE as any) ?? undefined,
    rpcUrl: process.env.RPC_URL ?? process.env.JSON_RPC_ENDPOINT,
    entryPoint: process.env.ENTRY_POINT,
    chainId: process.env.CHAIN_ID ? BigInt(process.env.CHAIN_ID) : undefined,
    bundlerPrivateKey: process.env.BUNDLER_PRIVATE_KEY,
    beneficiary: process.env.BENEFICIARY,
    port: toInt(process.env.BUNDLER_PORT),
    metricsPort: toInt(process.env.METRICS_PORT),
    maxBodyBytes: toInt(process.env.MAX_BODY_BYTES),
    authRequired: toBool(process.env.BUNDLER_REQUIRE_AUTH),
    rateLimitPerMinute: toInt(process.env.RATE_LIMIT),
    apiKeys,
    maxQueue: toInt(process.env.MAX_QUEUE),
    queueConcurrency: toInt(process.env.QUEUE_CONCURRENCY),
    logLevel: (process.env.LOG_LEVEL as LogLevel | undefined) ?? undefined,
    enablePasskeyHelpers: toBool(process.env.ENABLE_PASSKEY_HELPERS),
    dbUrl: process.env.DB_URL,
    validationEnabled: toBool(process.env.VALIDATION_ENABLED),
    gasBumpPercent: process.env.GAS_BUMP ? Number(process.env.GAS_BUMP) : undefined,
    gasBumpWei: toBigInt(process.env.GAS_BUMP_WEI),
    prefundEnabled: toBool(process.env.PREFUND_ENABLED),
    maxPrefundWei: toBigInt(process.env.MAX_PREFUND_WEI),
    prefundAllowlist: process.env.PREFUND_ALLOWLIST
      ? process.env.PREFUND_ALLOWLIST.split(",").map((v) => v.trim()).filter(Boolean)
      : undefined,
    submissionTimeoutMs: toInt(process.env.SUBMISSION_TIMEOUT_MS),
    finalityBlocks: toInt(process.env.FINALITY_BLOCKS),
    receiptLimit: toInt(process.env.RECEIPT_LIMIT),
    receiptPollIntervalMs: toInt(process.env.RECEIPT_POLL_INTERVAL_MS),
  }
}

function readConfigFile(filePath?: string): RawConfig {
  if (!filePath) return {}
  const resolved = path.resolve(filePath)
  if (!fs.existsSync(resolved)) {
    throw new Error(`Config file not found at ${resolved}`)
  }
  const content = fs.readFileSync(resolved, "utf8")
  try {
    return JSON.parse(content)
  } catch (err) {
    throw new Error(`Failed to parse config file ${resolved}: ${(err as Error).message}`)
  }
}
