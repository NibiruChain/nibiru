import { JsonRpcProvider } from "ethers"
import type { UserOperation } from "./userop"
import { toRpcUserOperation } from "./userop"

export async function sendUserOp(opts: {
  bundlerUrl: string
  userOp: UserOperation
  entryPoint: string
}) {
  const provider = new JsonRpcProvider(opts.bundlerUrl)
  const rpcUserOp = toRpcUserOperation(opts.userOp)
  return provider.send("eth_sendUserOperation", [rpcUserOp, opts.entryPoint])
}

export async function waitForUserOpReceipt(opts: {
  bundlerUrl: string
  userOpHash: string
  pollIntervalMs?: number
  timeoutMs?: number
}) {
  const { bundlerUrl, userOpHash, pollIntervalMs = 2000, timeoutMs = 60_000 } = opts
  const provider = new JsonRpcProvider(bundlerUrl)
  const started = Date.now()

  while (Date.now() - started < timeoutMs) {
    try {
      const receipt = await provider.send("eth_getUserOperationReceipt", [userOpHash])
      if (receipt) return receipt
    } catch (err) {
      // Bundler returns errors until the UserOp is indexed; ignore and retry.
      if (process.env.DEBUG) console.debug("waitForUserOpReceipt retry:", err)
    }
    await new Promise((resolve) => setTimeout(resolve, pollIntervalMs))
  }
  throw new Error(`timed out waiting for userOp receipt: ${userOpHash}`)
}
