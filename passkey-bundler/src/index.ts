import cors from "@fastify/cors"
import Fastify, { FastifyReply, FastifyRequest } from "fastify"
import { Contract, Interface, JsonRpcProvider, Wallet, toBeHex } from "ethers"

import { loadConfig } from "./config"
import { BundlerLogger } from "./logger"
import { Metrics } from "./metrics"
import { RateLimiter } from "./rateLimiter"
import { UserOpQueue } from "./queue"
import { SubmissionEngine } from "./submission"
import { BundlerConfig, BundlerReceipt, SubmissionJob } from "./types"
import { BundlerStore, InMemoryStore } from "./store"
import { getUserOpHash, parseRpcUserOp } from "./userop"

const FACTORY_ABI = [
  "function createAccount(bytes32 _qx, bytes32 _qy) returns (address account)",
  "function accountAddress(bytes32 _qx, bytes32 _qy) view returns (address predicted)",
  "event AccountCreated(address indexed account, bytes32 qx, bytes32 qy, bytes32 salt)",
]

async function main() {
  let config = loadConfig()
  const store = new InMemoryStore(config.receiptLimit)
  const logger = new BundlerLogger(config.logLevel, (entry) => store.appendLog(entry))
  const metrics = new Metrics()
  const rateLimiter = new RateLimiter(config.rateLimitPerMinute)

  const provider = new JsonRpcProvider(config.rpcUrl)
  const network = await provider.getNetwork()
  const resolvedChainId = config.chainId !== 0n ? config.chainId : BigInt(network.chainId)
  if (config.chainId !== 0n && config.chainId !== BigInt(network.chainId)) {
    logger.warn("Configured chainId differs from RPC network chainId", {
      configured: config.chainId.toString(),
      rpc: network.chainId.toString(),
    })
  }

  config = { ...config, chainId: resolvedChainId } as BundlerConfig

  const wallet = new Wallet(config.bundlerPrivateKey, provider)
  const submission = new SubmissionEngine({ config, provider, wallet, store, logger, metrics })
  await submission.start()

  const queue = new UserOpQueue({
    concurrency: config.queueConcurrency,
    maxSize: config.maxQueue,
    onDepthChange: (depth) => metrics.setQueueDepth(depth),
    processor: (job) => submission.process(job),
  })

  const app = Fastify({ logger: false })
  await app.register(cors, { origin: true })

  app.addHook("onRequest", (_req, reply, done) => {
    reply.header("Access-Control-Allow-Origin", "*")
    reply.header("Access-Control-Allow-Headers", "Content-Type, Authorization, x-api-key")
    done()
  })

  app.post("/", async (request, reply) => {
    const body = request.body as any
    if (!body) {
      reply.status(400)
      return { error: "Missing request body" }
    }

    if (Array.isArray(body)) {
      const responses = await Promise.all(
        body.map((p) => handleRpcRequest(p, { request, reply, config, queue, store, logger, metrics, rateLimiter, wallet })),
      )
      return responses.filter((r) => r !== null)
    }
    return handleRpcRequest(body, { request, reply, config, queue, store, logger, metrics, rateLimiter, wallet })
  })

  app.get("/healthz", async (_req, reply) => {
    try {
      const blockNumber = await provider.getBlockNumber()
      reply.send({ status: "ok", rpc: true, block: blockNumber })
    } catch (err) {
      reply.status(500).send({ status: "error", rpc: false, error: (err as Error).message })
    }
  })

  app.get("/readyz", async (_req, reply) => {
    try {
      const [blockNumber, nonce] = await Promise.all([provider.getBlockNumber(), wallet.getNonce("pending")])
      reply.send({
        status: "ok",
        rpc: true,
        block: blockNumber,
        signerNonce: nonce,
        chainId: toBeHex(config.chainId),
      })
    } catch (err) {
      reply.status(500).send({ status: "error", rpc: false, error: (err as Error).message })
    }
  })

  app.get("/metrics", async (_req, reply) => {
    reply.header("Content-Type", "text/plain; version=0.0.4")
    reply.send(await metrics.render())
  })

  await app.listen({ port: config.port, host: "0.0.0.0" })
  logger.info("Bundler listening", { port: config.port, rpcUrl: config.rpcUrl, entryPoint: config.entryPoint })

  if (config.metricsPort && config.metricsPort !== config.port) {
    const metricsApp = Fastify({ logger: false })
    await metricsApp.register(cors, { origin: true })
    metricsApp.get("/metrics", async (_req, reply) => {
      reply.header("Content-Type", "text/plain; version=0.0.4")
      reply.send(await metrics.render())
    })
    metricsApp.get("/healthz", async (_req, reply) => {
      reply.send({ status: "ok" })
    })
    await metricsApp.listen({ port: config.metricsPort, host: "0.0.0.0" })
    logger.info("Metrics endpoint listening", { port: config.metricsPort })
  }
}

interface RpcRequestContext {
  request: FastifyRequest
  reply: FastifyReply
  config: BundlerConfig
  queue: UserOpQueue
  store: BundlerStore
  logger: BundlerLogger
  metrics: Metrics
  rateLimiter: RateLimiter
  wallet: Wallet
}

async function handleRpcRequest(payload: any, ctx: RpcRequestContext): Promise<object | null> {
  const { request, reply, config, queue, store, logger, metrics, rateLimiter, wallet } = ctx

  if (!payload || typeof payload.method !== "string") {
    metrics.rpcFailures.inc({ method: "unknown" })
    return createError(payload?.id ?? null, -32600, "Invalid Request")
  }

  const method = payload.method as string
  metrics.rpcRequests.inc({ method })

  const authKey = extractAuthKey(request)
  const rateKey = authKey ?? request.ip
  if (!authorize(authKey, config)) {
    metrics.rpcFailures.inc({ method })
    return createError(payload.id, -32001, "Unauthorized")
  }

  if (!rateLimiter.allow(rateKey)) {
    metrics.rpcFailures.inc({ method })
    return createError(payload.id, -32001, "Rate limit exceeded")
  }

  try {
    switch (method) {
      case "eth_chainId":
        return createResult(payload.id, toBeHex(config.chainId))
      case "eth_supportedEntryPoints":
        return createResult(payload.id, [config.entryPoint])
      case "eth_sendUserOperation":
        return createResult(payload.id, await handleSendUserOperation(payload.params ?? [], payload.id ?? null, ctx))
      case "eth_getUserOperationReceipt":
        return createResult(payload.id, await handleGetUserOpReceipt(payload.params ?? [], store))
      case "passkey_createAccount":
        return createResult(payload.id, await handleCreatePasskeyAccount(payload.params ?? [], wallet))
      case "passkey_fundAccount":
        return createResult(payload.id, await handleFundAccount(payload.params ?? [], wallet))
      case "passkey_getLogs":
        return createResult(payload.id, await handleGetLogs(payload.params ?? [], store))
      default:
        return createError(payload.id, -32601, `Method ${method} not found`)
    }
  } catch (err) {
    metrics.rpcFailures.inc({ method })
    logger.error("RPC method failed", { method, error: (err as Error).message })
    return createError(payload.id, -32603, (err as Error).message)
  }
}

async function handleSendUserOperation(params: any[], requestId: string | number | null, ctx: RpcRequestContext): Promise<string> {
  const { config, queue, logger, request } = ctx
  if (params.length < 2) throw new Error("eth_sendUserOperation requires (userOp, entryPoint)")

  const entryPoint = (params[1] as string) ?? ""
  if (entryPoint.toLowerCase() !== config.entryPoint.toLowerCase()) {
    throw new Error(`Bundler configured for entryPoint ${config.entryPoint} but got ${entryPoint}`)
  }

  const { rpcUserOp, userOp } = parseRpcUserOp(params[0])
  const userOpHash = getUserOpHash(userOp, config.entryPoint, config.chainId)

  const job: SubmissionJob = {
    rpcUserOp,
    userOp,
    userOpHash,
    receivedAt: Date.now(),
    requestId,
    remoteAddress: request.ip,
  }

  logger.info("Enqueuing UserOperation", { userOpHash, sender: userOp.sender })
  queue.enqueue(job)
  return userOpHash
}

async function handleGetUserOpReceipt(params: any[], store: BundlerStore): Promise<BundlerReceipt | null> {
  if (!params.length) throw new Error("eth_getUserOperationReceipt requires userOpHash")
  const hash = params[0] as string
  return store.getReceipt(hash)
}

async function handleCreatePasskeyAccount(params: any[], wallet: Wallet): Promise<{ account: string; txHash: string }> {
  const qx = params[0] as string | undefined
  const qy = params[1] as string | undefined
  const factoryAddr = params[2] as string | undefined

  if (!factoryAddr) throw new Error("Factory address missing (param[2])")
  if (!qx || !qy) throw new Error("passkey_createAccount requires qx and qy")

  const iface = new Interface(FACTORY_ABI)
  const factory = new Contract(factoryAddr, iface, wallet)
  const predicted = (await factory.accountAddress(qx, qy)) as string
  const tx = await factory.createAccount(qx, qy)
  const receipt = await tx.wait()
  const account = parseAccountCreated(receipt?.logs, iface) ?? predicted
  return { account, txHash: receipt?.hash ?? tx.hash }
}

async function handleFundAccount(params: any[], wallet: Wallet): Promise<{ txHash: string }> {
  const to = params[0] as string | undefined
  const amount = params[1] ? BigInt(params[1] as string) : 1_000_000_000_000_000_000n
  if (!to) throw new Error("passkey_fundAccount requires target address")
  const tx = await wallet.sendTransaction({ to, value: amount })
  const receipt = await tx.wait()
  return { txHash: receipt?.hash ?? tx.hash }
}

async function handleGetLogs(params: any[], store: BundlerStore) {
  const limit = Number(params?.[0] ?? 100)
  if (!Number.isFinite(limit) || limit <= 0) return []
  return store.getLogs(limit)
}

function createResult(id: any, result: any) {
  return { jsonrpc: "2.0", id: id ?? null, result }
}

function createError(id: any, code: number, message: string) {
  return { jsonrpc: "2.0", id: id ?? null, error: { code, message } }
}

function authorize(apiKey: string | undefined, config: BundlerConfig): boolean {
  if (!config.apiKeys.length) return true
  if (!apiKey) return false
  return config.apiKeys.includes(apiKey)
}

function extractAuthKey(request: FastifyRequest): string | undefined {
  const header = (request.headers["x-api-key"] as string | undefined) ?? (request.headers.authorization as string | undefined)
  if (!header) return undefined
  if (header.startsWith("Bearer ")) return header.slice("Bearer ".length).trim()
  return header.trim()
}

function parseAccountCreated(logs: any[] | undefined, iface: Interface): string | undefined {
  for (const log of logs ?? []) {
    try {
      const parsed = iface.parseLog(log)
      if (parsed?.name === "AccountCreated" && parsed.args?.account) {
        return parsed.args.account as string
      }
    } catch {
      // ignore
    }
  }
  return undefined
}

main().catch((err) => {
  // eslint-disable-next-line no-console
  console.error("Bundler failed to start:", err)
  process.exit(1)
})
