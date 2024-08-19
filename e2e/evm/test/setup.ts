import { config } from "dotenv"
import { getDefaultProvider, Wallet } from "ethers"

config()

const provider = getDefaultProvider(process.env.JSON_RPC_ENDPOINT)
const account = Wallet.fromPhrase(process.env.MNEMONIC, provider)

export { account, provider }

