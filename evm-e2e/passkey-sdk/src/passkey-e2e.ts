import { Contract, Interface, JsonRpcProvider, Wallet, formatEther, getBytes, parseEther } from "ethers"
import { sendUserOp, waitForUserOpReceipt } from "./bundler"
import { generateNodePasskey, signUserOpHash } from "./p256-node"
import { bytes32FromUint } from "./utils"
import { defaultUserOp, encodeSignature, getUserOpHash, UserOperation } from "./userop"

const RPC_URL = process.env.JSON_RPC_ENDPOINT ?? "http://127.0.0.1:8545"
const BUNDLER_URL = process.env.BUNDLER_URL ?? "http://127.0.0.1:4337"
const ENTRY_POINT = requireEnv("ENTRY_POINT")
const FACTORY_ADDR = requireEnv("FACTORY_ADDR")
const MNEMONIC = requireEnv("MNEMONIC")

const ACCOUNT_FUND_VALUE = process.env.PASSKEY_FUND_VALUE ?? "0.1"
const TRANSFER_VALUE = process.env.PASSKEY_TRANSFER_VALUE ?? "0.01"

const FACTORY_ABI = [
  "function createAccount(bytes32 qx, bytes32 qy) returns (address)",
  "event AccountCreated(address indexed account, bytes32 qx, bytes32 qy)",
]
const ACCOUNT_ABI = ["function nonce() view returns (uint256)"]
const ACCOUNT_INTERFACE = new Interface(["function execute(address to, uint256 value, bytes data)"])

function requireEnv(name: string): string {
  const value = process.env[name]
  if (!value) {
    throw new Error(`${name} environment variable must be set`)
  }
  return value
}

function getSeedFromEnv(): Uint8Array | undefined {
  const seedHex = process.env.PASSKEY_SEED
  if (!seedHex) return undefined
  const bytes = getBytes(seedHex)
  if (bytes.length !== 32) {
    throw new Error("PASSKEY_SEED must decode to 32 bytes")
  }
  return bytes
}

async function main() {
  const provider = new JsonRpcProvider(RPC_URL)
  const wallet = Wallet.fromPhrase(MNEMONIC, provider)
  console.log("Deployer:", wallet.address)

  // 1) "Register" a deterministic passkey for Node testing.
  const nodePasskey = generateNodePasskey(getSeedFromEnv())
  const qxHex = bytes32FromUint(nodePasskey.pubQx)
  const qyHex = bytes32FromUint(nodePasskey.pubQy)
  console.log("P256 qx:", qxHex)
  console.log("P256 qy:", qyHex)

  // 2) Deploy PasskeyAccount via the factory (predict address via staticCall).
  const factory = new Contract(FACTORY_ADDR, FACTORY_ABI, wallet)
  const predictedAccount = await factory.createAccount.staticCall(qxHex, qyHex)
  console.log("Predicted PasskeyAccount:", predictedAccount)

  const txCreate = await factory.createAccount(qxHex, qyHex)
  console.log("createAccount tx hash:", txCreate.hash)
  await txCreate.wait()

  const account = new Contract(predictedAccount, ACCOUNT_ABI, provider)
  const entryPointContract = new Contract(
    ENTRY_POINT,
    ["function depositTo(address account) payable", "function balanceOf(address account) view returns (uint256)"],
    wallet,
  )

  // 3) Fund the PasskeyAccount.
  const fundTx = await wallet.sendTransaction({
    to: predictedAccount,
    value: parseEther(ACCOUNT_FUND_VALUE),
  })
  console.log("Funding tx hash:", fundTx.hash)
  await fundTx.wait()

  const balanceBefore = await provider.getBalance(predictedAccount)
  console.log("Account balance before:", formatEther(balanceBefore), "NIBI")

  // 4) Build the UserOperation that sends value back to deployer.
  const callData = ACCOUNT_INTERFACE.encodeFunctionData("execute", [
    wallet.address,
    parseEther(TRANSFER_VALUE),
    "0x",
  ])

  const onChainNonce = (await account.nonce()) as bigint
  const feeData = await provider.getFeeData()
  const maxPriority = feeData.maxPriorityFeePerGas ?? 1_000_000_000n
  const maxFee = feeData.maxFeePerGas ?? maxPriority + 1_000_000_000n

  const userOp: UserOperation = {
    ...defaultUserOp(predictedAccount),
    nonce: onChainNonce,
    callData,
    maxFeePerGas: maxFee,
    maxPriorityFeePerGas: maxPriority,
  }

  const requiredPrefund =
    (userOp.callGasLimit + userOp.verificationGasLimit + userOp.preVerificationGas) * userOp.maxFeePerGas
  console.log("Depositing prefund:", formatEther(requiredPrefund), "NIBI")
  const depositTx = await entryPointContract.depositTo(predictedAccount, { value: requiredPrefund })
  await depositTx.wait()
  console.log(
    "EntryPoint deposit before bundling:",
    formatEther(await entryPointContract.balanceOf(predictedAccount)),
    "NIBI",
  )

  const chainId = BigInt((await provider.getNetwork()).chainId)
  const userOpHash = getUserOpHash(userOp, ENTRY_POINT, chainId)
  const { r, s } = signUserOpHash(userOpHash, nodePasskey.privKey)
  userOp.signature = encodeSignature({ r, s })
  console.log("Signature:", userOp.signature)

  // 5) Submit to the bundler and wait for inclusion.
  console.log("Sending UserOperation to bundler", BUNDLER_URL)
  const bundlerUserOpHash = await sendUserOp({
    bundlerUrl: BUNDLER_URL,
    userOp,
    entryPoint: ENTRY_POINT,
  })
  console.log("Bundler returned userOpHash:", bundlerUserOpHash)

  const userOpReceipt = await waitForUserOpReceipt({
    bundlerUrl: BUNDLER_URL,
    userOpHash: bundlerUserOpHash,
  })
  const txHash = userOpReceipt?.receipt?.transactionHash
  if (txHash) {
    console.log("UserOperation executed in tx:", txHash)
    await provider.waitForTransaction(txHash)
  } else {
    console.warn("Bundler receipt did not include transaction hash - continuing.")
  }

  // 6) Confirm nonce + balance changes on-chain.
  const nonceAfter = (await account.nonce()) as bigint
  const balanceAfter = await provider.getBalance(predictedAccount)
  const depositAfter = await entryPointContract.balanceOf(predictedAccount)
  console.log("Nonce after:", nonceAfter.toString())
  console.log("Account balance after:", formatEther(balanceAfter), "NIBI")
  console.log("EntryPoint deposit after:", formatEther(depositAfter), "NIBI")

  if (nonceAfter === onChainNonce + 1n && balanceAfter < balanceBefore) {
    console.log("✅ Passkey ERC-4337 flow completed successfully")
  } else {
    console.warn("⚠️ Passkey flow did not update account as expected")
  }
}

main().catch((err) => {
  console.error(err)
  process.exitCode = 1
})
