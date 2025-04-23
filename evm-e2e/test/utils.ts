import { wnibiCaller } from "@nibiruchain/evm-core"
import {
  ContractFactory,
  ContractTransactionResponse,
  parseEther,
  toBigInt,
  Wallet,
  type TransactionRequest,
} from "ethers"

import WNIBI_JSON from "../../x/evm/embeds/artifacts/contracts/WNIBI.sol/WNIBI.json"
import {
  EventsEmitter__factory,
  InifiniteLoopGas__factory,
  NibiruOracleChainLinkLike__factory,
  SendNibi__factory,
  TestERC20__factory,
  TransactionReverter__factory,
  type NibiruOracleChainLinkLike,
} from "../types"
import { account, provider, TX_WAIT_TIMEOUT } from "./setup"

export const alice = Wallet.createRandom()

export const hexify = (x: number): string => {
  return "0x" + x.toString(16)
}

export const INTRINSIC_TX_GAS: bigint = 21000n

function hasNonceError(err: unknown): boolean {
  const msg = typeof err === "object" && err !== null && "message" in err
    ? String((err as any).message)
    : String(err)
  return msg.includes("invalid nonce") || msg.includes("invalid sequence")
}

export async function withRetry<T>(
  fn: () => Promise<T>,
  shouldRetry: (err: unknown) => boolean = hasNonceError,
  retries = 2
): Promise<T> {
  let attempt = 0

  while (attempt <= retries) {
    try {
      return await fn()
    } catch (err) {
      const should = shouldRetry(err)
      console.debug("withRetry caught error", { err, should })

      if (attempt === retries || !should) {
        throw err
      }

      console.warn(`Retrying (${attempt + 1}/${retries}) due to error: ${err}`)
      attempt++
    }
  }
  throw new Error("Exhausted retries for nonce errors")
}

export const deployContractTestERC20 = async () => await withRetry(
  async () => {
    const factory = new TestERC20__factory(account)
    const contract = await factory.deploy()
    await contract.waitForDeployment()
    return contract
  }
)

export const deployContractSendNibi = async () => await withRetry(
  async () => {
    const factory = new SendNibi__factory(account)
    const contract = await factory.deploy()
    await contract.waitForDeployment()
    return contract
  }
)

export const deployContractInfiniteLoopGas = async () => await withRetry(
  async () => {
    const factory = new InifiniteLoopGas__factory(account)
    const contract = await factory.deploy()
    await contract.waitForDeployment()
    return contract
  }
)

export const deployContractEventsEmitter = async () => await withRetry(
  async () => {
    const factory = new EventsEmitter__factory(account)
    const contract = await factory.deploy()
    await contract.waitForDeployment()
    return contract
  }
)

export const deployContractTransactionReverter = async () => await withRetry(
  async () => {
    const factory = new TransactionReverter__factory(account)
    const contract = await factory.deploy()
    await contract.waitForDeployment()
    return contract
  }
)

export const sendTestNibi = async () => await withRetry(
  async () => {
    const transaction: TransactionRequest = {
      gasLimit: toBigInt(100e3),
      to: alice,
      value: parseEther("0.01"),
    }
    const txResponse = await account.sendTransaction(transaction)
    await txResponse.wait(1, TX_WAIT_TIMEOUT)
    console.log(txResponse)
    return txResponse
  }
)

export const deployContractNibiruOracleChainLinkLike = async (): Promise<{
  oraclePair: string
  contract: NibiruOracleChainLinkLike & {
    deploymentTransaction(): ContractTransactionResponse
  }
}> => await withRetry(async () => {
  const oraclePair = "ueth:uusd"
  const factory = new NibiruOracleChainLinkLike__factory(account)
  const contract = await factory.deploy(oraclePair, toBigInt(8))
  await contract.waitForDeployment()
  return { oraclePair, contract }
})

export type WNIBI = ReturnType<typeof wnibiCaller>
export type DeploymentTx = {
  deploymentTransaction(): ContractTransactionResponse
}
export const deployContractWNIBI = async (): Promise<{
  contract: WNIBI & DeploymentTx
}> => await withRetry(async () => {
  const { abi, bytecode } = WNIBI_JSON
  const factory = new ContractFactory(abi, bytecode, account)
  const contract = await factory.deploy()
  await contract.waitForDeployment()
  return { contract: contract as unknown as WNIBI & DeploymentTx }
})

export const numberToHex = (num: Number) => {
  return "0x" + num.toString(16)
}
