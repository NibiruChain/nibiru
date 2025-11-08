#!/usr/bin/env bun

import { readFileSync, writeFileSync, existsSync, mkdirSync } from "fs";
import { join } from "path";
import { newClog } from "@uniquedivine/jiyuu";
// import { ethers } from "ethers";
// import { wasmPrecompile } from "@nibiruchain/evm-core";

const { clog, cerr, clogCmd } = newClog(
  import.meta.url.includes("/")
    ? import.meta.url.split("/").pop()!
    : import.meta.url,
);

// Config for localnet
const config = {
  rpcUrl: "http://localhost:8545", // Nibiru EVM RPC endpoint
  validatorAddress: "nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl",
  /** 
   * privateKey: This is not a private credential. It's the priv key for the: 
   * doc. Default account in the localnet. Please don't use this account on 
   * mainnet. 
   * */
  privateKey:
    "0x68e80819679abccddfa31064ea84b2fe6870b1eaa0ebe2a1ff40a38533cfab8b",
  gasPrice: 500000000,
  gasLimit: 5000000,
  sampleTxsDir: join(__dirname, "sample_txs"),
  initPerpFile: join(__dirname, "sample_txs/init_perp.json"),
  initVaultFile: join(__dirname, "sample_txs/init_vault.json"),
  initOracleFile: join(__dirname, "sample_txs/init_oracle.json"),
  initVaultTokenMinterFile: join(
    __dirname,
    "sample_txs/init_vault_token_minter.json",
  ),
  artifactsDir: join(__dirname, "artifacts"),

  cacheDir: join(__dirname, ".cache"),
  localnetContractsFile: join(__dirname, ".cache/localnet_contracts.env"),

  keyringBackend: "test",
  chainId: "nibiru-localnet-0",
  fees: "750000unibi",
  gas: "30000000",
};

async function execCommand(
  command: string,
): Promise<{ stdout: string; stderr: string }> {
  try {
    const proc = Bun.spawn(["bash", "-c", command], {
      env: process.env as Record<string, string>,
      stdout: "pipe",
      stderr: "pipe",
    });

    const stdout = await new Response(proc.stdout).text();
    const stderr = await new Response(proc.stderr).text();

    if (stderr && stderr.trim() !== "") {
      cerr(`Command stderr: ${stderr}`);
    }

    const exitCode = await proc.exited;

    if (exitCode !== 0) {
      throw new Error(`Command exited with code ${exitCode}: ${stderr}`);
    }

    return { stdout, stderr };
  } catch (error) {
    cerr(`Command execution failed: ${error}`);
    throw error;
  }
}

type TxResRaw = {
  height: string;
  /** Example: txhash: "3D2BED5C6D67BF2A5D84473575315D5BC516F60CA35F7FC624B9C68F584F677F" */
  txhash: string;
  codespace: string;
  /** code: error code */
  code: number;
  /** raw_log: where error messages go if a tx fails (code != 0) */
  raw_log: string;
  events: {
    type: string;
    attributes: { key: string; value: string }[];
  }[];
  gas_wanted: "30000000";
  gas_used: "48679";
  tx: {
    "@type": string;
    body: { messages: Object[] };
  };
};

type TxResLite = Omit<TxResRaw, "tx"> & {
  "tx.body.messages": { messages: Object[] };
  "tx.@type": string;
};

// Helper function to wait for transaction to be mined
async function waitForTx(txhash: string): Promise<TxResRaw> {
  clog(`Waiting for transaction ${txhash} to be mined...`);

  // Wait for 6 seconds
  await new Promise((resolve) => setTimeout(resolve, 2000));

  try {
    const txResult = execCommand(
      `nibid q tx ${txhash} --node http://localhost:26657`,
    );

    const { stdout, stderr } = await txResult;
    if (stderr) {
      cerr(`waitForTx txResult.stderr ${stderr}`);
    }
    const jsonResult = JSON.parse(stdout) as TxResRaw;
    if (stderr && jsonResult.code !== 0) {
      cerr(`Failed transaction execution (code !== 0) ${txhash}`);
    }
    return JSON.parse(stdout) as TxResRaw;
  } catch (error) {
    cerr(`Failed to query transaction ${txhash}`);
    throw error;
  }
}

type ContractName = "PERP" | "VAULT" | "VAULT_TOKEN_MINTER" | "ORACLE";

// Extract code ID from store code response
async function getCodeId(
  jsonResult: TxResRaw,
  contractName: ContractName,
): Promise<string> {
  try {
    const events = jsonResult.events || [];

    for (const event of events) {
      if (event.type === "store_code") {
        for (const attribute of event.attributes) {
          if (attribute.key === "code_id") {
            return attribute.value;
          }
        }
      }
    }

    cerr("Failed to extract code_id: %o", {
      contractName: contractName,
      txRes: txResultLite(jsonResult),
    });
    const reasonSuffix: string =
      jsonResult.code !== 0 ? ` (${jsonResult.raw_log} )` : "";
    throw new Error(`Exit: reason=getCodeId${reasonSuffix}`);
  } catch (error) {
    cerr(`Failed to parse result or extract code_id: ${error}`);
    throw error;
  }
}

const txResultLite = (res: TxResRaw): TxResLite => {
  return {
    // Tx connection to the block
    height: res.height,
    txhash: res.txhash,

    // Most important details
    code: res.code,
    raw_log: res.raw_log,
    "tx.body.messages": res.tx.body,
    events: res.events || [],

    // Other concise info for debugging
    gas_wanted: res.gas_wanted,
    gas_used: res.gas_used,
    "tx.@type": res.tx["@type"],
    codespace: res.codespace,
  };
};

/**
 * getDenomTF: Extract token factory denom for stnibi
 * */
async function getDenomTF(jsonResult: TxResRaw): Promise<string> {
  try {
    const events = jsonResult.events || [];

    for (const event of events) {
      if (event.type === "nibiru.tokenfactory.v1.EventCreateDenom") {
        for (const attribute of event.attributes) {
          if (attribute.key === "denom") {
            // Return the raw string without additional quotes
            return attribute.value;
          }
        }
      }
    }

    // Fallback: Query token factory for denoms created by validator
    clog("Event parsing failed, falling back to querying token factory...");
    const queryCmd = `nibid query tokenfactory denoms ${config.validatorAddress} --node http://localhost:26657`;
    const queryResult = await execCommand(queryCmd);
    const denomsData: { denoms: string[] } = JSON.parse(queryResult.stdout);

    // Case: False negative. stNIBI may already exist.
    const maybeDenom = denomsData.denoms.find((d) => d.includes("stnibi"));
    if (maybeDenom) {
      return maybeDenom;
    }

    const denoms = denomsData.denoms || [];
    cerr("Failed to extract token factory denom: %o", {
      creator: config.validatorAddress,
      tfDenomsFromCreator: denoms,
      txRes: txResultLite(jsonResult),
    });
    const reasonSuffix: string =
      jsonResult.code !== 0 ? ` (${jsonResult.raw_log} )` : "";
    throw new Error(`Exit: reason=getDenomTF${reasonSuffix}`);
  } catch (error) {
    cerr(`Failed to parse result or extract denom: ${error}`);
    throw error;
  }
}

// Extract contract address from instantiate response
async function getContractAddress(jsonResult: TxResRaw): Promise<string> {
  try {
    const events = jsonResult.events || [];

    for (const event of events) {
      if (event.type === "instantiate") {
        for (const attribute of event.attributes) {
          if (attribute.key === "_contract_address") {
            return attribute.value;
          }
        }
      }
    }

    throw new Error("Failed to extract contract address");
  } catch (error) {
    cerr(`Failed to parse result or extract contract address: ${error}`);
    throw error;
  }
}

// Ensure transaction succeeded
function ensureTxOk(result: string | object): void {
  let jsonResult: TxResRaw;
  if (typeof result === "string") {
    jsonResult = JSON.parse(result);
  } else {
    jsonResult = result as unknown as TxResRaw;
  }
  try {
    const rawLog = jsonResult.raw_log;
    if (rawLog.includes("failed to execute message")) {
      cerr("Transaction failed with detailed error:");
      cerr("Raw log:", rawLog);
      cerr(
        "Full transaction result:",
        JSON.stringify(jsonResult, null, 2),
      );
      throw new Error(`Transaction failed: ${rawLog}`);
    }
    if (jsonResult.code !== 0) {
      cerr("Transaction failed with code:", jsonResult.code);
      cerr("Raw log:", rawLog);
      cerr(
        "Full transaction result:",
        JSON.stringify(jsonResult, null, 2),
      );
      throw new Error(
        `Transaction failed with code ${jsonResult.code}: ${rawLog}`,
      );
    }
  } catch (error) {
    cerr(`Failed to check transaction result: ${error}`);
    throw error;
  }
}

// Store environment variables
const contractAddresses: Record<string, string> = {};

function storeContractAddress(name: string, address: string) {
  contractAddresses[name] = address;
}

function saveEnvironmentFile() {
  let envContent = "";

  for (const [key, value] of Object.entries(contractAddresses)) {
    envContent += `${key}="${value}"\n`;
  }

  writeFileSync(config.localnetContractsFile, envContent);
  clog(`Contract addresses saved to ${config.localnetContractsFile}`);
}

// Deploy WASM contracts
async function deployWasmContracts() {
  clog("Starting WASM contract deployment...");

  // Transaction flags
  const TX_FLAGS = `--fees ${config.fees} --gas ${config.gas} --yes --keyring-backend ${config.keyringBackend} --chain-id ${config.chainId} --node http://localhost:26657`;

  clog(`Creating stNibi collateral`);
  try {
    const tokenFactoryCreation = `nibid tx tokenfactory create-denom stnibi --from validator ${TX_FLAGS}`;
    const tokenFactoryCreationTxHash = JSON.parse(
      (await execCommand(tokenFactoryCreation)).stdout,
    ).txhash;
    const tokenFactoryCreationResult = await waitForTx(
      tokenFactoryCreationTxHash,
    );
    const collateralDenom = await getDenomTF(tokenFactoryCreationResult);
    storeContractAddress("STNIBI", collateralDenom);
  } catch (error) {
    cerr("Failed to create stnibi:", error);
    clog(
      "Note: If the token mapping already exists, this step can be skipped.",
    );
  } finally {
    const bankDenom =
      "tf/nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl/stnibi";
    const decimals = 6;
    const bankMetadata = {
      base: bankDenom,
      display: `decimals_denom_for-${bankDenom}`,
      name: "Liquid Staked NIBI",
      symbol: "stNIBI",
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
    };
    const fPathBankMetadata = join(config.cacheDir, "bank_metadata.json");
    const f = Bun.file(fPathBankMetadata);
    if (!(await f.exists())) {
      await Bun.write(f, JSON.stringify(bankMetadata, null, 2));
    }
    const setDenomMetadata = `nibid tx tokenfactory set-denom-metadata ${fPathBankMetadata} --from validator ${TX_FLAGS}`;
    clogCmd(setDenomMetadata);
    const setDenomMetadataTxHash = JSON.parse(
      (await execCommand(setDenomMetadata)).stdout,
    ).txhash;
    const jsonRes = await waitForTx(setDenomMetadataTxHash);
    if (jsonRes.code !== 0) {
      const reason = "nibid tx tokenfactory set-denom-metadata";
      cerr(`Exit: reason = ${reason}: (${jsonRes.raw_log})`);
      throw new Error(`Exit: reason = ${reason}`);
    }
  }

  // mint some tokens to validator
  const mintCmd = `nibid tx tokenfactory mint 100000000000tf/nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl/stnibi --mint-to nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl --from validator ${TX_FLAGS}`;
  const mintTxHash = JSON.parse((await execCommand(mintCmd)).stdout).txhash;
  const mintResult = await waitForTx(mintTxHash);
  ensureTxOk(mintResult);

  // assert balance is minted
  const balanceCmd = `nibid q bank balances nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl --node http://localhost:26657 | jq -r '
  (.balances[] | select(.denom == "tf/nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl/stnibi").amount) // "0"
'
`;
  const balanceResult = await execCommand(balanceCmd);
  clog(`Validator balance: %o`, {
    amount: balanceResult.stdout.trim(),
    token: "tf/nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl/stnibi",
  });

  // Store contracts
  clog("Storing WASM contracts...");

  // Store perp contract
  const perpStoreCmd = `nibid tx wasm store ${config.artifactsDir}/perp.wasm --from validator ${TX_FLAGS}`;
  clogCmd(perpStoreCmd);
  const perpStoreTxHash = JSON.parse(
    (await execCommand(perpStoreCmd)).stdout,
  ).txhash;
  const perpStoreResult = await waitForTx(perpStoreTxHash);
  const PERP_CODE_ID = await getCodeId(perpStoreResult, "PERP");
  clog(`Stored perp contract with code ID: ${PERP_CODE_ID}`);
  storeContractAddress("PERP_CODE_ID", PERP_CODE_ID);

  // Store vault contract
  const vaultStoreCmd = `nibid tx wasm store ${config.artifactsDir}/vault.wasm --from validator ${TX_FLAGS}`;
  clogCmd(vaultStoreCmd);
  const vaultStoreTxHash = JSON.parse(
    (await execCommand(vaultStoreCmd)).stdout,
  ).txhash;
  const vaultStoreResult = await waitForTx(vaultStoreTxHash);
  const VAULT_CODE_ID = await getCodeId(vaultStoreResult, "VAULT");
  clog(`Stored vault contract with code ID: ${VAULT_CODE_ID}`);
  storeContractAddress("VAULT_CODE_ID", VAULT_CODE_ID);

  // Store vault token minter contract
  const minterStoreCmd = `nibid tx wasm store ${config.artifactsDir}/vault_token_minter.wasm --from validator ${TX_FLAGS}`;
  clogCmd(minterStoreCmd);
  const minterStoreTxHash = JSON.parse(
    (await execCommand(minterStoreCmd)).stdout,
  ).txhash;
  const minterStoreResult = await waitForTx(minterStoreTxHash);
  const VAULT_TOKEN_MINTER_CODE_ID = await getCodeId(
    minterStoreResult,
    "VAULT_TOKEN_MINTER",
  );
  clog(
    `Stored vault token minter with code ID: ${VAULT_TOKEN_MINTER_CODE_ID}`,
  );
  storeContractAddress(
    "VAULT_TOKEN_MINTER_CODE_ID",
    VAULT_TOKEN_MINTER_CODE_ID,
  );

  // Store oracle contract
  const oracleStoreCmd = `nibid tx wasm store ${config.artifactsDir}/oracle.wasm --from validator ${TX_FLAGS}`;
  clogCmd(oracleStoreCmd);
  const oracleStoreTxHash = JSON.parse(
    (await execCommand(oracleStoreCmd)).stdout,
  ).txhash;
  const oracleStoreResult = await waitForTx(oracleStoreTxHash);
  const ORACLE_CODE_ID = await getCodeId(oracleStoreResult, "ORACLE");
  clog(`Stored oracle contract with code ID: ${ORACLE_CODE_ID}`);
  storeContractAddress("ORACLE_CODE_ID", ORACLE_CODE_ID);

  // Instantiate contracts
  clog("\nInstantiating contracts...");

  // Instantiate oracle contract
  const oracleInstantiateCmd = `nibid tx wasm instantiate ${ORACLE_CODE_ID} "$(cat ${config.initOracleFile})" --amount 1000unibi --label "oracle" --admin ${config.validatorAddress} --from validator ${TX_FLAGS}`;
  clogCmd(oracleInstantiateCmd);
  const oracleInstantiateTxHash = JSON.parse(
    (await execCommand(oracleInstantiateCmd)).stdout,
  ).txhash;
  const oracleInstantiateResult = await waitForTx(oracleInstantiateTxHash);
  const ORACLE_ADDR = await getContractAddress(oracleInstantiateResult);
  clog(`Instantiated oracle at: ${ORACLE_ADDR}`);
  storeContractAddress("ORACLE_ADDRESS", ORACLE_ADDR);

  // Update init messages with oracle address
  const perpInitMsg = JSON.parse(readFileSync(config.initPerpFile, "utf8"));
  perpInitMsg.oracle_address = ORACLE_ADDR;
  writeFileSync(config.initPerpFile, JSON.stringify(perpInitMsg, null, 2));

  const vaultInitMsg = JSON.parse(readFileSync(config.initVaultFile, "utf8"));
  vaultInitMsg.oracle = ORACLE_ADDR;
  writeFileSync(config.initVaultFile, JSON.stringify(vaultInitMsg, null, 2));

  // Instantiate perp contract
  const perpInstantiateCmd = `nibid tx wasm instantiate ${PERP_CODE_ID} "$(cat ${config.initPerpFile})" --amount 1000unibi --label "perp" --admin ${config.validatorAddress} --from validator ${TX_FLAGS}`;
  const perpInstantiateTxHash = JSON.parse(
    (await execCommand(perpInstantiateCmd)).stdout,
  ).txhash;
  const perpInstantiateResult = await waitForTx(perpInstantiateTxHash);
  const PERP_ADDRESS = await getContractAddress(perpInstantiateResult);
  clog(`Instantiated perp contract at: ${PERP_ADDRESS}`);
  storeContractAddress("PERP_ADDRESS", PERP_ADDRESS);

  // Instantiate vault token minter
  const minterInstantiateCmd = `nibid tx wasm instantiate ${VAULT_TOKEN_MINTER_CODE_ID} "$(cat ${config.initVaultTokenMinterFile})" --amount 1000unibi --label "vault_token_minter" --admin ${config.validatorAddress} --from validator ${TX_FLAGS}`;
  const minterInstantiateTxHash = JSON.parse(
    (await execCommand(minterInstantiateCmd)).stdout,
  ).txhash;
  const minterInstantiateResult = await waitForTx(minterInstantiateTxHash);
  const VAULT_TOKEN_MINTER_ADDR = await getContractAddress(
    minterInstantiateResult,
  );
  clog(`Instantiated vault token minter at: ${VAULT_TOKEN_MINTER_ADDR}`);
  storeContractAddress("VAULT_TOKEN_MINTER", VAULT_TOKEN_MINTER_ADDR);

  // Update vault init message with perp and minter addresses
  vaultInitMsg.perp_contract = PERP_ADDRESS;
  vaultInitMsg.vault_token_minter_contract = VAULT_TOKEN_MINTER_ADDR;
  writeFileSync(config.initVaultFile, JSON.stringify(vaultInitMsg, null, 2));

  // Instantiate vault contract
  const vaultInstantiateCmd = `nibid tx wasm instantiate ${VAULT_CODE_ID} "$(cat ${config.initVaultFile})" --amount 1000unibi --label "vault" --admin ${config.validatorAddress} --from validator ${TX_FLAGS}`;
  const vaultInstantiateTxHash = JSON.parse(
    (await execCommand(vaultInstantiateCmd)).stdout,
  ).txhash;
  const vaultInstantiateResult = await waitForTx(vaultInstantiateTxHash);
  const VAULT_ADDR = await getContractAddress(vaultInstantiateResult);
  clog(`Instantiated vault at: ${VAULT_ADDR}`);
  storeContractAddress("VAULT_ADDRESS", VAULT_ADDR);

  // Post prices
  clog("\nPosting prices...");
  const setupPricesJson = JSON.parse(
    readFileSync(`${config.sampleTxsDir}/setup_prices.json`, "utf8"),
  );

  // Update oracle address in setup_prices.json
  for (const message of setupPricesJson.body.messages) {
    if (message.contract) {
      message.contract = ORACLE_ADDR;
    }
  }

  writeFileSync(
    `${config.sampleTxsDir}/setup_prices.json`,
    JSON.stringify(setupPricesJson, null, 2),
  );

  // Sign and broadcast setup_prices.json
  const signPricesCmd = `nibid tx sign ${config.sampleTxsDir}/setup_prices.json --from validator ${TX_FLAGS}`;
  const signedPricesJson = JSON.parse(
    (await execCommand(signPricesCmd)).stdout,
  );
  writeFileSync(
    `${config.sampleTxsDir}/signed.json`,
    JSON.stringify(signedPricesJson, null, 2),
  );

  const broadcastPricesCmd = `nibid tx broadcast ${config.sampleTxsDir}/signed.json --from validator ${TX_FLAGS}`;
  const broadcastPricesTxHash = JSON.parse(
    (await execCommand(broadcastPricesCmd)).stdout,
  ).txhash;
  const broadcastPricesResult = await waitForTx(broadcastPricesTxHash);
  ensureTxOk(broadcastPricesResult);

  // Setup market
  clog("\nSetting up market...");
  const setupMarketJson = JSON.parse(
    readFileSync(`${config.sampleTxsDir}/setup_market.json`, "utf8"),
  );

  // Update perp and vault addresses in setup_market.json
  for (const message of setupMarketJson.body.messages) {
    if (message.contract) {
      message.contract = PERP_ADDRESS;
    }

    if (
      message.msg &&
      message.msg.admin &&
      message.msg.admin.msg &&
      message.msg.admin.msg.update_vault_address
    ) {
      message.msg.admin.msg.update_vault_address.vault_address =
        VAULT_ADDR;
    }
  }

  writeFileSync(
    `${config.sampleTxsDir}/setup_market.json`,
    JSON.stringify(setupMarketJson, null, 2),
  );

  // Sign and broadcast setup_market.json
  const signMarketCmd = `nibid tx sign ${config.sampleTxsDir}/setup_market.json --from validator ${TX_FLAGS}`;
  const signedMarketJson = JSON.parse(
    (await execCommand(signMarketCmd)).stdout,
  );
  writeFileSync(
    `${config.sampleTxsDir}/signed.json`,
    JSON.stringify(signedMarketJson, null, 2),
  );

  const broadcastMarketCmd = `nibid tx broadcast ${config.sampleTxsDir}/signed.json --from validator ${TX_FLAGS}`;
  const broadcastMarketTxHash = JSON.parse(
    (await execCommand(broadcastMarketCmd)).stdout,
  ).txhash;
  const broadcastMarketResult = await waitForTx(broadcastMarketTxHash);
  ensureTxOk(broadcastMarketResult);

  // Whitelist vault to be able to mint
  clog("\nWhitelisting vault...");
  const whitelistCmd = `nibid tx wasm execute ${VAULT_TOKEN_MINTER_ADDR} '{"add_minter":{"address":"${VAULT_ADDR}"}}' --from validator ${TX_FLAGS}`;
  const whitelistTxHash = JSON.parse(
    (await execCommand(whitelistCmd)).stdout,
  ).txhash;
  const whitelistResult = await waitForTx(whitelistTxHash);
  ensureTxOk(whitelistResult);

  // LP deposit
  clog("\nMaking LP deposit...");
  const lpDepositCmd = `nibid tx wasm execute ${VAULT_ADDR} "$(cat ${config.sampleTxsDir}/lp_deposit.json)" --from validator ${TX_FLAGS} --amount 1000tf/nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl/stnibi`;
  const lpDepositTxHash = JSON.parse(
    (await execCommand(lpDepositCmd)).stdout,
  ).txhash;
  const lpDepositResult = await waitForTx(lpDepositTxHash);
  ensureTxOk(lpDepositResult);

  clog("WASM contract deployment completed successfully!");
}

const requireFile = async (path: string) => {
  if (!(await Bun.file(path).exists())) {
    throw new Error(`Required file missing: ${path}`);
  }
}

const requireCmd = async (name: string): Promise<boolean> => {
  let haveCmd: boolean = true
  try {
    await execCommand(`command -v ${name}`);
  } catch {
    haveCmd = false
  }
  return haveCmd
}

const preflightChecks = async () => {
  // artifacts dir should exist
  // artifacts dir must contain each wasm binary
  // artifacts dir must contain each wasm binary
  //   requireDir(config.artifactsDir);

  // Preflight 1 - CLI tools


  // Preflight 2 - Wasm
  let missingCmds: string[] = []
  for (const bashCmd of ["bash", "nibid", "jq"]) {
    const haveCmd = await requireCmd(bashCmd)
    if (!haveCmd) {
      missingCmds.concat(bashCmd)
    }
  }
  if (missingCmds.length != 0) {
    throw new Error(`Preflight: missing commands: ${missingCmds}`)
  }


  // Preflight 3 - Wasm binaries
  const REQUIRED_WASMS = [
    "perp.wasm",
    "vault.wasm",
    "vault_token_minter.wasm",
    "oracle.wasm",
  ];
  for (const f of REQUIRED_WASMS) {
    requireFile(join(config.artifactsDir, f));
  }


  // outputFile dir must exist
  // 
  clog("Preflight checks passed")
}

// Main function to run the deployment process
async function main() {
  try {
    // Create output directory if it doesn't exist
    const TX_FLAGS = `--fees ${config.fees} --gas ${config.gas} --yes --keyring-backend ${config.keyringBackend} --chain-id ${config.chainId} --node http://localhost:26657`;
    if (!existsSync(config.cacheDir)) {
      mkdirSync(config.cacheDir, { recursive: true });
    }

    await preflightChecks()

    // Step 1: Deploy WASM contracts
    await deployWasmContracts();

    // Step 2: Create funtoken mapping for stnibi
    clog("\nCreating funtoken mapping for stnibi...");
    try {
      const createFuntokenCmd = `nibid tx evm create-funtoken --bank-denom tf/nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl/stnibi --from validator ${TX_FLAGS}`;

      const createFuntokenResult = await execCommand(createFuntokenCmd);
      const createFuntokenTxHash = JSON.parse(
        createFuntokenResult.stdout,
      ).txhash;
      const jsonRes = await waitForTx(createFuntokenTxHash);
      if (
        jsonRes.code !== 0 &&
        jsonRes.raw_log.includes("funtoken mapping already created")
      ) {
        clog(
          "nibid tx evm create-funtoken failed because the funtoken mapping was already created (false negative)",
        );
      } else if (jsonRes.code !== 0) {
        cerr(
          `Exit: reason = nibid tx evm create-funtoken: (${jsonRes.raw_log})`,
        );
        throw new Error(`Exit: reason = nibid tx evm create-funtoken`);
      }
    } catch (error) {
      cerr("Failed to create funtoken mapping:", error);
      clog(
        "Note: If the token mapping already exists, this step can be skipped.",
      );
    }

    // Query the created funtoken address
    clog(
      "Funtoken mapping created successfully. Querying the ERC-20 address...",
    );
    const queryFuntokenResult = await execCommand(
      "nibid q evm funtoken tf/nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl/stnibi --node http://localhost:26657",
    );

    const cleanedOutput = queryFuntokenResult.stdout;
    let funTokenInfo;

    funTokenInfo = JSON.parse(cleanedOutput);
    const tokenNibiAddress = funTokenInfo.fun_token.erc20_addr;

    clog(`NIBI ERC-20 token address: ${tokenNibiAddress}`);
    storeContractAddress("TOKEN_STNIBI", tokenNibiAddress);

    // convert some stnibi to tf/nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl/stnibi
    const convertCmd = `nibid tx evm convert-coin-to-evm 0xC0f4b45712670cf7865A14816bE9Af9091EDdA1d 1000000000tf/nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl/stnibi --from validator ${TX_FLAGS}`;
    const convertResult = await execCommand(convertCmd);
    const convertTxHash = JSON.parse(convertResult.stdout).txhash;
    const convertResultWait = await waitForTx(convertTxHash);
    ensureTxOk(convertResultWait);
    clog(
      "Converted 10000 tf/nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl/stnibi to stnibi",
    );

    clog("Creating vault shares into erc-20 form");

    const vaultSharesDenom = `tf/${contractAddresses["VAULT_TOKEN_MINTER"]}/vault_shares`;

    {
      const bankDenom = vaultSharesDenom;
      const decimals = 6;
      const bankMetadata = {
        base: bankDenom,
        display: `decimals_denom_for-${bankDenom}`,
        name: "SLP Shares for stNIBI",
        symbol: "SLP-stNIBI",
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
      };
      const fPath = join(config.cacheDir, "metadata-shares.json");
      const f = Bun.file(fPath);
      clog(
        `Always write the metadata file with the current vault token minter address`,
      );
      await Bun.write(f, JSON.stringify(bankMetadata, null, 2));
      const setDenomMetadata = `nibid tx tokenfactory sudo-set-denom-metadata ${fPath} --from validator ${TX_FLAGS}`;
      clogCmd(setDenomMetadata);
      const setDenomMetadataTxHash = JSON.parse(
        (await execCommand(setDenomMetadata)).stdout,
      ).txhash;
      const jsonRes = await waitForTx(setDenomMetadataTxHash);
      if (jsonRes.code !== 0) {
        const reason = "nibid tx tokenfactory sudo-set-denom-metadata";
        cerr(`Exit: reason = ${reason}: (${jsonRes.raw_log})`);
        throw new Error(`Exit: reason = ${reason}`);
      }
      const lite = txResultLite(jsonRes);
      clog(`Successful sudo-set-denom-metadata %o`, {
        "tx.body.messages": lite["tx.body.messages"],
        txhash: lite.txhash,
      });
    }

    const createSharesCmd = `nibid tx evm create-funtoken --bank-denom ${vaultSharesDenom} --from validator ${TX_FLAGS}`;
    const createSharesResult = await execCommand(createSharesCmd);
    const createSharesTxHash = JSON.parse(createSharesResult.stdout).txhash;
    const createSharesResultWait = await waitForTx(createSharesTxHash);
    ensureTxOk(createSharesResultWait);
    clog(`Created vault shares into erc-20: %o`, createSharesResultWait)

    // Save all contract addresses to environment file
    saveEnvironmentFile();

    clog("\nAll contracts deployed successfully!");
    clog(`Contract addresses saved to ${config.localnetContractsFile}`);

    return 0;
  } catch (error) {
    cerr("Deployment failed:", error);
    return 1;
  }
}

// Run the main function
main()
  .then(process.exit)
  .catch((error) => {
    console.error(error);
    process.exit(1);
  });
