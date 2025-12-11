require("dotenv/config")
const { ethers } = require("hardhat")

async function main() {
  const deployer = getDeployer()
  console.log("Deployer:", deployer.address)
  console.log("Balance:", (await deployer.provider.getBalance(deployer.address)).toString())

  const EntryPoint = await ethers.getContractFactory("EntryPoint", deployer)
  const entryPoint = await EntryPoint.deploy()
  await entryPoint.waitForDeployment()

  const entryPointAddr = await entryPoint.getAddress()
  console.log("EntryPoint deployed at:", entryPointAddr)
}

function getDeployer() {
  const pk = process.env.PRIVATE_KEY
  const mnemonic = process.env.MNEMONIC
  if (!pk && !mnemonic) {
    throw new Error("Set PRIVATE_KEY or MNEMONIC in .env to sign deploy txs (eth_sendTransaction unsupported)")
  }

  const signer = pk
    ? new ethers.Wallet(pk, ethers.provider)
    : ethers.Wallet.fromPhrase(mnemonic, ethers.provider)

  return signer
}

main().catch((e) => {
  console.error(e)
  process.exitCode = 1
})
