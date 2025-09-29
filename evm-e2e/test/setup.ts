import { config } from "dotenv"
import { ethers, Mnemonic } from "ethers"
import { HDNodeWallet } from "ethers/wallet"

config()

const provider = new ethers.JsonRpcProvider(process.env.JSON_RPC_ENDPOINT)
const mnemonic = Mnemonic.fromPhrase(process.env.MNEMONIC!)
// First account derived from the mnemonic at index 0, already funded with native tokens
const account = HDNodeWallet.fromMnemonic(mnemonic, "m/44'/60'/0'/0/0").connect(provider)
// Second account derived from the same mnemonic but at index 1, used for testing with zero native balance
const account2 = HDNodeWallet.fromMnemonic(mnemonic, "m/44'/60'/0'/0/1").connect(provider)
const TEST_TIMEOUT = Number(process.env.TEST_TIMEOUT) || 15000
const TX_WAIT_TIMEOUT = Number(process.env.TX_WAIT_TIMEOUT) || 5000



export { account, account2, provider, TEST_TIMEOUT, TX_WAIT_TIMEOUT }

