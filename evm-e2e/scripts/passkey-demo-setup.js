#!/usr/bin/env node
/**
 * Deploys EntryPoint + PasskeyAccountFactory to the configured RPC and
 * writes outputs for the passkey UI.
 */
require("dotenv/config")
const fs = require("fs")
const path = require("path")
const { ethers } = require("ethers")

const DEFAULT_MNEMONIC =
  "guard cream sadness conduct invite crumble clock pudding hole grit liar hotel maid produce squeeze return argue turtle know drive eight casino maze host"
const DEFAULT_RPC = process.env.JSON_RPC_ENDPOINT || "http://127.0.0.1:8545"

const ROOT = path.join(__dirname, "..", "..")
const PASSKEY_APP_DIR = path.join(ROOT, "passkey-app")
const CACHE_DIR = path.join(__dirname, "..", ".cache")
const OUT_JSON = path.join(CACHE_DIR, "passkey-demo.json")
const ENV_LOCAL = path.join(PASSKEY_APP_DIR, ".env.local")

function getArtifact(relPath) {
  const full = path.join(__dirname, relPath)
  if (!fs.existsSync(full)) {
    throw new Error(
      `Missing artifact at ${relPath}. Run "npx hardhat compile" in evm-e2e first.`
    )
  }
  return JSON.parse(fs.readFileSync(full, "utf8"))
}

async function main() {
  const provider = new ethers.JsonRpcProvider(DEFAULT_RPC)
  const chainIdHex = await provider.send("eth_chainId", [])
  const chainId = Number(chainIdHex)

  const signer = getSigner(provider)
  console.log("RPC:", DEFAULT_RPC, "chainId:", chainId)
  console.log("Deployer:", signer.address)

  const entrypointArtifact = getArtifact(
    "../artifacts/@account-abstraction/contracts/core/EntryPoint.sol/EntryPoint.json"
  )
  const factoryArtifact = getArtifact(
    "../artifacts/contracts/passkey/PasskeyAccount.sol/PasskeyAccountFactory.json"
  )

  console.log("Deploying EntryPoint...")
  const entryPoint = await new ethers.ContractFactory(
    entrypointArtifact.abi,
    entrypointArtifact.bytecode,
    signer
  ).deploy()
  await entryPoint.waitForDeployment()
  const entryPointAddr = await entryPoint.getAddress()
  console.log("EntryPoint deployed at:", entryPointAddr)

  console.log("Deploying PasskeyAccountFactory...")
  const passkeyFactory = await new ethers.ContractFactory(
    factoryArtifact.abi,
    factoryArtifact.bytecode,
    signer
  ).deploy(entryPointAddr)
  await passkeyFactory.waitForDeployment()
  const passkeyFactoryAddr = await passkeyFactory.getAddress()
  console.log("PasskeyAccountFactory deployed at:", passkeyFactoryAddr)

  const summary = {
    rpcUrl: DEFAULT_RPC,
    chainId,
    entryPoint: entryPointAddr,
    passkeyFactory: passkeyFactoryAddr,
    deployer: signer.address,
  }

  fs.mkdirSync(CACHE_DIR, { recursive: true })
  fs.writeFileSync(OUT_JSON, JSON.stringify(summary, null, 2))

  const envLines = [
    `VITE_RPC_URL=${DEFAULT_RPC}`,
    `VITE_CHAIN_ID=${chainId}`,
    `VITE_ENTRYPOINT=${entryPointAddr}`,
    `VITE_PASSKEY_FACTORY=${passkeyFactoryAddr}`,
    `VITE_DEFAULT_FROM=`,
    `VITE_SAMPLE_RECIPIENT=${signer.address}`,
  ]
  fs.writeFileSync(ENV_LOCAL, envLines.join("\n") + "\n")

  console.log("Wrote:", OUT_JSON)
  console.log("Wrote:", ENV_LOCAL)
  console.log("Ready: start the UI with npm run dev in passkey-app.")
}

function getSigner(provider) {
  if (process.env.PRIVATE_KEY) {
    return new ethers.Wallet(process.env.PRIVATE_KEY, provider)
  }
  const phrase = process.env.MNEMONIC || DEFAULT_MNEMONIC
  return ethers.Wallet.fromPhrase(phrase, provider)
}

main().catch((err) => {
  console.error("passkey-demo-setup failed:", err)
  process.exit(1)
})
