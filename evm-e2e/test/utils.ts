import { wnibiCaller } from "@nibiruchain/evm-core"
import {
  ContractFactory,
  ContractTransactionResponse,
  parseEther,
  toBigInt,
  TransactionResponse,
  Wallet,
  type TransactionRequest,
} from "ethers"

import WNIBI_JSON from "../../x/evm/embeds/artifacts/contracts/WNIBI.sol/WNIBI.json"
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
  await txResponse.wait(1, TX_WAIT_TIMEOUT)
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
