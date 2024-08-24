import { config } from "dotenv"
import { ethers, getDefaultProvider, Wallet } from "ethers"

config()

const provider = new ethers.JsonRpcProvider(process.env.JSON_RPC_ENDPOINT)
const account = Wallet.fromPhrase(process.env.MNEMONIC, provider)

export { account, provider }
