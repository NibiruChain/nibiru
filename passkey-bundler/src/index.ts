import cors from "@fastify/cors"
import Fastify, { FastifyReply, FastifyRequest } from "fastify"
import path from "node:path"
import { Contract, Interface, JsonRpcProvider, Wallet, toBeHex } from "ethers"

import { loadConfig } from "./config"
import { BundlerLogger } from "./logger"
import { Metrics } from "./metrics"
import { RateLimiter } from "./rateLimiter"
import { UserOpQueue } from "./queue"
import { SubmissionEngine } from "./submission"
import { BundlerConfig, BundlerReceipt, SubmissionJob } from "./types"
import { BundlerStore, InMemoryStore, SqliteStore } from "./store"
import { getUserOpHash, parseRpcUserOp } from "./userop"
import { estimateUserOperationGas, simulateValidationOrThrow } from "./validation"

const FACTORY_ABI = [
  "function createAccount(bytes32 _qx, bytes32 _qy) returns (address account)",
  "function accountAddress(bytes32 _qx, bytes32 _qy) view returns (address predicted)",
  "event AccountCreated(address indexed account, bytes32 qx, bytes32 qy, bytes32 salt)",
]

async function main() {
  let config = loadConfig()
  const store = createStore(config)
  const logger = new BundlerLogger(config.logLevel, (entry) => {
    void store.appendLog(entry).catch(() => {
      // ignore store logging errors
    })
  })
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

  await restoreQueuedUserOps({ config, store, queue, logger })

  const app = Fastify({ logger: false, bodyLimit: config.maxBodyBytes })
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
        body.map((p) =>
          handleRpcRequest(p, { request, reply, config, provider, queue, store, logger, metrics, rateLimiter, wallet }),
        ),
      )
      return responses.filter((r) => r !== null)
    }
    return handleRpcRequest(body, { request, reply, config, provider, queue, store, logger, metrics, rateLimiter, wallet })
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

function createStore(config: BundlerConfig): BundlerStore {
  if (!config.dbUrl) return new InMemoryStore(config.receiptLimit)
  if (config.dbUrl.startsWith("sqlite:")) {
    const raw = config.dbUrl.replace(/^sqlite:(\/\/)?/, "")
    const dbPath = raw.startsWith("/") ? raw : path.resolve(raw)
    return new SqliteStore({ dbPath, receiptLimit: config.receiptLimit })
  }
  return new SqliteStore({ dbPath: config.dbUrl, receiptLimit: config.receiptLimit })
}

async function restoreQueuedUserOps(opts: {
  config: BundlerConfig
  store: BundlerStore
  queue: UserOpQueue
  logger: BundlerLogger
}) {
  const { config, store, queue, logger } = opts
  const pending = await store.listUserOpsByStatus(["queued", "processing"], config.maxQueue)
  if (!pending.length) return

  let restored = 0
  for (const record of pending) {
    if (!record.rpcUserOp) {
      await store.upsertUserOp({ ...record, status: "failed", lastUpdated: Date.now(), revertReason: "Missing rpcUserOp" })
      continue
    }

    try {
      const { rpcUserOp, userOp } = parseRpcUserOp(record.rpcUserOp)
      const computedHash = getUserOpHash(userOp, config.entryPoint, config.chainId)
      if (computedHash.toLowerCase() !== record.userOpHash.toLowerCase()) {
        await store.upsertUserOp({
          ...record,
          status: "failed",
          lastUpdated: Date.now(),
          revertReason: "Stored userOpHash mismatch",
        })
        continue
      }

      queue.enqueue({
        rpcUserOp,
        userOp,
        userOpHash: record.userOpHash,
        receivedAt: record.receivedAt,
        requestId: record.requestId ?? null,
        remoteAddress: record.remoteAddress,
      })
      restored += 1
    } catch (err) {
      await store.upsertUserOp({
        ...record,
        status: "failed",
        lastUpdated: Date.now(),
        revertReason: (err as Error).message ?? "Failed to restore queued UserOperation",
      })
    }
  }

  if (restored > 0) {
    logger.info("Restored queued UserOperations from DB", { count: restored })
  }
}

interface RpcRequestContext {
  request: FastifyRequest
  reply: FastifyReply
  config: BundlerConfig
  provider: JsonRpcProvider
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
      case "eth_estimateUserOperationGas":
        return createResult(payload.id, await handleEstimateUserOperationGas(payload.params ?? [], payload.id ?? null, ctx))
      case "eth_sendUserOperation":
        return createResult(payload.id, await handleSendUserOperation(payload.params ?? [], payload.id ?? null, ctx))
      case "eth_getUserOperationReceipt":
        return createResult(payload.id, await handleGetUserOpReceipt(payload.params ?? [], store))
      case "passkey_createAccount":
        if (!config.enablePasskeyHelpers) return createError(payload.id, -32601, `Method ${method} not found`)
        return createResult(payload.id, await handleCreatePasskeyAccount(payload.params ?? [], wallet))
      case "passkey_fundAccount":
        if (!config.enablePasskeyHelpers) return createError(payload.id, -32601, `Method ${method} not found`)
        return createResult(payload.id, await handleFundAccount(payload.params ?? [], wallet))
      case "passkey_getLogs":
        if (!config.enablePasskeyHelpers) return createError(payload.id, -32601, `Method ${method} not found`)
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
  const { config, provider, queue, logger, request, store } = ctx
  if (params.length < 2) throw new Error("eth_sendUserOperation requires (userOp, entryPoint)")

  const entryPoint = (params[1] as string) ?? ""
  if (entryPoint.toLowerCase() !== config.entryPoint.toLowerCase()) {
    throw new Error(`Bundler configured for entryPoint ${config.entryPoint} but got ${entryPoint}`)
  }

  const { rpcUserOp, userOp } = parseRpcUserOp(params[0])
  const userOpHash = getUserOpHash(userOp, config.entryPoint, config.chainId)

  const existing = await store.getUserOp(userOpHash)
  if (existing) {
    if (existing.status === "rejected") {
      throw new Error(existing.revertReason ?? "UserOperation rejected")
    }
    if (existing.status !== "failed") {
      logger.info("Deduped UserOperation", { userOpHash, sender: existing.sender, status: existing.status })
      return userOpHash
    }
    logger.warn("Retrying failed UserOperation", { userOpHash, sender: existing.sender })
  }

  const receivedAt = Date.now()

  if (config.validationEnabled) {
    try {
      await simulateValidationOrThrow({ provider, entryPoint: config.entryPoint, userOp })
    } catch (err) {
      const message = (err as Error).message ?? "UserOperation validation failed"
      await store.upsertUserOp({
        userOpHash,
        entryPoint: config.entryPoint,
        sender: userOp.sender,
        nonce: toBeHex(userOp.nonce),
        rpcUserOp,
        receivedAt,
        lastUpdated: Date.now(),
        status: "rejected",
        revertReason: message,
        requestId,
        remoteAddress: request.ip,
      })
      throw err
    }
  }

  const job: SubmissionJob = {
    rpcUserOp,
    userOp,
    userOpHash,
    receivedAt,
    requestId,
    remoteAddress: request.ip,
  }

  await store.upsertUserOp({
    userOpHash,
    entryPoint: config.entryPoint,
    sender: userOp.sender,
    nonce: toBeHex(userOp.nonce),
    rpcUserOp,
    receivedAt: job.receivedAt,
    lastUpdated: Date.now(),
    status: "queued",
    requestId,
    remoteAddress: request.ip,
  })

  logger.info("Enqueuing UserOperation", { userOpHash, sender: userOp.sender })
  try {
    queue.enqueue(job)
  } catch (err) {
    const message = (err as Error).message ?? "Failed to enqueue UserOperation"
    await store.upsertUserOp({
      userOpHash,
      entryPoint: config.entryPoint,
      sender: userOp.sender,
      nonce: toBeHex(userOp.nonce),
      rpcUserOp,
      receivedAt: job.receivedAt,
      lastUpdated: Date.now(),
      status: "failed",
      revertReason: message,
      requestId,
      remoteAddress: request.ip,
    })
    throw err
  }
  return userOpHash
}

async function handleEstimateUserOperationGas(params: any[], _requestId: string | number | null, _ctx: RpcRequestContext) {
  if (params.length < 2) throw new Error("eth_estimateUserOperationGas requires (userOp, entryPoint)")
  const { config, provider, wallet } = _ctx
  const entryPoint = (params[1] as string) ?? ""
  if (entryPoint.toLowerCase() !== config.entryPoint.toLowerCase()) {
    throw new Error(`Bundler configured for entryPoint ${config.entryPoint} but got ${entryPoint}`)
  }

  const { userOp } = parseRpcUserOp(params[0])
  const beneficiary = config.beneficiary ?? wallet.address
  const estimated = await estimateUserOperationGas({ provider, entryPoint: config.entryPoint, beneficiary, userOp })
  return {
    callGasLimit: toBeHex(estimated.callGasLimit),
    verificationGasLimit: toBeHex(estimated.verificationGasLimit),
    preVerificationGas: toBeHex(estimated.preVerificationGas),
  }
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
  if (config.authRequired && !config.apiKeys.length) return false
  if (!config.apiKeys.length) return !config.authRequired
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
