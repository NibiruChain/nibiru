import {
  SendNibiCompiled__factory,
  TestERC20Compiled__factory,
} from "../types/ethers-contracts"
import { account } from "./setup"
import { parseEther, toBigInt, Wallet } from "ethers"

const alice = Wallet.createRandom()

const deployERC20 = async () => {
  const factory = new TestERC20Compiled__factory(account)
  const contract = await factory.deploy()
  await contract.waitForDeployment()
  return contract
}
const deploySendReceiveNibi = async () => {
  const factory = new SendNibiCompiled__factory(account)
  const contract = await factory.deploy()
  await contract.waitForDeployment()
  return contract
}

const sendTestNibi = async () => {
  const transaction = {
    gasLimit: toBigInt(100e3),
    to: alice,
    value: parseEther("0.01"),
  }
  const txResponse = await account.sendTransaction(transaction)
  await txResponse.wait(1, 10e3)
  return txResponse
}

export { alice, deployERC20, deploySendReceiveNibi, sendTestNibi }
