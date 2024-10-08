import {
  InfiniteLoopGasCompiled__factory,
  SendNibiCompiled__factory,
  TestERC20Compiled__factory,
} from "../types/ethers-contracts"
import { account } from "./setup"
import { parseEther, toBigInt, TransactionRequest, Wallet } from "ethers"

export const alice = Wallet.createRandom()

export const hexify = (x: number): string => {
  return "0x" + x.toString(16)
}

/** 10 to the power of 12 */
export const TENPOW12 = toBigInt(1e12)

export const INTRINSIC_TX_GAS: bigint = 21000n

export const deployContractTestERC20 = async () => {
  const factory = new TestERC20Compiled__factory(account)
  const contract = await factory.deploy({ maxFeePerGas: TENPOW12 })
  await contract.waitForDeployment()
  return contract
}

export const deployContractSendNibi = async () => {
  const factory = new SendNibiCompiled__factory(account)
  const contract = await factory.deploy({ maxFeePerGas: TENPOW12 })
  await contract.waitForDeployment()
  return contract
}

export const deployContractInfiniteLoopGas = async () => {
  const factory = new InfiniteLoopGasCompiled__factory(account)
  const contract = await factory.deploy({ maxFeePerGas: TENPOW12 })
  await contract.waitForDeployment()
  return contract
}

export const sendTestNibi = async () => {
  const transaction: TransactionRequest = {
    gasLimit: toBigInt(100e3),
    to: alice,
    value: parseEther("0.01"),
    maxFeePerGas: TENPOW12,
  }
  const txResponse = await account.sendTransaction(transaction)
  await txResponse.wait(1, 10e3)
  console.log(txResponse)
  return txResponse
}
