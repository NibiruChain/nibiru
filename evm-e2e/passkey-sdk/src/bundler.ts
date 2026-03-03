import type { UserOperation } from "./userop"
import { toRpcUserOperation } from "./userop"

let rpcId = 0

type JsonRpcResponse<T> = { result?: T; error?: { code?: number; message?: string } }

async function rpcCall<T>(opts: { url: string; method: string; params: any[]; timeoutMs?: number }): Promise<T> {
  const { url, method, params, timeoutMs = 10_000 } = opts
  const controller = typeof AbortController !== "undefined" ? new AbortController() : undefined
  const timer = controller ? setTimeout(() => controller.abort(), timeoutMs) : undefined
  try {
    const res = await fetch(url, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ jsonrpc: "2.0", id: ++rpcId, method, params }),
      signal: controller?.signal,
    })
    if (!res.ok) {
      throw new Error(`RPC ${method} failed with status ${res.status}`)
    }
    const json = (await res.json()) as JsonRpcResponse<T>
    if (json.error) {
      throw new Error(json.error.message ?? `RPC ${method} returned error`)
    }
    return json.result as T
  } finally {
    if (timer) clearTimeout(timer)
  }
}

const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms))

export async function sendUserOp(opts: {
  bundlerUrl: string
  userOp: UserOperation
  entryPoint: string
  timeoutMs?: number
}) {
  const rpcUserOp = toRpcUserOperation(opts.userOp)
  return rpcCall<string>({
    url: opts.bundlerUrl,
    method: "eth_sendUserOperation",
    params: [rpcUserOp, opts.entryPoint],
    timeoutMs: opts.timeoutMs,
  })
}

export async function waitForUserOpReceipt(opts: {
  bundlerUrl: string
  userOpHash: string
  pollIntervalMs?: number
  timeoutMs?: number
  onError?: (err: unknown) => void
}) {
  const { bundlerUrl, userOpHash, pollIntervalMs = 2000, timeoutMs = 60_000, onError } = opts
  const deadline = Date.now() + timeoutMs

  while (Date.now() < deadline) {
    try {
      const receipt = await rpcCall<any>({
        url: bundlerUrl,
        method: "eth_getUserOperationReceipt",
        params: [userOpHash],
        timeoutMs: pollIntervalMs,
      })
      if (receipt) return receipt
    } catch (err) {
      onError?.(err)
    }
    await sleep(pollIntervalMs)
  }
  throw new Error(`timed out waiting for userOp receipt: ${userOpHash}`)
}
