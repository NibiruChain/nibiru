import * as path from "path"
import { config } from "dotenv"
import { ethers, getDefaultProvider, Wallet } from "ethers"

/**
 * This function helps guarantee reasonable default values are configured even
 * if a .env file is not set up. It makes the tests more stable for people with
 * freshly cloned repositories.
 * */
async function configureProvessEnv(): Promise<void> {
  console.info(`\n
# e2e-evm/test/setup.ts: variable configuration
-------------------------------
`)

  const projectAbsRootPath = new URL("..", import.meta.url).pathname
  const envPath = path.join(projectAbsRootPath, ".env")
  const envSamplePath = path.join(projectAbsRootPath, ".env_sample")

  const envFile = Bun.file(envPath)
  const exists = await envFile.exists()
  if (!exists) {
    console.debug("Missing .env file at evm-e2e/.env")
    console.debug("Creating default .env file using evm-e2e/.env_sample")

    const sampleEnvFile = Bun.file(envSamplePath)
    const sampleEnvExists = await sampleEnvFile.exists()
    if (!sampleEnvExists) {
      throw new Error("Missing .env_sample file. Cannot create default .env.")
    }

    const contents = await sampleEnvFile.text()
    await Bun.write(envFile, contents)
    console.debug(".env file created successfully.")
  }

  console.debug("Parsing .env file to set environment variables")
  config()
}

configureProvessEnv()

const defaults = {
  JSON_RPC_ENDPOINT: "http://127.0.0.1:8545",
  MNEMONIC:
    "guard cream sadness conduct invite crumble clock pudding hole grit liar hotel maid produce squeeze return argue turtle know drive eight casino maze host",
  TEST_TIMEOUT: 15000,
  TX_WAIT_TIMEOUT: 5000,
}

const provider = new ethers.JsonRpcProvider(
  process.env.JSON_RPC_ENDPOINT || defaults.JSON_RPC_ENDPOINT,
)
const account = Wallet.fromPhrase(
  process.env.MNEMONIC || defaults.MNEMONIC,
  provider,
)
const TEST_TIMEOUT = Number(process.env.TEST_TIMEOUT) || defaults.TEST_TIMEOUT
const TX_WAIT_TIMEOUT =
  Number(process.env.TX_WAIT_TIMEOUT) || defaults.TX_WAIT_TIMEOUT

export { account, provider, TEST_TIMEOUT, TX_WAIT_TIMEOUT }
