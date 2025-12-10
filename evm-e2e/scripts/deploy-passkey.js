require("dotenv/config")
const { ethers } = require("hardhat")

const P256_PRECOMPILE = "0x0000000000000000000000000000000000000100"

async function main() {
  const entryPoint = process.env.ENTRY_POINT
  if (!entryPoint) {
    throw new Error("ENTRY_POINT env var is required (deployed ERC-4337 EntryPoint on Nibiru)")
  }

  console.log("Using EntryPoint:", entryPoint)
  console.log("Assuming P-256 precompile at:", P256_PRECOMPILE)

  const deployer = getDeployer()
  console.log("Deployer:", deployer.address)
  console.log("Balance:", (await deployer.provider.getBalance(deployer.address)).toString())

  const Factory = await ethers.getContractFactory("PasskeyAccountFactory", deployer)
  const factory = await Factory.deploy(entryPoint)
  await factory.waitForDeployment()
  console.log("PasskeyAccountFactory deployed to:", await factory.getAddress())

  const qx = process.env.QX
  const qy = process.env.QY
  if (qx && qy) {
    const predicted = await factory.accountAddress(qx, qy).catch(() => null)
    if (predicted) {
      console.log("Predicted PasskeyAccount:", predicted)
    }
    console.log("Creating PasskeyAccount with provided pubkey coords")
    const tx = await factory.createAccount(qx, qy)
    const receipt = await tx.wait()
    const created = receipt?.logs?.find((l) => l.fragment?.name === "AccountCreated")
    const acct =
      created?.args && "account" in created.args
        ? created.args.account
        : undefined
    console.log("PasskeyAccount created at:", acct ?? "<unknown>")
  } else {
  console.log("QX/QY not provided; skipped initial account creation")
  }
}

function getDeployer() {
  const pk = process.env.PRIVATE_KEY
  const mnemonic = process.env.MNEMONIC
  if (!pk && !mnemonic) {
    throw new Error("Set PRIVATE_KEY or MNEMONIC in .env to sign deploy txs (eth_sendTransaction unsupported)")
  }

  return pk
    ? new ethers.Wallet(pk, ethers.provider)
    : ethers.Wallet.fromPhrase(mnemonic, ethers.provider)
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
