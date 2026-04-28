import { config } from "dotenv"
import { ethers, Wallet } from "ethers"

config()

const JSON_RPC_ENDPOINT =
  process.env.JSON_RPC_ENDPOINT ?? "http://127.0.0.1:8545"
/** Maximum time to wait for EVM JSON-RPC readiness before failing tests. */
const RPC_READY_TIMEOUT_MS = 24000
/** How often to print progress while waiting for EVM JSON-RPC readiness. */
const RPC_READY_PULSE_MS = 3000
/** How frequently to probe `eth_chainId` while waiting for readiness. */
const RPC_READY_RETRY_MS = 250

/**
 * Poll `eth_chainId` until EVM JSON-RPC is ready, or throw after timeout.
 *
 * Localnet brings up multiple RPC surfaces with different startup times.
 * In CI benchmarking, the EVM JSON-RPC endpoint (:8545) is typically the last
 * to become available (~20s), so tests gate on this endpoint to avoid
 * intermittent "connection refused" races during early startup.
 */
const waitForEvmJsonRpcReady = async () => {
  const startedAt = Date.now()
  let lastPulseAt = 0
  let attempts = 0
  let lastError = "unknown error"

  while (Date.now() - startedAt < RPC_READY_TIMEOUT_MS) {
    attempts++
    try {
      const response = await fetch(JSON_RPC_ENDPOINT, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          jsonrpc: "2.0",
          method: "eth_chainId",
          params: [],
          id: 1,
        }),
      })

      if (!response.ok) {
        lastError = `http ${response.status}`
      } else {
        const payload = (await response.json()) as {
          result?: string
          error?: unknown
        }
        if (typeof payload.result === "string" && payload.result.length > 0) {
          return
        }
        lastError = payload.error
          ? JSON.stringify(payload.error)
          : "missing result in response"
      }
    } catch (err) {
      lastError = err instanceof Error ? err.message : String(err)
    }

    const now = Date.now()
    if (now - lastPulseAt >= RPC_READY_PULSE_MS) {
      const elapsed = now - startedAt
      console.log(
        `[evm-e2e setup] waiting for EVM JSON-RPC (${elapsed}ms/${RPC_READY_TIMEOUT_MS}ms) at ${JSON_RPC_ENDPOINT}; last error: ${lastError}`,
      )
      lastPulseAt = now
    }

    await new Promise((resolve) => setTimeout(resolve, RPC_READY_RETRY_MS))
  }

  throw new Error(
    `[evm-e2e setup] timed out after ${RPC_READY_TIMEOUT_MS}ms waiting for EVM JSON-RPC at ${JSON_RPC_ENDPOINT} (attempts=${attempts}, last error=${lastError})`,
  )
}

let preflightPromise: Promise<void> | null = null
let preflightError: unknown = null

/**
 * Run one-time test preflight for EVM RPC readiness.
 *
 * - First caller starts the check.
 * - Concurrent callers await the same in-flight promise.
 * - If preflight failed once, all later callers fail immediately with
 *   the same error.
 */
const preflight = async (): Promise<void> => {
  if (preflightError !== null) {
    throw preflightError
  }

  if (preflightPromise === null) {
    preflightPromise = waitForEvmJsonRpcReady().catch((err) => {
      preflightError = err
      throw err
    })
  }

  await preflightPromise
}

// Hard-fail early at module load for suites importing testdeps.ts.
await preflight()

const provider: ethers.JsonRpcProvider = await (async () => {
  await preflight()
  return new ethers.JsonRpcProvider(JSON_RPC_ENDPOINT)
})()

/**
 * `account` is the primary funded signer for many of the EVM E2E tests.
 * The seed phrase, `process.env.MNEMONIC`, is set by
 * `contrib/scripts/localnet.sh` to create the `validator` key on
 * the `nibiru-localnet-0` blockchain. That validator/dev account is
 * funded in genesis and can deploy contracts, pay gas, and fund
 * other test wallets.
 */
const account = Wallet.fromPhrase(process.env.MNEMONIC, provider)

const TEST_TIMEOUT = Number(process.env.TEST_TIMEOUT) || 15000
const TX_WAIT_TIMEOUT = Number(process.env.TX_WAIT_TIMEOUT) || 5000

export { account, provider, preflight, TEST_TIMEOUT, TX_WAIT_TIMEOUT }
