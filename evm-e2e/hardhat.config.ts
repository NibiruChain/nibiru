import "dotenv/config"
import { HardhatUserConfig } from "hardhat/config"
import "@nomicfoundation/hardhat-toolbox"
import "@typechain/hardhat"

const RPC_URL = process.env.JSON_RPC_ENDPOINT || "http://127.0.0.1:8545"
const CHAIN_ID = process.env.CHAIN_ID ? Number(process.env.CHAIN_ID) : undefined
const MNEMONIC = process.env.MNEMONIC

const config: HardhatUserConfig = {
  solidity: {
    version: "0.8.24",
    settings: {
      optimizer: {
        enabled: true,
        runs: 100,
      },
    },
  },
  typechain: {
    outDir: "types",
    target: "ethers-v6",
    alwaysGenerateOverloads: false,
    dontOverrideCompile: false,
  },
  networks: {
    localhost: {
      url: RPC_URL,
      chainId: CHAIN_ID,
      accounts: MNEMONIC ? { mnemonic: MNEMONIC } : undefined,
    },
  },
}

export default config
