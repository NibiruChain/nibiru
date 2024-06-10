import { ethers } from "ethers"
import { config } from "dotenv"
import * as fs from "fs"

config()

const rpcEndpoint = process.env.JSON_RPC_ENDPOINT
const mnemonic = process.env.MNEMONIC

const provider = ethers.getDefaultProvider(rpcEndpoint)
const wallet = ethers.Wallet.fromPhrase(mnemonic)
const account = wallet.connect(provider)

const deployContract = async (path: string) => {
  const contractJSON = JSON.parse(
    fs.readFileSync(`contracts/${path}`).toString(),
  )
  const bytecode = contractJSON["bytecode"]
  const abi = contractJSON["abi"]

  const contractFactory = new ethers.ContractFactory(abi, bytecode, account)
  const contract = await contractFactory.deploy()
  await contract.waitForDeployment()
  return contract
}

export { provider, account, deployContract }
