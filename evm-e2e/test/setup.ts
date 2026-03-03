import { config } from "dotenv"
import { ethers, getDefaultProvider, Wallet } from "ethers"

config()

const provider = new ethers.JsonRpcProvider(process.env.JSON_RPC_ENDPOINT)

/** 
 * `account` is the primary funded signer for many of the  EVM E2E tests.
 * The seed phrase, `process.env.MNEMONIC`, is set by
 * `contrib/scripts/localnet.sh` to create the `validator` key on
 * the `nibiru-localnet-0` blockchain. That validator/dev account is
 * funded in genesis and can deploy contracts, pay gas, and fund
 * other test wallets.
 * */
const account = Wallet.fromPhrase(process.env.MNEMONIC, provider)

const TEST_TIMEOUT = Number(process.env.TEST_TIMEOUT) || 15000
const TX_WAIT_TIMEOUT = Number(process.env.TX_WAIT_TIMEOUT) || 5000

export { account, provider, TEST_TIMEOUT, TX_WAIT_TIMEOUT }
