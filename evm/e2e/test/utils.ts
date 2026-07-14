import { wnibiCaller } from "@nibiruchain/evm-core"
import {
  ContractFactory,
  ContractTransactionResponse,
  parseEther,
  toBigInt,
  TransactionReceipt,
  TransactionResponse,
  Wallet,
  type TransactionRequest,
} from "ethers"

import WNIBI_JSON from "../../embeds/artifacts/contracts/WNIBI.sol/WNIBI.json"
import {
  EventsEmitter__factory,
  InifiniteLoopGas__factory,
  SendNibi__factory,
  TestERC20__factory,
  TransactionReverter__factory,
} from "../types"
import { account, provider, TX_WAIT_TIMEOUT } from "./testdeps"

export const alice = Wallet.createRandom()

export const hexify = (x: number): string => {
  return "0x" + x.toString(16)
}

export const INTRINSIC_TX_GAS: bigint = 21000n

type TxResLite = {
  hash: string
  from: string
  to: string | null
  nonce: number
  type: number
  chainId: string
  value: string
  gasLimit: string
  maxPriorityFeePerGas: string | null
  maxFeePerGas: string | null
  blockNumber: number | null
  blockHash: string | null
  data: string
}

export const txResultLite = (
  txResponse: TransactionResponse | ContractTransactionResponse,
): TxResLite => {
  return {
    hash: txResponse.hash,
    from: txResponse.from,
    to: txResponse.to,
    nonce: txResponse.nonce,
    type: txResponse.type,
    chainId: txResponse.chainId.toString(),
    value: txResponse.value.toString(),
    gasLimit: txResponse.gasLimit.toString(),
    maxPriorityFeePerGas:
      txResponse.maxPriorityFeePerGas === null
        ? null
        : txResponse.maxPriorityFeePerGas.toString(),
    maxFeePerGas:
      txResponse.maxFeePerGas === null
        ? null
        : txResponse.maxFeePerGas.toString(),
    blockNumber: txResponse.blockNumber,
    blockHash: txResponse.blockHash,
    data: txResponse.data,
  }
}

type WaitForTxReceiptOptions = {
  attempts?: number
  blockTimeoutMs?: number
  pollIntervalMs?: number
  label?: string
  requireSuccess?: boolean
}

const waitForBlockAfter = async (
  blockNumber: number,
  timeoutMs: number,
  pollIntervalMs: number,
  txHash: string,
): Promise<number> => {
  const startedAt = Date.now()
  let latestBlock = blockNumber

  while (Date.now() - startedAt < timeoutMs) {
    await new Promise((resolve) => setTimeout(resolve, pollIntervalMs))
    latestBlock = await provider.getBlockNumber()
    if (latestBlock > blockNumber) {
      return latestBlock
    }
  }

  throw new Error(
    `[txWait] timed out after ${timeoutMs}ms waiting for next block after ${blockNumber} while waiting for tx ${txHash}; latestBlock=${latestBlock}`,
  )
}

type TxWaitInput = string | TransactionResponse | ContractTransactionResponse

const txHashOf = (tx: TxWaitInput): string => {
  return typeof tx === "string" ? tx : tx.hash
}

export const txWait = async (
  tx: TxWaitInput,
  options: WaitForTxReceiptOptions = {},
): Promise<TransactionReceipt> => {
  const normalizedTxHash = txHashOf(tx).trim()
  if (normalizedTxHash.length === 0) {
    throw new Error("txWait received an empty tx hash")
  }

  const attempts = options.attempts ?? 4
  const blockTimeoutMs = options.blockTimeoutMs ?? TX_WAIT_TIMEOUT
  const pollIntervalMs = options.pollIntervalMs ?? 250
  const requireSuccess = options.requireSuccess ?? true
  const label = options.label ?? "tx"
  const startedAt = Date.now()

  let receipt: TransactionReceipt | null = null
  for (let attempt = 1; attempt <= attempts; attempt++) {
    const latestBlock = await provider.getBlockNumber()
    console.log(
      `[txWait:${label}] txQuery=${attempt}/${attempts} tx=${normalizedTxHash} latestBlock=${latestBlock}`,
    )

    receipt = await provider.getTransactionReceipt(normalizedTxHash)
    if (receipt !== null) {
      console.log(
        `[txWait:${label}] confirmed txQuery=${attempt}/${attempts} tx=${normalizedTxHash} block=${receipt.blockNumber} status=${receipt.status} logs=${receipt.logs.length} elapsedMs=${Date.now() - startedAt}`,
      )
      break
    }

    if (attempt < attempts) {
      await waitForBlockAfter(
        latestBlock,
        blockTimeoutMs,
        pollIntervalMs,
        normalizedTxHash,
      )
    }
  }

  if (receipt === null) {
    throw new Error(
      `[txWait:${label}] tx ${normalizedTxHash} not found after ${attempts} receipt queries and ${attempts - 1} block waits; elapsedMs=${Date.now() - startedAt}`,
    )
  }

  if (typeof tx !== "string") {
    receipt = (await tx.wait(0)) ?? receipt
  }

  if (requireSuccess && receipt.status !== 1) {
    throw new Error(
      `transaction execution reverted: [txWait:${label}] tx=${normalizedTxHash} block=${receipt.blockNumber} status=${receipt.status} logs=${receipt.logs.length} elapsedMs=${Date.now() - startedAt}`,
    )
  }

  return receipt
}

export const deployContractTestERC20 = async () => {
  const factory = new TestERC20__factory(account)
  const contract = await factory.deploy()
  await contract.waitForDeployment()
  return contract
}

export const deployContractSendNibi = async () => {
  const factory = new SendNibi__factory(account)
  const contract = await factory.deploy()
  await contract.waitForDeployment()
  return contract
}

export const deployContractInfiniteLoopGas = async () => {
  const factory = new InifiniteLoopGas__factory(account)
  const contract = await factory.deploy()
  await contract.waitForDeployment()
  return contract
}

export const deployContractEventsEmitter = async () => {
  const factory = new EventsEmitter__factory(account)
  const contract = await factory.deploy()
  await contract.waitForDeployment()
  return contract
}

export const deployContractTransactionReverter = async () => {
  const factory = new TransactionReverter__factory(account)
  const contract = await factory.deploy()
  await contract.waitForDeployment()
  return contract
}

export const sendTestNibi = async () => {
  const transaction: TransactionRequest = {
    gasLimit: toBigInt(100e3),
    to: alice,
    value: parseEther("0.01"),
  }
  const txResponse = await account.sendTransaction(transaction)
  await txWait(txResponse, { label: "sendTestNibi" })
  console.log("sendTestNibi txResp: %o", txResultLite(txResponse))
  return txResponse
}

export type WNIBI = ReturnType<typeof wnibiCaller>
export type DeploymentTx = {
  deploymentTransaction(): ContractTransactionResponse
}
export const deployContractWNIBI = async (): Promise<{
  contract: WNIBI & DeploymentTx
}> => {
  const { abi, bytecode } = WNIBI_JSON
  const factory = new ContractFactory(abi, bytecode, account)
  const contract = await factory.deploy()
  await contract.waitForDeployment()
  return { contract: contract as unknown as WNIBI & DeploymentTx }
}

export const numberToHex = (num: Number) => {
  return "0x" + num.toString(16)
}
