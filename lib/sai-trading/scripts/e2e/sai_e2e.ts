/**
 * Sai e2e test types, shared config (`baseCfg`), and utilities.
 * Combines: setup_prices types, perp ExecuteMsg types, base config,
 * and common command execution helpers.
 */

import { join } from "path"

/** E2E config */
export const baseCfg = {
  rpcUrl: "http://localhost:8545", // Nibiru EVM RPC endpoint
  keyringBackend: "test",
  chainId: "nibiru-localnet-0",
  signers: {
    /**
     * Address of the "validator" signer that runs the local network.
     * NOTE: Do not use this account in production.
     * */
    valAddr: "nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl",
    /**
     * Primary E2E EVM signer: `new ethers.Wallet(config.signers.evmPrivKey, provider)` in
     * test_SaiEvm.ts, test_simple_evm.ts, and e2e_deploy.ts (SaiEvm deploy, etc.).
     *
     * - EVM hex addr: 0xabbd6cFEd876F518dbdFE9429c4A02479774821c
     * - x/evm linked bech32 (`nibid q evm account <evm>`): nibi14w7kelkcwm633k7la9pfcjszg7thfqsue7fnuv
     * - `e2e_deploy` sends `convert-coin-to-evm` to this address so USDC
     *   lands on the same account tests sign with.
     * - This is NOT the`--from validator` account on the keyring.
     *   Use `signers.valAddr` / `--from validator` for `nibid tx` only.
     *
     * NOTE: Do not use this account in production.
     */
    evmPrivKey:
      "0x387c72b124f8ae4adb478224e6886921c3c51a99ed0e42adfa775caeb34c9569",
    /**
     * Secondary EVM signer used by `test_SaiEvm.ts` for credit-only trading.
     *
     * - EVM hex addr: 0xc83af6eeb848211e7b3376Be629c18911169876c
     * - This account is intentionally not funded with native `unibi` in the
     *   E2E suite. It proves that the zero-gas SaiEvm registration lets a fresh
     *   account spend trading credits without receiving normal collateral or gas.
     *
     * NOTE: Public localnet-only test key. Do not use in production.
     */
    creditUserEvmPrivKey:
      "0x59c6995e998f97a5a0044966f094538b29283e765f1076b148c6e5fb3310c7e8",
    valMnemonic:
      "guard cream sadness conduct invite crumble clock pudding hole grit liar hotel maid produce squeeze return argue turtle know drive eight casino maze host",
  },

  cacheDir: join(__dirname, ".cache"),
  envFile: join(__dirname, ".cache/localnet_contracts.env"),

  artifactsDir: join(__dirname, "../../artifacts"),
  txArgsDir: join(__dirname, "tx_args"),
  scriptDir: join(__dirname),

  initPerpFile: join(__dirname, "tx_args/init_perp.json"),
  initVaultFile: join(__dirname, "tx_args/init_vault.json"),
  initOracleFile: join(__dirname, "tx_args/init_oracle.json"),
  initVaultTokenMinterFile: join(
    __dirname,
    "tx_args/init_vault_token_minter.json",
  ),
}

/** Oracle and setup_prices types (mirrors scripts/e2e/tx_args/setup_prices.json) */

/** Oracle set_price message (token_id + price_usd) */
export interface SetPriceMsg {
  set_price: {
    token_id: number
    price_usd: string
  }
}

/** Oracle create_token message */
export interface CreateTokenMsg {
  create_token: {
    base: string
    permission_group: number
  }
}

/** Single message in the tx body */
export interface SetupPricesMessage {
  "@type": string
  sender: string
  contract: string
  msg: SetPriceMsg | CreateTokenMsg
  funds: unknown[]
}

/** Root structure of setup_prices.json */
export interface SetupPricesJson {
  body: {
    messages: SetupPricesMessage[]
    memo: string
    timeout_height: string
    extension_options: unknown[]
    non_critical_extension_options: unknown[]
  }
  auth_info: unknown
  signatures: unknown[]
}

export function isSetPriceMessage(
  msg: SetupPricesMessage,
): msg is SetupPricesMessage & { msg: SetPriceMsg } {
  return "set_price" in (msg?.msg ?? {})
}

/** Perp ExecuteMsg types (mirrors contracts/perp/src/msgs.rs) */

/** Trade type: market order, limit order, or stop order */
export type TradeType = "trade" | "limit" | "stop"

/** OpenTrade message args (ExecuteMsg::OpenTrade) */
export interface OpenTradeArgs {
  market_index: string
  leverage: string
  long: boolean
  collateral_index: string
  trade_type: TradeType
  open_price: string
  tp?: string | null
  sl?: string | null
  slippage_p: string
  is_evm_origin: boolean
  client_origin?: string
  return_to_deposits?: boolean
  collateral_amount: string
}

/** ExecuteMsg payload for open_trade */
export interface OpenTradeMsg {
  open_trade: OpenTradeArgs
}

type LoggerFn = (msg: string, ...args: unknown[]) => void

/**
 * Logger callbacks are injected by each script so command logs keep
 * the local per-file logger context (e.g. `newClog` prefixes).
 */
export interface ExecCommandLogger {
  clogCmd: LoggerFn
  cerr: LoggerFn
}

export interface ExecCommandOptions {
  logCommand?: boolean
  logStderr?: boolean
  logFailure?: boolean
}

export interface TxResRaw {
  height?: string
  txhash: string
  codespace?: string
  code: number
  raw_log: string
  events?: Array<{
    type: string
    attributes: Array<{ key: string; value: string }>
  }>
  gas_wanted?: string
  gas_used?: string
  tx?: {
    "@type": string
    body: { messages: Object[] }
  }
  [key: string]: unknown
}

type SequenceMismatch = {
  expected: string
  got: string
}

/**
 * Execute a shell command using Bun and return collected stdout/stderr.
 * Throws when the command exits non-zero.
 */
export async function execCommand(
  command: string,
  logger: ExecCommandLogger,
  options: ExecCommandOptions = {},
): Promise<{ stdout: string; stderr: string }> {
  const logCommand = options.logCommand ?? true
  const logStderr = options.logStderr ?? true
  const logFailure = options.logFailure ?? true

  try {
    if (logCommand) {
      logger.clogCmd(command)
    }
    const proc = Bun.spawn(["bash", "-c", command], {
      env: process.env as Record<string, string>,
      stdout: "pipe",
      stderr: "pipe",
    })

    const stdout = await new Response(proc.stdout).text()
    const stderr = await new Response(proc.stderr).text()

    if (logStderr && stderr && stderr.trim() !== "") {
      logger.cerr(`Command stderr: ${stderr}`)
    }

    const exitCode = await proc.exited

    if (exitCode !== 0) {
      throw new Error(
        `Command exited with code ${exitCode}: ${stderr || stdout || command}`,
      )
    }

    return { stdout, stderr }
  } catch (error) {
    if (logFailure) {
      logger.cerr(`Command execution failed: ${error}`)
    }
    throw error
  }
}

export interface WaitForNibidTxOptions {
  initialDelayMs?: number
  attempts?: number
  queryNode?: string
  timeoutMs?: number
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

function buildQueryTxCommand(txhash: string, queryNode: string): string {
  return `nibid query tx --type hash ${txhash} --node ${queryNode} -o json`
}

function normalizeNode(queryNode: string): string {
  return queryNode.endsWith("/") ? queryNode.slice(0, -1) : queryNode
}

function parseNumberAfter(raw: string, marker: string): string | null {
  const start = raw.indexOf(marker)
  if (start < 0) {
    return null
  }
  let cursor = start + marker.length
  let end = cursor
  while (end < raw.length && raw[end] >= "0" && raw[end] <= "9") {
    end++
  }
  if (end === cursor) {
    return null
  }
  return raw.slice(cursor, end)
}

export function parseAccountSequenceMismatch(
  rawLog: string,
): SequenceMismatch | null {
  if (!rawLog.includes("account sequence mismatch")) {
    return null
  }
  const expected = parseNumberAfter(rawLog, "expected ")
  const got = parseNumberAfter(rawLog, "got ")
  if (!expected || !got) {
    return null
  }
  return { expected, got }
}

function asString(value: unknown): string | null {
  if (typeof value === "string") {
    return value
  }
  if (typeof value === "number" && Number.isFinite(value)) {
    return String(value)
  }
  return null
}

function readJsonPath(root: unknown, path: string[]): unknown {
  let node: unknown = root
  for (const key of path) {
    if (!node || typeof node !== "object" || !(key in node)) {
      return undefined
    }
    node = (node as Record<string, unknown>)[key]
  }
  return node
}

export async function latestBlockHeight(queryNode: string): Promise<number> {
  const statusUrl = `${normalizeNode(queryNode)}/status`
  const response = await fetch(statusUrl)
  if (!response.ok) {
    throw new Error(`failed status query ${statusUrl}: HTTP ${response.status}`)
  }
  const statusJson = (await response.json()) as unknown
  const heightRaw = readJsonPath(statusJson, [
    "result",
    "sync_info",
    "latest_block_height",
  ])
  const height = Number(asString(heightRaw))
  if (!Number.isFinite(height) || height <= 0) {
    throw new Error(`unexpected latest_block_height from ${statusUrl}`)
  }
  return height
}

async function waitForHeightWithTimeout(
  queryNode: string,
  minHeight: number,
  timeoutMs: number,
): Promise<number> {
  const deadline = Date.now() + timeoutMs
  let lastHeight = await latestBlockHeight(queryNode)
  if (lastHeight >= minHeight) {
    return lastHeight
  }
  while (Date.now() < deadline) {
    await sleep(1000)
    lastHeight = await latestBlockHeight(queryNode)
    if (lastHeight >= minHeight) {
      return lastHeight
    }
  }
  throw new Error("timeout exceeded waiting for localnet block")
}

export async function waitForNextBlockVerbose(
  queryNode: string,
  timeoutMs: number = 5 * 60 * 1000,
): Promise<number> {
  const lastBlock = await latestBlockHeight(queryNode)
  const nextBlock = lastBlock + 1
  await waitForHeightWithTimeout(queryNode, nextBlock, timeoutMs)
  return nextBlock
}

export async function waitForNextBlock(
  queryNode: string,
  timeoutMs: number = 5 * 60 * 1000,
): Promise<void> {
  await waitForNextBlockVerbose(queryNode, timeoutMs)
}

function parseTxResponse(stdout: string, command: string): TxResRaw {
  let parsed: unknown
  try {
    parsed = JSON.parse(stdout)
  } catch (error) {
    throw new Error(`failed to parse tx response for ${command}: ${error}`)
  }
  if (!parsed || typeof parsed !== "object") {
    throw new Error(`unexpected tx response shape for ${command}`)
  }
  const txResp = parsed as TxResRaw
  if (typeof txResp.txhash !== "string") {
    throw new Error(`missing txhash in tx response for ${command}`)
  }
  if (typeof txResp.code !== "number") {
    throw new Error(`missing code in tx response for ${command}`)
  }
  if (typeof txResp.raw_log !== "string") {
    txResp.raw_log = ""
  }
  return txResp
}

async function queryAuthAccountNumber(
  fromAddress: string,
  logger: ExecCommandLogger,
  queryNode: string,
): Promise<string> {
  const queryAccountCmd = `nibid q auth account ${fromAddress} --node ${queryNode} -o json`
  const { stdout } = await execCommand(queryAccountCmd, logger)
  let parsed: unknown
  try {
    parsed = JSON.parse(stdout)
  } catch (error) {
    throw new Error(`failed to parse auth account query: ${error}`)
  }
  const candidates = [
    readJsonPath(parsed, ["account", "base_account", "account_number"]),
    readJsonPath(parsed, [
      "account",
      "base_vesting_account",
      "base_account",
      "account_number",
    ]),
    readJsonPath(parsed, ["account", "account_number"]),
    readJsonPath(parsed, ["account_number"]),
  ]
  for (const candidate of candidates) {
    const accountNumber = asString(candidate)
    if (accountNumber != null) {
      return accountNumber
    }
  }
  throw new Error(`failed to resolve account_number for ${fromAddress}`)
}

function ensureBroadcastModeSync(command: string): string {
  if (/\s--broadcast-mode(?:=|\s)/.test(command)) {
    return command
  }
  return `${command} --broadcast-mode=sync`
}

/** Return true when a wasm contract is already instantiated at `address`. */
export async function wasmContractExists(
  address: string,
  queryNode = "http://localhost:26657",
): Promise<boolean> {
  const silentLogger = { clogCmd: () => {}, cerr: () => {} }
  try {
    await execCommand(
      `nibid q wasm contract ${address} --node ${queryNode} -o json`,
      silentLogger,
      { logCommand: false, logFailure: false, logStderr: false },
    )
    return true
  } catch {
    return false
  }
}

export async function waitForNibidTx(
  txhash: string,
  logger: ExecCommandLogger & { clog: LoggerFn },
  options: WaitForNibidTxOptions = {},
): Promise<string> {
  const normalizedTxHash = txhash.trim()
  if (!normalizedTxHash) {
    throw new Error("waitForNibidTx received an empty txhash")
  }

  const attempts = options.attempts ?? 3
  const initialDelayMs = options.initialDelayMs ?? 0
  const queryNode = options.queryNode ?? "http://localhost:26657"
  const timeoutMs = options.timeoutMs ?? 5 * 60 * 1000

  logger.clog(`Waiting for transaction ${normalizedTxHash} to be mined...`)
  if (initialDelayMs > 0) {
    await sleep(initialDelayMs)
  }

  let lastErr: unknown
  const queryTxCommand = buildQueryTxCommand(normalizedTxHash, queryNode)
  for (let attempt = 0; attempt < attempts; attempt++) {
    try {
      const { stdout } = await execCommand(queryTxCommand, logger, {
        logCommand: false,
        logStderr: false,
        logFailure: false,
      })
      const txResp = parseTxResponse(stdout, queryTxCommand)
      return JSON.stringify(txResp)
    } catch (error) {
      lastErr = error
    }

    if (attempt < attempts - 1) {
      try {
        await waitForNextBlock(queryNode, timeoutMs)
      } catch (waitError) {
        throw new Error(
          `failed waiting for tx ${normalizedTxHash} block inclusion: ${waitError}`,
        )
      }
    }
  }

  throw new Error(
    `failed to query tx ${normalizedTxHash} after ${attempts} attempts ` +
      `and ${Math.max(attempts - 1, 0)} block waits: ${lastErr}`,
  )
}

export interface ExecNibidTxOptions {
  fromAddress: string
  queryNode?: string
  queryInitialDelayMs?: number
  queryAttempts?: number
  requireDeliveredSuccess?: boolean
}

export async function execNibidTx(
  command: string,
  logger: ExecCommandLogger & { clog: LoggerFn },
  options: ExecNibidTxOptions,
): Promise<TxResRaw> {
  const queryNode = options.queryNode ?? "http://localhost:26657"
  const fromAddress = options.fromAddress.trim()
  if (!fromAddress) {
    throw new Error("execNibidTx requires a non-empty fromAddress")
  }

  const baseCommand = ensureBroadcastModeSync(command)
  let txResp: TxResRaw | null = null
  let txCommand = baseCommand
  for (let attempt = 0; attempt < 3; attempt++) {
    const { stdout } = await execCommand(txCommand, logger)
    txResp = parseTxResponse(stdout, txCommand)
    if (txResp.code === 0) {
      break
    }

    const mismatch = parseAccountSequenceMismatch(txResp.raw_log)
    if (!mismatch || attempt === 2) {
      throw new Error(
        `tx failed for ${txCommand} with code ${txResp.code}: ${txResp.raw_log}`,
      )
    }

    const accountNumber = await queryAuthAccountNumber(
      fromAddress,
      logger,
      queryNode,
    )
    txCommand = `${baseCommand} --offline=true --account-number=${accountNumber} --sequence=${mismatch.expected}`
  }

  if (!txResp) {
    throw new Error(`failed to execute tx command: ${baseCommand}`)
  }
  if (!txResp.txhash) {
    return txResp
  }

  let deliveredResp: TxResRaw
  try {
    const deliveredStdout = await waitForNibidTx(txResp.txhash, logger, {
      queryNode,
      initialDelayMs: options.queryInitialDelayMs ?? 0,
      attempts: options.queryAttempts,
    })
    deliveredResp = parseTxResponse(
      deliveredStdout,
      buildQueryTxCommand(txResp.txhash, queryNode),
    )
  } catch (error) {
    throw new Error(
      `failed waiting for tx ${txResp.txhash} (${baseCommand}): ${error}`,
    )
  }

  if (deliveredResp.code !== 0 && options.requireDeliveredSuccess !== false) {
    throw new Error(
      `delivered tx failed for ${baseCommand} with code ${deliveredResp.code}: ${deliveredResp.raw_log}`,
    )
  }
  return deliveredResp
}
