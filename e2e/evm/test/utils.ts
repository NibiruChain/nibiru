import {
  SendNibiCompiled__factory,
  TestERC20Compiled__factory,
} from "../types/ethers-contracts"
import { account } from "./setup"

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

export { deployERC20, deploySendReceiveNibi }
