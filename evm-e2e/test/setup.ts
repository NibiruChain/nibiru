import { config } from "dotenv"
import { ethers, getDefaultProvider, Wallet, Mnemonic } from "ethers"
import { HDNodeWallet } from "ethers/wallet"

config()

const provider = new ethers.JsonRpcProvider(process.env.JSON_RPC_ENDPOINT)
const account = Wallet.fromPhrase(process.env.MNEMONIC, provider)
const TEST_TIMEOUT = Number(process.env.TEST_TIMEOUT) || 15000
const TX_WAIT_TIMEOUT = Number(process.env.TX_WAIT_TIMEOUT) || 5000

const mnemonic = Mnemonic.fromPhrase(process.env.MNEMONIC!)
const account2 = HDNodeWallet.fromMnemonic(mnemonic, "m/44'/60'/0'/0/1").connect(provider)

export { account, account2, provider, TEST_TIMEOUT, TX_WAIT_TIMEOUT }

