import { config } from "dotenv"
import { ethers, getDefaultProvider, Wallet,Mnemonic } from "ethers"
import { HDNodeWallet } from "ethers/wallet"

config()

const mnemonic = Mnemonic.fromPhrase(process.env.MNEMONIC!)

const provider = new ethers.JsonRpcProvider(process.env.JSON_RPC_ENDPOINT)
const account = HDNodeWallet.fromMnemonic(mnemonic, "m/44'/118'/0'/0/0").connect(provider)
const TEST_TIMEOUT = Number(process.env.TEST_TIMEOUT) || 15000
const TX_WAIT_TIMEOUT = Number(process.env.TX_WAIT_TIMEOUT) || 5000

export { account, provider, TEST_TIMEOUT, TX_WAIT_TIMEOUT }
