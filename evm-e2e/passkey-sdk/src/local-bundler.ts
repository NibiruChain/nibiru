import http, { IncomingMessage } from "http"
import { Contract, ContractTransactionReceipt, Interface, JsonRpcProvider, Wallet, toBeHex } from "ethers"
import { getUserOpHash, rpcUserOpToStruct, RpcUserOperation } from "./userop"

const RPC_URL = process.env.JSON_RPC_ENDPOINT ?? "http://127.0.0.1:8545"
const ENTRY_POINT = requireEnv("ENTRY_POINT")
const PORT = Number(process.env.BUNDLER_PORT ?? "4337")
const FACTORY_ADDR = process.env.FACTORY_ADDR
const PRIVATE_KEY =
  process.env.BUNDLER_PRIVATE_KEY ??
  "0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d" // dev key

type BundlerLogEntry = { ts: number; level: string; message: string }
const LOG_BUFFER: BundlerLogEntry[] = []
const MAX_LOGS = 500
const originalLog = console.log
const originalError = console.error

function pushLog(level: string, ...args: unknown[]) {
  const message = args
    .map((a) => {
      try {
        if (typeof a === "string") return a
        return JSON.stringify(a)
      } catch {
        return String(a)
      }
    })
    .join(" ")
  LOG_BUFFER.push({ ts: Date.now(), level, message })
  if (LOG_BUFFER.length > MAX_LOGS) {
    LOG_BUFFER.shift()
  }
}

console.log = (...args: unknown[]) => {
  pushLog("info", ...args)
  originalLog(...args)
}

console.error = (...args: unknown[]) => {
  pushLog("error", ...args)
  originalError(...args)
}

const provider = new JsonRpcProvider(RPC_URL)
const wallet = new Wallet(PRIVATE_KEY, provider)
const ENTRY_POINT_ABI = [
  "function handleOps((address sender,uint256 nonce,bytes initCode,bytes callData,uint256 callGasLimit,uint256 verificationGasLimit,uint256 preVerificationGas,uint256 maxFeePerGas,uint256 maxPriorityFeePerGas,bytes paymasterAndData,bytes signature)[] ops, address payable beneficiary)",
  "function depositTo(address) payable",
]
const FACTORY_ABI = [
  "function createAccount(bytes32 _qx, bytes32 _qy) returns (address account)",
  "event AccountCreated(address indexed account, bytes32 qx, bytes32 qy)",
]
const entryPoint = new Contract(ENTRY_POINT, new Interface(ENTRY_POINT_ABI), wallet)
const factoryInterface = new Interface(FACTORY_ABI)

type JsonRpcId = number | string | null
interface JsonRpcRequest {
  id: JsonRpcId
  jsonrpc: string
  method: string
  params?: any[]
}

interface BundlerReceipt {
  userOpHash: string
  entryPoint: string
  sender: string
  nonce: string
  actualGasCost?: string
  actualGasUsed?: string
  success: boolean
  receipt: {
    transactionHash: string
  }
}

const receiptStore = new Map<string, BundlerReceipt>()

async function main() {
  const network = await provider.getNetwork()
  const chainId = process.env.CHAIN_ID ? BigInt(process.env.CHAIN_ID) : BigInt(network.chainId)
  const chainIdHex = toBeHex(chainId)
  console.log(
    `Local bundler listening on port ${PORT} (RPC=${RPC_URL}, entryPoint=${ENTRY_POINT}, chainId=${chainIdHex}, signer=${wallet.address})`,
  )

  const server = http.createServer(async (req, res) => {
    const headers = {
      "Content-Type": "application/json",
      "Access-Control-Allow-Origin": "*",
      "Access-Control-Allow-Methods": "POST, OPTIONS",
      "Access-Control-Allow-Headers": "Content-Type",
    }

    if (req.method === "OPTIONS") {
      res.writeHead(204, headers).end()
      return
    }

    if (req.method !== "POST") {
      res.writeHead(405, headers).end()
      return
    }

    try {
      const body = await readBody(req)
      const payload = JSON.parse(body)
      if (Array.isArray(payload)) {
        const responses = await Promise.all(payload.map((reqItem) => handleRpcRequest(reqItem, chainId, chainIdHex)))
        res.writeHead(200, headers).end(JSON.stringify(responses.filter((r): r is object => !!r)))
      } else {
        const response = await handleRpcRequest(payload, chainId, chainIdHex)
        res.writeHead(200, headers).end(JSON.stringify(response ?? null))
      }
    } catch (err) {
      console.error("bundler request failed:", err)
      res
        .writeHead(500, headers)
        .end(JSON.stringify({ jsonrpc: "2.0", error: { code: -32603, message: "Internal bundler error" } }))
    }
  })

  server.listen(PORT, () => {
    console.log(`Bundler JSON-RPC listening at http://127.0.0.1:${PORT}`)
  })
}

async function handleRpcRequest(
  payload: JsonRpcRequest,
  chainId: bigint,
  chainIdHex: string,
): Promise<object | null> {
  if (!payload || typeof payload.method !== "string") {
    return createError(payload?.id ?? null, -32600, "Invalid Request")
  }

  switch (payload.method) {
    case "eth_chainId":
      return createResult(payload.id, chainIdHex)
    case "eth_supportedEntryPoints":
      return createResult(payload.id, [ENTRY_POINT])
    case "eth_sendUserOperation":
      return createResult(payload.id, await handleSendUserOperation(payload.params ?? [], chainId))
    case "eth_getUserOperationReceipt":
      return createResult(payload.id, handleGetUserOpReceipt(payload.params ?? []))
    case "passkey_createAccount":
      return createResult(payload.id, await handleCreatePasskeyAccount(payload.params ?? []))
    case "passkey_fundAccount":
      return createResult(payload.id, await handleFundAccount(payload.params ?? []))
    case "passkey_getLogs":
      return createResult(payload.id, handleGetLogs(payload.params ?? []))
    default:
      return createError(payload.id, -32601, `Method ${payload.method} not found`)
  }
}

async function handleSendUserOperation(params: any[], chainId: bigint): Promise<string> {
  if (params.length < 2) throw new Error("eth_sendUserOperation requires (userOp, entryPoint)")
  const rpcUserOp = params[0] as RpcUserOperation
  const entryPointAddr = (params[1] as string) ?? ""
  if (entryPointAddr.toLowerCase() !== ENTRY_POINT.toLowerCase()) {
    throw new Error(`Bundler configured for entryPoint ${ENTRY_POINT} but got ${entryPointAddr}`)
  }

  console.log("Bundling UserOperation", rpcUserOp)
  const userOp = rpcUserOpToStruct(rpcUserOp)
  const userOpHash = getUserOpHash(userOp, ENTRY_POINT, chainId)
  console.log("Received UserOperation", {
    userOpHash,
    sender: rpcUserOp.sender,
    nonce: rpcUserOp.nonce,
  })

  // Simple pre-fund: deposit the required amount into EntryPoint for this sender.
  const requiredPrefund =
    (userOp.callGasLimit + userOp.verificationGasLimit + userOp.preVerificationGas) * userOp.maxFeePerGas
  if (requiredPrefund > 0n) {
    console.log(`Prefunding sender ${rpcUserOp.sender} with ${requiredPrefund} wei in EntryPoint`)
    await entryPoint.depositTo(rpcUserOp.sender, { value: requiredPrefund })
  }

  const tx = await entryPoint.handleOps([userOp], wallet.address, {
    gasLimit: userOp.callGasLimit + userOp.verificationGasLimit + userOp.preVerificationGas + 200000n,
  })
  console.log(`handleOps tx broadcast: ${tx.hash}`)

  tx
    .wait()
    .then((receipt: ContractTransactionReceipt) => {
      const actualGasUsed = toBeHex(receipt.gasUsed ?? 0n)
      const actualGasCost = receipt.gasPrice ? toBeHex((receipt.gasUsed ?? 0n) * receipt.gasPrice) : actualGasUsed
      receiptStore.set(userOpHash, {
        userOpHash,
        entryPoint: ENTRY_POINT,
        sender: rpcUserOp.sender,
        nonce: rpcUserOp.nonce,
        actualGasCost,
        actualGasUsed,
        success: Boolean(receipt.status),
        receipt: { transactionHash: receipt.hash },
      })
      console.log(`UserOperation ${userOpHash} executed in tx ${receipt.hash} (gas ${actualGasUsed})`)
    })
    .catch((err: unknown) => {
      console.error("handleOps failed", err)
      receiptStore.set(userOpHash, {
        userOpHash,
        entryPoint: ENTRY_POINT,
        sender: rpcUserOp.sender,
        nonce: rpcUserOp.nonce,
        success: false,
        receipt: { transactionHash: tx.hash },
      })
    })

  return userOpHash
}

function handleGetUserOpReceipt(params: any[]): BundlerReceipt | null {
  if (!params.length) throw new Error("eth_getUserOperationReceipt requires userOpHash")
  const hash = params[0] as string
  return receiptStore.get(hash) ?? null
}

function handleGetLogs(params: any[]): BundlerLogEntry[] {
  const limit = Number(params?.[0] ?? MAX_LOGS)
  if (!Number.isFinite(limit) || limit <= 0) return []
  return LOG_BUFFER.slice(-limit)
}

async function handleCreatePasskeyAccount(params: any[]): Promise<{ account: string; txHash: string }> {
  const qx = params[0] as string | undefined
  const qy = params[1] as string | undefined
  const factoryAddr = (params[2] as string | undefined) ?? FACTORY_ADDR
  if (!factoryAddr) {
    throw new Error("Factory address missing (set FACTORY_ADDR env var or pass as param[2])")
  }
  if (!qx || !qy) {
    throw new Error("passkey_createAccount requires qx and qy (bytes32 hex strings)")
  }

  const factory = new Contract(factoryAddr, FACTORY_ABI, wallet)
  const tx = await factory.createAccount(qx, qy)
  const receipt = await tx.wait()

  let account: string | undefined
  for (const log of receipt.logs ?? []) {
    try {
      const parsed = factoryInterface.parseLog(log)
      if (parsed?.name === "AccountCreated" && parsed.args?.account) {
        account = parsed.args.account as string
        break
      }
    } catch {
      // ignore non-matching logs
    }
  }

  if (!account) {
    // fall back to static call for predicted address if the event was not found
    try {
      account = await factory.createAccount.staticCall(qx, qy)
    } catch {
      // ignore
    }
  }

  if (!account) {
    throw new Error("AccountCreated event not found; account address unavailable")
  }

  return { account, txHash: tx.hash }
}

async function handleFundAccount(params: any[]): Promise<{ txHash: string }> {
  const to = params[0] as string | undefined
  const amount = params[1] ? BigInt(params[1] as string) : 1_000_000_000_000_000_000n
  if (!to) {
    throw new Error("passkey_fundAccount requires target address")
  }
  const tx = await wallet.sendTransaction({ to, value: amount })
  const receipt = await tx.wait()
  console.log(`Funded ${to} with ${amount} wei from bundler wallet (tx ${receipt?.hash ?? tx.hash})`)
  return { txHash: receipt?.hash ?? tx.hash }
}

function readBody(req: IncomingMessage): Promise<string> {
  return new Promise((resolve, reject) => {
    let data = ""
    req.on("data", (chunk) => {
      data += chunk
    })
    req.on("end", () => resolve(data))
    req.on("error", reject)
  })
}

function createResult(id: JsonRpcId, result: any) {
  return { jsonrpc: "2.0", id, result }
}

function createError(id: JsonRpcId, code: number, message: string) {
  return { jsonrpc: "2.0", id, error: { code, message } }
}

function requireEnv(name: string): string {
  const value = process.env[name]
  if (!value) {
    throw new Error(`Missing required env var ${name}`)
  }
  return value
}

main().catch((err) => {
  console.error("Bundler failed to start:", err)
  process.exit(1)
})
