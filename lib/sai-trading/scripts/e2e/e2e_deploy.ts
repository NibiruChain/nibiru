#!/usr/bin/env bun
import { existsSync, mkdirSync, readFileSync, writeFileSync } from "fs"
import { join } from "path"
import { newClog } from "@uniquedivine/jiyuu"
import { ethers } from "ethers"

import { loadSaiEvmArtifact } from "./evm_artifacts"
import {
  baseCfg,
  execCommand,
  execNibidTx,
  waitForNibidTx,
  wasmContractExists,
  type SetupPricesJson,
  type TxResRaw,
} from "./sai_e2e"

const { clog, cerr, clogCmd } = newClog(
  import.meta.url.includes("/")
    ? import.meta.url.split("/").pop()!
    : import.meta.url,
)

// Config for localnet
const config = {
  fees: "750000unibi",
  gas: "30000000",
  ...baseCfg,
}

const DECIMALS_6 = 6
const TEST_USDC_DENOM = "tf/nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl/usdc"
const ONE_THOUSAND_UNIBI = `${1000}${"0".repeat(DECIMALS_6)}unibi`
const ONE_THOUSAND_USDC_BASE_UNITS = `${1000}${"0".repeat(DECIMALS_6)}`
const TEN_THOUSAND_USDC_BASE_UNITS = `${10000}${"0".repeat(DECIMALS_6)}`
const ONE_HUNDRED_THOUSAND_USDC_BASE_UNITS = `${100000}${"0".repeat(
  DECIMALS_6,
)}`
const CONTRACT_DEPLOY_QUERY_ATTEMPTS = 7
// Fixed salt for CosmWasm instantiate2. Changing this changes the predicted
// vault-token-minter address and therefore the tokenfactory share denom.
const VAULT_TOKEN_MINTER_SALT_HEX = Buffer.from(
  "vault-token-minter-e2e-1",
).toString("hex")

/**
 * Keep per-script logging prefixes while using the shared command runner.
 */
const execCmd = (command: string) => execCommand(command, { clogCmd, cerr })

type TxResLite = Omit<TxResRaw, "tx"> & {
  "tx.body.messages": { messages: Object[] }
  "tx.@type": string
}

const execValidatorTx = async ({
  command,
  requireDeliveredSuccess = true,
  queryAttempts,
}: {
  command: string
  requireDeliveredSuccess?: boolean
  queryAttempts?: number
}): Promise<TxResRaw> => {
  return execNibidTx(
    command,
    { clog, clogCmd, cerr },
    {
      fromAddress: config.signers.valAddr,
      queryNode: "http://localhost:26657",
      queryAttempts,
      requireDeliveredSuccess,
    },
  )
}

async function broadcastSignedTx(command: string): Promise<TxResRaw> {
  const { stdout } = await execCmd(command)
  const broadcastResult = JSON.parse(stdout) as TxResRaw
  if (broadcastResult.code !== 0 || !broadcastResult.txhash) {
    return broadcastResult
  }

  const deliveredStdout = await waitForNibidTx(
    broadcastResult.txhash,
    { clog, clogCmd, cerr },
    { queryNode: "http://localhost:26657" },
  )
  return JSON.parse(deliveredStdout) as TxResRaw
}

type ContractName = "PERP" | "VAULT" | "VAULT_TOKEN_MINTER" | "ORACLE"

// Extract code ID from store code response
async function getCodeId(
  jsonResult: TxResRaw,
  contractName: ContractName,
): Promise<string> {
  try {
    const events = jsonResult.events || []

    for (const event of events) {
      if (event.type === "store_code") {
        for (const attribute of event.attributes) {
          if (attribute.key === "code_id") {
            return attribute.value
          }
        }
      }
    }

    cerr("Failed to extract code_id: %o", {
      contractName: contractName,
      txRes: txResultLite(jsonResult),
    })
    const reasonSuffix: string =
      jsonResult.code !== 0 ? ` (${jsonResult.raw_log} )` : ""
    throw new Error(`Exit: reason=getCodeId${reasonSuffix}`)
  } catch (error) {
    cerr(`Failed to parse result or extract code_id: ${error}`)
    throw error
  }
}

const txResultLite = (res: TxResRaw): TxResLite => {
  const txBody = res.tx?.body ?? { messages: [] }
  const txType = res.tx?.["@type"] ?? ""
  return {
    // Tx connection to the block
    height: res.height,
    txhash: res.txhash,

    // Most important details
    code: res.code,
    raw_log: res.raw_log,
    "tx.body.messages": txBody,
    events: res.events || [],

    // Other concise info for debugging
    gas_wanted: res.gas_wanted,
    gas_used: res.gas_used,
    "tx.@type": txType,
    codespace: res.codespace,
  }
}

/**
 * getDenomTF: Extract token factory denom for the local USDC fixture.
 * */
async function getDenomTF(jsonResult: TxResRaw): Promise<string> {
  try {
    const events = jsonResult.events || []

    for (const event of events) {
      if (event.type === "nibiru.tokenfactory.v1.EventCreateDenom") {
        for (const attribute of event.attributes) {
          if (attribute.key === "denom") {
            // Return the raw string without additional quotes
            return attribute.value
          }
        }
      }
    }

    // Fallback: Query token factory for denoms created by validator
    clog("Event parsing failed, falling back to querying token factory...")
    const queryCmd = `nibid query tokenfactory denoms ${config.signers.valAddr} --node http://localhost:26657`
    const queryResult = await execCmd(queryCmd)
    const denomsData: { denoms: string[] } = JSON.parse(queryResult.stdout)

    // Case: False negative. USDC may already exist.
    const maybeDenom = denomsData.denoms.find((d) => d.includes("usdc"))
    if (maybeDenom) {
      return maybeDenom
    }

    const denoms = denomsData.denoms || []
    cerr("Failed to extract token factory denom: %o", {
      creator: config.signers.valAddr,
      tfDenomsFromCreator: denoms,
      txRes: txResultLite(jsonResult),
    })
    const reasonSuffix: string =
      jsonResult.code !== 0 ? ` (${jsonResult.raw_log} )` : ""
    throw new Error(`Exit: reason=getDenomTF${reasonSuffix}`)
  } catch (error) {
    cerr(`Failed to parse result or extract denom: ${error}`)
    throw error
  }
}

// Extract contract address from instantiate response
async function getContractAddress(jsonResult: TxResRaw): Promise<string> {
  try {
    const events = jsonResult.events || []

    for (const event of events) {
      if (event.type === "instantiate") {
        for (const attribute of event.attributes) {
          if (attribute.key === "_contract_address") {
            return attribute.value
          }
        }
      }
    }

    cerr("%o", {
      creator: config.signers.valAddr,
      txRes: txResultLite(jsonResult),
    })
    const reasonSuffix: string =
      jsonResult.code !== 0 ? ` (${jsonResult.raw_log} )` : ""
    throw new Error(`Exit: reason=getContractAddress${reasonSuffix}`)
  } catch (error) {
    cerr(`Failed to parse result or extract contract address: ${error}`)
    throw error
  }
}

// Ensure transaction succeeded
function ensureTxOk(result: string | object): void {
  let jsonResult: TxResRaw
  if (typeof result === "string") {
    jsonResult = JSON.parse(result)
  } else {
    jsonResult = result as unknown as TxResRaw
  }
  try {
    const rawLog = jsonResult.raw_log
    if (rawLog.includes("failed to execute message")) {
      cerr("Transaction failed with detailed error:")
      cerr("Raw log:", rawLog)
      cerr("Full transaction result:", JSON.stringify(jsonResult, null, 2))
      throw new Error(`Transaction failed: ${rawLog}`)
    }
    if (jsonResult.code !== 0) {
      cerr("Transaction failed with code:", jsonResult.code)
      cerr("Raw log:", rawLog)
      cerr("Full transaction result:", JSON.stringify(jsonResult, null, 2))
      throw new Error(
        `Transaction failed with code ${jsonResult.code}: ${rawLog}`,
      )
    }
  } catch (error) {
    cerr(`Failed to check transaction result: ${error}`)
    throw error
  }
}

type Bech32AddrEOA = `nibi1${string}`

/**
 * Resolve bech32 address from EVM address via `nibid q evm account` (Nibiru CLI).
 * Uses --offline so it works for brand-new accounts not yet onchain.
 */
async function getBech32FromEvmAddress(
  evmAddr: string,
): Promise<Bech32AddrEOA> {
  const queryCmd = `nibid q evm account ${evmAddr} --offline --output json`
  const { stdout } = await execCmd(queryCmd)
  const parsed = JSON.parse(stdout) as {
    eth_address?: `0x${string}`
    bech32_address?: Bech32AddrEOA
  }
  const bech32 = parsed.bech32_address
  if (!bech32 || typeof bech32 !== "string") {
    throw new Error(
      `Failed to get bech32 from evm account ${evmAddr}: missing bech32_address`,
    )
  }
  return bech32
}

// Store environment variables
const contractAddresses: Record<string, string> = {}

function storeContractAddress(name: string, address: string) {
  contractAddresses[name] = address
}

function saveEnvironmentFile() {
  if (!existsSync(config.cacheDir)) {
    mkdirSync(config.cacheDir, { recursive: true })
  }

  let envContent = ""

  for (const [key, value] of Object.entries(contractAddresses)) {
    envContent += `${key}="${value}"\n`
  }

  writeFileSync(config.envFile, envContent)
  clog(`Contract addresses saved to ${config.envFile}`)
}

// Deploy WASM contracts
async function deployWasmContracts() {
  clog("Starting WASM contract deployment...")

  // Transaction flags
  const TX_FLAGS = `--fees ${config.fees} --gas ${config.gas} --yes --broadcast-mode sync --keyring-backend ${config.keyringBackend} --chain-id ${config.chainId} --node http://localhost:26657`

  if (!existsSync(config.cacheDir)) {
    mkdirSync(config.cacheDir, { recursive: true })
  }

  clog(`Creating USDC collateral`)
  try {
    const tokenFactoryCreation = `nibid tx tokenfactory create-denom usdc --from validator ${TX_FLAGS}`
    const tokenFactoryCreationResult = await execValidatorTx({
      command: tokenFactoryCreation,
      requireDeliveredSuccess: false,
    })
    const collateralDenom = await getDenomTF(tokenFactoryCreationResult)
    storeContractAddress("USDC", collateralDenom)
  } catch (error) {
    cerr("Failed to create USDC:", error)
    clog("Note: If the token mapping already exists, this step can be skipped.")
  } finally {
    const bankDenom = TEST_USDC_DENOM
    const decimals = 6
    const bankMetadata = {
      base: bankDenom,
      display: `decimals_denom_for-${bankDenom}`,
      name: "USD Coin",
      symbol: "USDC",
      description: "Tokens for Sai E2E tests on localnet",
      denom_units: [
        {
          denom: bankDenom,
          exponent: 0,
        },
        {
          denom: `decimals_denom_for-${bankDenom}`,
          exponent: decimals,
        },
      ],
    }
    const fPath = join(config.cacheDir, "metadata.json")
    const f = Bun.file(fPath)
    if (!(await f.exists())) {
      await Bun.write(f, JSON.stringify(bankMetadata, null, 2))
    }
    const setDenomMetadata = `nibid tx tokenfactory set-denom-metadata ${fPath} --from validator ${TX_FLAGS}`
    clogCmd(setDenomMetadata)
    const jsonRes = await execValidatorTx({ command: setDenomMetadata })
    if (jsonRes.code !== 0) {
      const reason = "nibid tx tokenfactory set-denom-metadata"
      cerr(`Exit: reason = ${reason}: (${jsonRes.raw_log})`)
      throw new Error(`Exit: reason = ${reason}`)
    }
  }

  // mint some tokens to validator
  const mintCmd = `nibid tx tokenfactory mint ${ONE_HUNDRED_THOUSAND_USDC_BASE_UNITS}${TEST_USDC_DENOM} --mint-to nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl --from validator ${TX_FLAGS}`
  const mintResult = await execValidatorTx({ command: mintCmd })
  ensureTxOk(mintResult)

  // assert balance is minted
  const balanceCmd = `nibid q bank balances nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl --node http://localhost:26657 | jq -r '
  (.balances[] | select(.denom == "${TEST_USDC_DENOM}").amount) // "0"
'
`
  const balanceResult = await execCmd(balanceCmd)
  clog(`Validator balance: %o`, {
    amount: balanceResult.stdout.trim(),
    token: TEST_USDC_DENOM,
  })

  // Store contracts
  clog("Storing WASM contracts...")

  // Store perp contract
  const perpStoreCmd = `nibid tx wasm store ${config.artifactsDir}/perp.wasm --from validator ${TX_FLAGS}`
  clogCmd(perpStoreCmd)
  const perpStoreResult = await execValidatorTx({
    command: perpStoreCmd,
    queryAttempts: CONTRACT_DEPLOY_QUERY_ATTEMPTS,
  })
  const PERP_CODE_ID = await getCodeId(perpStoreResult, "PERP")
  clog(`Stored perp contract with code ID: ${PERP_CODE_ID}`)
  storeContractAddress("PERP_CODE_ID", PERP_CODE_ID)

  // Store vault contract
  const vaultStoreCmd = `nibid tx wasm store ${config.artifactsDir}/vault.wasm --from validator ${TX_FLAGS}`
  clogCmd(vaultStoreCmd)
  const vaultStoreResult = await execValidatorTx({
    command: vaultStoreCmd,
    queryAttempts: CONTRACT_DEPLOY_QUERY_ATTEMPTS,
  })
  const VAULT_CODE_ID = await getCodeId(vaultStoreResult, "VAULT")
  clog(`Stored vault contract with code ID: ${VAULT_CODE_ID}`)
  storeContractAddress("VAULT_CODE_ID", VAULT_CODE_ID)

  // Store vault token minter contract
  const minterStoreCmd = `nibid tx wasm store ${config.artifactsDir}/vault_token_minter.wasm --from validator ${TX_FLAGS}`
  clogCmd(minterStoreCmd)
  const minterStoreResult = await execValidatorTx({
    command: minterStoreCmd,
    queryAttempts: CONTRACT_DEPLOY_QUERY_ATTEMPTS,
  })
  const VAULT_TOKEN_MINTER_CODE_ID = await getCodeId(
    minterStoreResult,
    "VAULT_TOKEN_MINTER",
  )
  clog(`Stored vault token minter with code ID: ${VAULT_TOKEN_MINTER_CODE_ID}`)
  storeContractAddress("VAULT_TOKEN_MINTER_CODE_ID", VAULT_TOKEN_MINTER_CODE_ID)

  // Store oracle contract
  const oracleStoreCmd = `nibid tx wasm store ${config.artifactsDir}/oracle.wasm --from validator ${TX_FLAGS}`
  clogCmd(oracleStoreCmd)
  const oracleStoreResult = await execValidatorTx({
    command: oracleStoreCmd,
    queryAttempts: CONTRACT_DEPLOY_QUERY_ATTEMPTS,
  })
  const ORACLE_CODE_ID = await getCodeId(oracleStoreResult, "ORACLE")
  clog(`Stored oracle contract with code ID: ${ORACLE_CODE_ID}`)
  storeContractAddress("ORACLE_CODE_ID", ORACLE_CODE_ID)

  // Instantiate contracts
  clog("\nInstantiating contracts...")

  // Instantiate oracle contract
  const oracleInstantiateCmd = `nibid tx wasm instantiate ${ORACLE_CODE_ID} "$(cat ${config.initOracleFile})" --amount ${ONE_THOUSAND_UNIBI} --label "oracle" --admin ${config.signers.valAddr} --from validator ${TX_FLAGS}`
  clogCmd(oracleInstantiateCmd)
  const oracleInstantiateResult = await execValidatorTx({
    command: oracleInstantiateCmd,
    queryAttempts: CONTRACT_DEPLOY_QUERY_ATTEMPTS,
  })
  const ORACLE_ADDR = await getContractAddress(oracleInstantiateResult)
  clog(`Instantiated oracle at: ${ORACLE_ADDR}`)
  storeContractAddress("ORACLE_ADDRESS", ORACLE_ADDR)

  // Build init messages with oracle address (write to .cache, never mutate templates)
  const cacheInitPerpFile = join(config.cacheDir, "init_perp.json")
  const cacheInitVaultFile = join(config.cacheDir, "init_vault.json")

  const perpInitMsg = JSON.parse(readFileSync(config.initPerpFile, "utf8"))
  perpInitMsg.oracle_address = ORACLE_ADDR
  writeFileSync(cacheInitPerpFile, JSON.stringify(perpInitMsg, null, 2))

  const vaultInitMsg = JSON.parse(readFileSync(config.initVaultFile, "utf8"))
  vaultInitMsg.oracle = ORACLE_ADDR
  writeFileSync(cacheInitVaultFile, JSON.stringify(vaultInitMsg, null, 2))

  // Instantiate perp contract
  const perpInstantiateCmd = `nibid tx wasm instantiate ${PERP_CODE_ID} "$(cat ${cacheInitPerpFile})" --amount ${ONE_THOUSAND_UNIBI} --label "perp" --admin ${config.signers.valAddr} --from validator ${TX_FLAGS}`
  const perpInstantiateResult = await execValidatorTx({
    command: perpInstantiateCmd,
    queryAttempts: CONTRACT_DEPLOY_QUERY_ATTEMPTS,
  })
  const PERP_ADDRESS = await getContractAddress(perpInstantiateResult)
  clog(`Instantiated perp contract at: ${PERP_ADDRESS}`)
  storeContractAddress("PERP_ADDRESS", PERP_ADDRESS)

  /**
   * Use CosmWasm instantiate2 for the vault-token-minter.
   *
   * instantiate2 derives the contract address from the code hash, creator
   * address, salt, and (with --fix-msg) init message. That lets this test know
   * the minter address before the contract exists.
   *
   * Nibiru v2.14 restricts tokenfactory denom creation to governance or x/sudo
   * sudoers. The minter creates its share denom during instantiate via a
   * submessage, so the contract address must already be in sudoers before the
   * instantiate tx runs. Predicting the address here lets the validator add that
   * exact future contract to sudoers, then instantiate the contract normally.
   */
  const minterCodeInfo = JSON.parse(
    (
      await execCmd(
        `nibid q wasm code-info ${VAULT_TOKEN_MINTER_CODE_ID} --node http://localhost:26657 -o json`,
      )
    ).stdout,
  ) as { data_hash: string }
  const predictedMinterAddr = (
    await execCmd(
      `nibid q wasm build-address ${minterCodeInfo.data_hash} ${config.signers.valAddr} ${VAULT_TOKEN_MINTER_SALT_HEX} "$(cat ${config.initVaultTokenMinterFile})"`,
    )
  ).stdout.trim()

  let VAULT_TOKEN_MINTER_ADDR: string
  if (await wasmContractExists(predictedMinterAddr)) {
    clog(
      `vault token minter already deployed at ${predictedMinterAddr}, reusing`,
    )
    VAULT_TOKEN_MINTER_ADDR = predictedMinterAddr
  } else {
    clog("\nGranting vault token minter x/sudo permission...")
    const editSudoersFile = join(
      config.cacheDir,
      "edit_sudoers_vault_minter.json",
    )
    writeFileSync(
      editSudoersFile,
      JSON.stringify(
        {
          action: "add_contracts",
          contracts: [predictedMinterAddr],
        },
        null,
        2,
      ),
    )
    const editSudoersCmd = `nibid tx sudo edit-sudoers ${editSudoersFile} --from validator ${TX_FLAGS}`
    const editSudoersResult = await execValidatorTx({
      command: editSudoersCmd,
      requireDeliveredSuccess: false,
    })
    if (editSudoersResult.code !== 0) {
      cerr(`edit-sudoers note: (${editSudoersResult.raw_log})`)
    }

    const minterInstantiateCmd = `nibid tx wasm instantiate2 ${VAULT_TOKEN_MINTER_CODE_ID} "$(cat ${config.initVaultTokenMinterFile})" ${VAULT_TOKEN_MINTER_SALT_HEX} --fix-msg --amount ${ONE_THOUSAND_UNIBI} --label "vault_token_minter" --admin ${config.signers.valAddr} --from validator ${TX_FLAGS}`
    try {
      const minterInstantiateResult = await execValidatorTx({
        command: minterInstantiateCmd,
        queryAttempts: CONTRACT_DEPLOY_QUERY_ATTEMPTS,
      })
      VAULT_TOKEN_MINTER_ADDR = await getContractAddress(
        minterInstantiateResult,
      )
    } catch (error) {
      if (await wasmContractExists(predictedMinterAddr)) {
        clog(
          `instantiate2 duplicate on consecutive deploy; reusing ${predictedMinterAddr}`,
        )
        VAULT_TOKEN_MINTER_ADDR = predictedMinterAddr
      } else {
        throw error
      }
    }
  }
  // Guard the clever bit above: the sudo grant is only useful if instantiate2
  // produced the same address we predicted and added to sudoers.
  if (VAULT_TOKEN_MINTER_ADDR !== predictedMinterAddr) {
    throw new Error(
      `vault token minter address mismatch: predicted ${predictedMinterAddr}, got ${VAULT_TOKEN_MINTER_ADDR}`,
    )
  }
  clog(`Instantiated vault token minter at: ${VAULT_TOKEN_MINTER_ADDR}`)
  storeContractAddress("VAULT_TOKEN_MINTER", VAULT_TOKEN_MINTER_ADDR)

  // Update vault init message with perp and minter addresses
  vaultInitMsg.perp_contract = PERP_ADDRESS
  vaultInitMsg.vault_token_minter_contract = VAULT_TOKEN_MINTER_ADDR
  writeFileSync(cacheInitVaultFile, JSON.stringify(vaultInitMsg, null, 2))

  // Instantiate vault contract
  const vaultInstantiateCmd = `nibid tx wasm instantiate ${VAULT_CODE_ID} "$(cat ${cacheInitVaultFile})" --amount ${ONE_THOUSAND_UNIBI} --label "vault" --admin ${config.signers.valAddr} --from validator ${TX_FLAGS}`
  const vaultInstantiateResult = await execValidatorTx({
    command: vaultInstantiateCmd,
    queryAttempts: CONTRACT_DEPLOY_QUERY_ATTEMPTS,
  })
  const VAULT_ADDR = await getContractAddress(vaultInstantiateResult)
  clog(`Instantiated vault at: ${VAULT_ADDR}`)
  storeContractAddress("VAULT_ADDRESS", VAULT_ADDR)

  // Post prices
  clog("\nPosting prices...")
  const setupPricesJson = JSON.parse(
    readFileSync(`${config.txArgsDir}/setup_prices.json`, "utf8"),
  ) as SetupPricesJson

  // Update oracle address in setup_prices.json
  for (const message of setupPricesJson.body.messages) {
    if (message.contract) {
      message.contract = ORACLE_ADDR
    }
  }

  const setupPricesCacheFile = join(config.cacheDir, "setup_prices.json")
  writeFileSync(setupPricesCacheFile, JSON.stringify(setupPricesJson, null, 2))

  // Sign and broadcast setup_prices.json
  const signedPricesCacheFile = join(config.cacheDir, "signed_prices.json")
  const signPricesCmd = `nibid tx sign ${setupPricesCacheFile} --from validator ${TX_FLAGS}`
  const signedPricesJson = JSON.parse((await execCmd(signPricesCmd)).stdout)
  writeFileSync(
    signedPricesCacheFile,
    JSON.stringify(signedPricesJson, null, 2),
  )

  const broadcastPricesCmd = `nibid tx broadcast ${signedPricesCacheFile} --from validator ${TX_FLAGS}`
  const broadcastPricesResult = await broadcastSignedTx(broadcastPricesCmd)
  ensureTxOk(broadcastPricesResult)

  // Setup market
  clog("\nSetting up market...")
  const setupMarketJson = JSON.parse(
    readFileSync(`${config.txArgsDir}/setup_market.json`, "utf8"),
  )

  // Update perp and vault addresses in setup_market.json
  for (const message of setupMarketJson.body.messages) {
    if (message.contract) {
      message.contract = PERP_ADDRESS
    }

    if (
      message.msg &&
      message.msg.admin &&
      message.msg.admin.msg &&
      message.msg.admin.msg.update_vault_address
    ) {
      message.msg.admin.msg.update_vault_address.vault_address = VAULT_ADDR
    }
  }

  const setupMarketCacheFile = join(config.cacheDir, "setup_market.json")
  writeFileSync(setupMarketCacheFile, JSON.stringify(setupMarketJson, null, 2))

  // Sign and broadcast setup_market.json
  const signedMarketCacheFile = join(config.cacheDir, "signed_market.json")
  const signMarketCmd = `nibid tx sign ${setupMarketCacheFile} --from validator ${TX_FLAGS}`
  const signedMarketJson = JSON.parse((await execCmd(signMarketCmd)).stdout)
  writeFileSync(
    signedMarketCacheFile,
    JSON.stringify(signedMarketJson, null, 2),
  )

  const broadcastMarketCmd = `nibid tx broadcast ${signedMarketCacheFile} --from validator ${TX_FLAGS}`
  const broadcastMarketResult = await broadcastSignedTx(broadcastMarketCmd)
  ensureTxOk(broadcastMarketResult)

  // Whitelist vault to be able to mint
  clog("\nWhitelisting vault...")
  const whitelistCmd = `nibid tx wasm execute ${VAULT_TOKEN_MINTER_ADDR} '{"add_minter":{"address":"${VAULT_ADDR}"}}' --from validator ${TX_FLAGS}`
  const whitelistResult = await execValidatorTx({ command: whitelistCmd })
  ensureTxOk(whitelistResult)

  // LP deposit
  clog("\nMaking LP deposit...")
  const lpDepositCmd = `nibid tx wasm execute ${VAULT_ADDR} "$(cat ${config.txArgsDir}/lp_deposit.json)" --from validator ${TX_FLAGS} --amount ${ONE_THOUSAND_USDC_BASE_UNITS}${TEST_USDC_DENOM}`
  const lpDepositResult = await execValidatorTx({ command: lpDepositCmd })
  ensureTxOk(lpDepositResult)

  clog("WASM contract deployment completed successfully!")
}

/** Register an EVM contract as zero-gas via x/sudo edit-zero-gas. */
async function addZeroGasContract(evmAddr: string): Promise<void> {
  const TX_FLAGS = `--fees ${config.fees} --gas ${config.gas} --yes --broadcast-mode sync --keyring-backend ${config.keyringBackend} --chain-id ${config.chainId} --node http://localhost:26657`
  const actors = JSON.stringify({
    senders: [] as string[],
    contracts: [] as string[],
    always_zero_gas_contracts: [evmAddr],
  })
  const cmd = `nibid tx sudo edit-zero-gas '${actors}' --from validator ${TX_FLAGS}`
  clogCmd(cmd)
  const txRes = await execValidatorTx({
    command: cmd,
    requireDeliveredSuccess: false,
  })
  if (txRes.code !== 0) {
    cerr(`edit-zero-gas note: (${txRes.raw_log})`)
  }
  clog(`SaiEvm registered as zero-gas contract: ${evmAddr}`)
}

const QUERY_NODE = "http://localhost:26657"

/**
 * `signers.evmPrivKey` can start with EVM balance_wei 0; `factory.deploy` pays intrinsic gas before
 * SaiEvm is registered as zero-gas. `convert-coin-to-evm` for USDC only funds ERC-20.
 * Send native `unibi` to the x/evm-linked bech32 for this ETH address (same pattern as
 * funding users in test_simple_evm).
 */
async function ensureEvmSignerNativeForDeploy(
  evmAddr: string,
  txFlags: string,
): Promise<void> {
  const minWei = 30n * 10n ** 18n
  const { stdout } = await execCmd(
    `nibid q evm account ${evmAddr} --node ${QUERY_NODE} -o json`,
  )
  const acc = JSON.parse(stdout) as {
    balance_wei: string
    bech32_address: string
  }
  const wei = BigInt(acc.balance_wei ?? "0")
  if (wei >= minWei) {
    clog(
      `E2E EVM signer ${evmAddr} balance_wei=${wei} (>= ${minWei}), skip native top-up`,
    )
    return
  }
  const amountUnibi = `50${"0".repeat(6)}` // 50 NIBI in unibi
  clog(
    `Funding E2E EVM signer for SaiEvm deploy: bank send ${amountUnibi}unibi → ${acc.bech32_address} (linked ${evmAddr}); balance_wei was ${wei}`,
  )
  const fundCmd = `nibid tx bank send ${config.signers.valAddr} ${acc.bech32_address} ${amountUnibi}unibi --from validator ${txFlags}`
  ensureTxOk(await execValidatorTx({ command: fundCmd }))
}

/** Deploy SaiEvm directly via constructor (non-upgradeable). */
async function deploySaiEvmDirect(): Promise<void> {
  clog("\nDeploying SaiEvm (direct, non-upgradeable)...")

  const provider = new ethers.JsonRpcProvider(config.rpcUrl)
  const evmSignerWallet = new ethers.Wallet(config.signers.evmPrivKey, provider)

  const perpAddress = contractAddresses["PERP_ADDRESS"]
  if (!perpAddress) {
    throw new Error("PERP_ADDRESS not in contractAddresses")
  }

  const SaiEvmArtifact = loadSaiEvmArtifact(__dirname)

  const saiEvmFactory = new ethers.ContractFactory(
    SaiEvmArtifact.abi,
    SaiEvmArtifact.bytecode,
    evmSignerWallet,
  )
  const saiEvm = await saiEvmFactory.deploy(perpAddress, 0)
  await saiEvm.waitForDeployment()
  const contractAddr = await saiEvm.getAddress()
  clog(`SaiEvm deployed at: ${contractAddr}`)

  storeContractAddress("SAI_EVM_ADDRESS", contractAddr)
  clog("SaiEvm deployment completed.")
}

// Main function to run the deployment process
async function main() {
  try {
    const TX_FLAGS = `--fees ${config.fees} --gas ${config.gas} --yes --broadcast-mode sync --keyring-backend ${config.keyringBackend} --chain-id ${config.chainId} --node http://localhost:26657`
    const e2eEvmSignerAddress = new ethers.Wallet(config.signers.evmPrivKey)
      .address
    if (!existsSync(config.cacheDir)) {
      mkdirSync(config.cacheDir, { recursive: true })
    }

    // Step 1: Deploy WASM contracts
    await deployWasmContracts()

    await ensureEvmSignerNativeForDeploy(e2eEvmSignerAddress, TX_FLAGS)

    // Step 1b: Deploy SaiEvm directly (constructor only; non-upgradeable)
    await deploySaiEvmDirect()

    // Step 1c: Register SaiEvm as zero-gas contract via x/sudo
    const saiEvmAddr = contractAddresses["SAI_EVM_ADDRESS"]
    if (!saiEvmAddr) {
      throw new Error("SAI_EVM_ADDRESS not set after deploySaiEvmDirect")
    }
    await addZeroGasContract(saiEvmAddr)

    // Step 2: Create funtoken mapping for USDC
    clog("\nCreating funtoken mapping for USDC...")
    try {
      const createFuntokenCmd = `nibid tx evm create-funtoken --bank-denom ${TEST_USDC_DENOM} --from validator ${TX_FLAGS}`
      const jsonRes = await execValidatorTx({
        command: createFuntokenCmd,
        requireDeliveredSuccess: false,
      })
      if (
        jsonRes.code !== 0 &&
        jsonRes.raw_log.includes("funtoken mapping already created")
      ) {
        clog(
          "nibid tx evm create-funtoken failed because the funtoken mapping was already created (false negative)",
        )
      } else if (jsonRes.code !== 0) {
        cerr(
          `Exit: reason = nibid tx evm create-funtoken: (${jsonRes.raw_log})`,
        )
        throw new Error(`Exit: reason = nibid tx evm create-funtoken`)
      }
    } catch (error) {
      cerr("Failed to create funtoken mapping:", error)
      clog(
        "Note: If the token mapping already exists, this step can be skipped.",
      )
    }

    // Query the created funtoken address
    clog(
      "Funtoken mapping created successfully. Querying the ERC-20 address...",
    )
    const queryFuntokenResult = await execCmd(
      `nibid q evm funtoken ${TEST_USDC_DENOM} --node http://localhost:26657`,
    )

    const cleanedOutput = queryFuntokenResult.stdout
    let funTokenInfo

    funTokenInfo = JSON.parse(cleanedOutput)
    const tokenNibiAddress = funTokenInfo.fun_token.erc20_addr

    clog(`USDC ERC-20 token address: ${tokenNibiAddress}`)
    storeContractAddress("TOKEN_USDC", tokenNibiAddress)

    // Convert some local USDC bank coins to ERC-20 form for EVM tests.
    const convertCmd = `nibid tx evm convert-coin-to-evm ${e2eEvmSignerAddress} ${TEN_THOUSAND_USDC_BASE_UNITS}${TEST_USDC_DENOM} --from validator ${TX_FLAGS}`
    const convertResultWait = await execValidatorTx({ command: convertCmd })
    ensureTxOk(convertResultWait)
    clog(`Converted 10000 ${TEST_USDC_DENOM} to EVM`)

    clog("Creating vault shares into erc-20 form")

    const vaultSharesDenom = `tf/${contractAddresses["VAULT_TOKEN_MINTER"]}/vault_shares`

    {
      const bankDenom = vaultSharesDenom
      const decimals = 6
      const bankMetadata = {
        base: bankDenom,
        display: `decimals_denom_for-${bankDenom}`,
        name: "SLP Shares for USDC",
        symbol: "SLP-USDC",
        description: "Tokens for Sai E2E tests on localnet",
        denom_units: [
          {
            denom: bankDenom,
            exponent: 0,
          },
          {
            denom: `decimals_denom_for-${bankDenom}`,
            exponent: decimals,
          },
        ],
      }
      const fPath = join(config.cacheDir, "metadata-shares.json")
      const f = Bun.file(fPath)
      clog(
        `Always write the metadata file with the current vault token minter address`,
      )
      await Bun.write(f, JSON.stringify(bankMetadata, null, 2))
      const setDenomMetadata = `nibid tx tokenfactory sudo-set-denom-metadata ${fPath} --from validator ${TX_FLAGS}`
      clogCmd(setDenomMetadata)
      const jsonRes = await execValidatorTx({
        command: setDenomMetadata,
        requireDeliveredSuccess: false,
      })
      if (jsonRes.code !== 0) {
        cerr(`sudo-set-denom-metadata note: (${jsonRes.raw_log})`)
      } else {
        const lite = txResultLite(jsonRes)
        clog(`Successful sudo-set-denom-metadata %o`, {
          "tx.body.messages": lite["tx.body.messages"],
          txhash: lite.txhash,
        })
      }
    }

    try {
      const createSharesCmd = `nibid tx evm create-funtoken --bank-denom ${vaultSharesDenom} --from validator ${TX_FLAGS}`
      const createSharesResultWait = await execValidatorTx({
        command: createSharesCmd,
        requireDeliveredSuccess: false,
      })
      if (
        createSharesResultWait.code !== 0 &&
        createSharesResultWait.raw_log.includes("funtoken mapping already created")
      ) {
        clog(
          "vault shares create-funtoken: mapping already exists (false negative)",
        )
      } else if (createSharesResultWait.code !== 0) {
        cerr(
          `Exit: reason = vault shares create-funtoken: (${createSharesResultWait.raw_log})`,
        )
        throw new Error(`Exit: reason = vault shares create-funtoken`)
      } else {
        clog(`Created vault shares into erc-20: %o ${createSharesResultWait}`)
      }
    } catch (error) {
      cerr("Failed to create vault shares funtoken mapping:", error)
      clog(
        "Note: If the vault shares funtoken mapping already exists, this step can be skipped.",
      )
    }

    // Save all contract addresses to environment file
    saveEnvironmentFile()

    clog("\nAll contracts deployed successfully!")
    clog(`Contract addresses saved to ${config.envFile}`)

    return 0
  } catch (error) {
    cerr("Deployment failed:", error)
    return 1
  }
}

// Run the main function
main()
  .then(process.exit)
  .catch((error) => {
    console.error(error)
    process.exit(1)
  })
