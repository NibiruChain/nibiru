const { ethers } = require("hardhat")

async function main() {
  const [deployer] = await ethers.getSigners()
  console.log("Deployer:", deployer.address)

  const EntryPoint = await ethers.getContractFactory("EntryPoint")
  const entryPoint = await EntryPoint.deploy()
  await entryPoint.waitForDeployment()

  const entryPointAddr = await entryPoint.getAddress()
  console.log("EntryPoint deployed at:", entryPointAddr)
}

main().catch((e) => {
  console.error(e)
  process.exitCode = 1
})

