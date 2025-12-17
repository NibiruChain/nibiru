import { Contract, Interface, JsonRpcProvider, TransactionReceipt, Wallet, toBeHex } from "ethers"

import { BundlerConfig, BundlerReceipt, SubmissionJob } from "./types"
import { BundlerLogger } from "./logger"
import { Metrics } from "./metrics"
import { BundlerStore } from "./store"
import { entryPointInterfaceV06 } from "./entryPoint"
import { getUserOpHash, requiredPrefund } from "./userop"

const standardErrorInterface = new Interface(["error Error(string)", "error Panic(uint256)"])

export class NonceManager {
  private nextNonce?: bigint
  private lock: Promise<void> = Promise.resolve()
  private readonly inFlight = new Set<bigint>()
  private readonly wallet: Wallet
  private readonly logger: BundlerLogger

  constructor(wallet: Wallet, logger: BundlerLogger) {
    this.wallet = wallet
    this.logger = logger
  }

  async init(): Promise<void> {
    await this.enqueue(async () => {
      await this.initUnlocked()
    })
  }

  async reserve(): Promise<bigint> {
    return this.enqueue(async () => {
      if (this.nextNonce === undefined) {
        await this.initUnlocked()
      }
      let nonce = this.nextNonce as bigint
      while (this.inFlight.has(nonce)) {
        nonce += 1n
      }
      this.inFlight.add(nonce)
      this.nextNonce = nonce + 1n
      return nonce
    })
  }

  async onSendFailure(nonce: bigint, err?: unknown): Promise<void> {
    await this.enqueue(async () => {
      this.inFlight.delete(nonce)
      const chainNonce = BigInt(await this.wallet.getNonce("pending"))
      // drop any tracked nonces that are now below chain pending
      for (const n of this.inFlight) {
        if (n < chainNonce) {
          this.inFlight.delete(n)
        }
      }

      let next = this.nextNonce ?? chainNonce
      if (nonce < next) next = nonce
      if (chainNonce > next) next = chainNonce
      while (this.inFlight.has(next)) {
        next += 1n
      }
      this.nextNonce = next

      this.logger.warn("Resynced bundler nonce after send failure", {
        failedNonce: nonce.toString(),
        chainNonce: chainNonce.toString(),
        nextNonce: next.toString(),
        error: (err as Error | undefined)?.message,
      })
    })
  }

  async onMined(nonce: bigint): Promise<void> {
    await this.enqueue(async () => {
      this.inFlight.delete(nonce)
    })
  }

  private async initUnlocked(): Promise<void> {
    this.nextNonce = BigInt(await this.wallet.getNonce("pending"))
    this.inFlight.clear()
    this.logger.info("Initialized bundler nonce", { nonce: this.nextNonce.toString() })
  }

  private enqueue<T>(fn: () => Promise<T>): Promise<T> {
    const run = this.lock.then(fn, fn)
    this.lock = run.then(
      () => undefined,
      () => undefined,
    )
    return run
  }
}

export class SubmissionEngine {
  private readonly config: BundlerConfig
  private readonly provider: JsonRpcProvider
  private readonly wallet: Wallet
  private readonly store: BundlerStore
  private readonly logger: BundlerLogger
  private readonly metrics: Metrics
  private readonly entryPoint: Contract
  private readonly entryPointInterface: Interface
  private readonly nonceManager: NonceManager
  private pollingReceipts = false

  constructor(opts: {
    config: BundlerConfig
    provider: JsonRpcProvider
    wallet: Wallet
    store: BundlerStore
    logger: BundlerLogger
    metrics: Metrics
  }) {
    this.config = opts.config
    this.provider = opts.provider
    this.wallet = opts.wallet
    this.store = opts.store
    this.logger = opts.logger
    this.metrics = opts.metrics
    this.entryPointInterface = entryPointInterfaceV06
    this.entryPoint = new Contract(this.config.entryPoint, this.entryPointInterface, this.wallet)
    this.nonceManager = new NonceManager(this.wallet, this.logger)
  }

  async start(): Promise<void> {
    await this.nonceManager.init()
    this.startReceiptPoller()
  }

  async process(job: SubmissionJob): Promise<void> {
    const start = Date.now()
    await this.store.upsertUserOp({
      userOpHash: job.userOpHash,
      entryPoint: this.config.entryPoint,
      sender: job.userOp.sender,
      nonce: toBeHex(job.userOp.nonce),
      receivedAt: job.receivedAt,
      lastUpdated: Date.now(),
      status: "processing",
      requestId: job.requestId,
      remoteAddress: job.remoteAddress,
    })

    try {
      const hash = getUserOpHash(job.userOp, this.config.entryPoint, this.config.chainId)
      if (hash.toLowerCase() !== job.userOpHash.toLowerCase()) {
        throw new Error("UserOp hash mismatch after validation")
      }
      if (this.config.prefundEnabled) {
        await this.ensurePrefund(job)
      }

      const submission = await this.submitWithRetries(job)
      if (submission.kind === "submitted") {
        await this.store.upsertUserOp({
          userOpHash: job.userOpHash,
          entryPoint: this.config.entryPoint,
          sender: job.userOp.sender,
          nonce: toBeHex(job.userOp.nonce),
          receivedAt: job.receivedAt,
          lastUpdated: Date.now(),
          status: "submitted",
          txHash: submission.txHash,
          requestId: job.requestId,
          remoteAddress: job.remoteAddress,
        })
        return
      }

      if (submission.receipt.status !== 1) {
        throw new Error("handleOps transaction reverted")
      }

      const outcome = this.extractUserOpOutcome(submission.receipt, job.userOpHash)
      if (!outcome) {
        throw new Error("Missing UserOperationEvent in handleOps receipt")
      }

      await this.store.saveReceipt({
        userOpHash: job.userOpHash,
        entryPoint: this.config.entryPoint,
        sender: job.userOp.sender,
        nonce: toBeHex(job.userOp.nonce),
        actualGasCost: outcome.actualGasCost,
        actualGasUsed: outcome.actualGasUsed,
        success: outcome.success,
        revertReason: outcome.revertReason,
        receipt: { transactionHash: submission.txHash },
        receivedAt: job.receivedAt,
        lastUpdated: Date.now(),
      })

      if (outcome.success) {
        this.metrics.userOpSuccess.inc()
      } else {
        this.metrics.userOpFailed.inc({ reason: "execution" })
      }
    } catch (err) {
      this.metrics.userOpFailed.inc({ reason: "submission" })
      const message = (err as Error).message
      this.logger.error("UserOperation submission failed", {
        userOpHash: job.userOpHash,
        sender: job.userOp.sender,
        error: message,
      })
      await this.store.upsertUserOp({
        userOpHash: job.userOpHash,
        entryPoint: this.config.entryPoint,
        sender: job.userOp.sender,
        nonce: toBeHex(job.userOp.nonce),
        receivedAt: job.receivedAt,
        lastUpdated: Date.now(),
        status: "failed",
        revertReason: message,
        requestId: job.requestId,
        remoteAddress: job.remoteAddress,
      })
    } finally {
      this.metrics.submissionDuration.observe(Date.now() - start)
    }
  }

  private async ensurePrefund(job: SubmissionJob) {
    if (
      this.config.prefundAllowlist.length > 0 &&
      !this.config.prefundAllowlist.some((a) => a.toLowerCase() === job.userOp.sender.toLowerCase())
    ) {
      throw new Error(`Prefund not allowed for sender ${job.userOp.sender}`)
    }

    // If a paymaster is provided, it should cover gas; do not top up the sender deposit.
    if (job.userOp.paymasterAndData && job.userOp.paymasterAndData !== "0x") {
      return
    }

    const needed = requiredPrefund(job.userOp)
    if (needed === 0n) return
    const current = (await this.entryPoint.balanceOf(job.userOp.sender)) as bigint
    if (current >= needed) return
    const topUp = needed - current
    if (topUp > this.config.maxPrefundWei) {
      throw new Error(
        `Prefund requirement ${topUp.toString()} wei exceeds configured maxPrefundWei (${this.config.maxPrefundWei.toString()})`,
      )
    }

    this.metrics.prefundAttempts.inc()
    this.logger.info("Prefunding sender in EntryPoint", {
      sender: job.userOp.sender,
      needed: topUp.toString(),
      current: current.toString(),
    })
    const nonce = await this.nonceManager.reserve()
    let tx: any
    try {
      tx = await this.entryPoint.depositTo(job.userOp.sender, {
        value: topUp,
        nonce,
      })
    } catch (err) {
      this.metrics.prefundFailures.inc()
      await this.nonceManager.onSendFailure(nonce, err)
      throw err
    }

    try {
      await tx.wait()
      await this.nonceManager.onMined(nonce)
    } catch (err) {
      this.metrics.prefundFailures.inc()
      throw err
    }
  }

  private async submitWithRetries(
    job: SubmissionJob,
  ): Promise<{ kind: "submitted"; txHash: string } | { kind: "mined"; txHash: string; receipt: TransactionReceipt }> {
    const beneficiary = this.config.beneficiary ?? this.wallet.address
    const data = this.entryPointInterface.encodeFunctionData("handleOps", [[job.userOp], beneficiary])
    const gasLimit =
      job.userOp.callGasLimit + job.userOp.verificationGasLimit + job.userOp.preVerificationGas + 200_000n
    const nonce = await this.nonceManager.reserve()
    let lastTxHash: string | undefined

    const feeData = await this.provider.getFeeData()
    let maxPriorityFeePerGas =
      feeData.maxPriorityFeePerGas && feeData.maxPriorityFeePerGas > 0n
        ? feeData.maxPriorityFeePerGas
        : job.userOp.maxPriorityFeePerGas
    let maxFeePerGas =
      feeData.maxFeePerGas && feeData.maxFeePerGas > 0n ? feeData.maxFeePerGas : job.userOp.maxFeePerGas

    maxPriorityFeePerGas = capFee(maxPriorityFeePerGas, job.userOp.maxPriorityFeePerGas)
    maxFeePerGas = capFee(maxFeePerGas, job.userOp.maxFeePerGas)

    const attempts = 3
    for (let i = 0; i < attempts; i += 1) {
      try {
        const tx = await this.wallet.sendTransaction({
          to: this.config.entryPoint,
          data,
          gasLimit,
          maxFeePerGas,
          maxPriorityFeePerGas,
          nonce,
        })
        lastTxHash = tx.hash
        this.logger.info("Broadcasted handleOps", { txHash: tx.hash, userOpHash: job.userOpHash, attempt: i + 1 })
        await this.store.upsertUserOp({
          userOpHash: job.userOpHash,
          entryPoint: this.config.entryPoint,
          sender: job.userOp.sender,
          nonce: toBeHex(job.userOp.nonce),
          receivedAt: job.receivedAt,
          lastUpdated: Date.now(),
          status: "submitted",
          txHash: tx.hash,
          requestId: job.requestId,
          remoteAddress: job.remoteAddress,
        })
        const receipt = await this.provider.waitForTransaction(
          tx.hash,
          this.config.finalityBlocks,
          this.config.submissionTimeoutMs,
        )
        if (!receipt) {
          throw new Error("Timed out waiting for handleOps receipt")
        }
        await this.nonceManager.onMined(nonce)
        return { kind: "mined", txHash: tx.hash, receipt }
      } catch (err) {
        const message = (err as Error).message ?? "handleOps failed"
        if (i === attempts - 1) {
          if (!lastTxHash) {
            await this.nonceManager.onSendFailure(nonce, err)
            throw err
          }
          this.logger.warn("handleOps submission did not finalize; leaving as pending", {
            userOpHash: job.userOpHash,
            txHash: lastTxHash,
            error: message,
          })
          return { kind: "submitted", txHash: lastTxHash }
        }
        const bumpedFees = bumpFees(
          { maxFeePerGas, maxPriorityFeePerGas },
          { gasBumpPercent: this.config.gasBumpPercent, gasBumpWei: this.config.gasBumpWei },
          job.userOp,
        )
        maxFeePerGas = bumpedFees.maxFeePerGas
        maxPriorityFeePerGas = bumpedFees.maxPriorityFeePerGas
        this.logger.warn("handleOps failed; retrying with bumped gas", {
          attempt: i + 1,
          error: message,
          maxFeePerGas: maxFeePerGas.toString(),
          maxPriorityFeePerGas: maxPriorityFeePerGas.toString(),
        })
      }
    }
    if (lastTxHash) return { kind: "submitted", txHash: lastTxHash }
    throw new Error("handleOps retries exhausted")
  }

  private extractUserOpOutcome(
    receipt: TransactionReceipt,
    userOpHash: string,
  ): { success: boolean; actualGasCost?: string; actualGasUsed?: string; revertReason?: string } | null {
    const wanted = userOpHash.toLowerCase()
    let opEvent: any | undefined
    let revertData: string | undefined

    for (const log of receipt.logs ?? []) {
      if ((log.address ?? "").toLowerCase() !== this.config.entryPoint.toLowerCase()) continue
      try {
        const parsed = this.entryPointInterface.parseLog(log)
        if (parsed.name === "UserOperationEvent" && String(parsed.args.userOpHash).toLowerCase() === wanted) {
          opEvent = parsed.args
        } else if (
          parsed.name === "UserOperationRevertReason" &&
          String(parsed.args.userOpHash).toLowerCase() === wanted
        ) {
          revertData = parsed.args.revertReason as string
        }
      } catch {
        // ignore unrelated logs
      }
    }

    if (!opEvent) return null

    const success = Boolean(opEvent.success)
    const actualGasCost = typeof opEvent.actualGasCost !== "undefined" ? toBeHex(opEvent.actualGasCost) : undefined
    const actualGasUsed = typeof opEvent.actualGasUsed !== "undefined" ? toBeHex(opEvent.actualGasUsed) : undefined
    const revertReason = success ? undefined : decodeRevertData(revertData)
    return { success, actualGasCost, actualGasUsed, revertReason }
  }

  private startReceiptPoller() {
    const interval = this.config.receiptPollIntervalMs
    if (interval <= 0) return
    void this.pollPendingReceipts().catch((err) => {
      this.logger.warn("Pending receipt poll failed", { error: (err as Error).message })
    })
    setInterval(() => {
      void this.pollPendingReceipts().catch((err) => {
        this.logger.warn("Pending receipt poll failed", { error: (err as Error).message })
      })
    }, interval)
  }

  private async pollPendingReceipts() {
    if (this.pollingReceipts) return
    this.pollingReceipts = true
    try {
      const pending = await this.store.listUserOpsByStatus(["submitted"], 50)
      if (!pending.length) return
      const currentBlock = await this.provider.getBlockNumber()

      for (const record of pending) {
        if (!record.txHash) continue
        const receipt = await this.provider.getTransactionReceipt(record.txHash)
        if (!receipt) continue

        const confirmations = receipt.blockNumber ? currentBlock - receipt.blockNumber + 1 : 0
        if (confirmations < this.config.finalityBlocks) continue

        if (receipt.status !== 1) {
          await this.store.upsertUserOp({
            ...record,
            lastUpdated: Date.now(),
            status: "failed",
            revertReason: "handleOps transaction reverted",
          })
          this.metrics.userOpFailed.inc({ reason: "submission" })
          continue
        }

        const outcome = this.extractUserOpOutcome(receipt, record.userOpHash)
        if (!outcome) {
          await this.store.upsertUserOp({
            ...record,
            lastUpdated: Date.now(),
            status: "failed",
            revertReason: "Missing UserOperationEvent in handleOps receipt",
          })
          this.metrics.userOpFailed.inc({ reason: "submission" })
          continue
        }

        await this.store.saveReceipt({
          userOpHash: record.userOpHash,
          entryPoint: record.entryPoint,
          sender: record.sender,
          nonce: record.nonce,
          actualGasCost: outcome.actualGasCost,
          actualGasUsed: outcome.actualGasUsed,
          success: outcome.success,
          revertReason: outcome.revertReason,
          receipt: { transactionHash: record.txHash },
          receivedAt: record.receivedAt,
          lastUpdated: Date.now(),
        })

        if (outcome.success) {
          this.metrics.userOpSuccess.inc()
        } else {
          this.metrics.userOpFailed.inc({ reason: "execution" })
        }
      }
    } finally {
      this.pollingReceipts = false
    }
  }
}

function decodeRevertData(data?: string): string | undefined {
  if (!data || data === "0x") return undefined
  try {
    const parsed = standardErrorInterface.parseError(data)
    if (parsed.name === "Error") return parsed.args[0] as string
    if (parsed.name === "Panic") return `Panic(${String(parsed.args[0])})`
  } catch {
    // ignore
  }
  return data
}

function capFee(value: bigint, cap: bigint): bigint {
  return value > cap ? cap : value
}

function bumpFees(
  current: { maxFeePerGas: bigint; maxPriorityFeePerGas: bigint },
  bump: { gasBumpPercent: number; gasBumpWei?: bigint },
  userOp: SubmissionJob["userOp"],
): { maxFeePerGas: bigint; maxPriorityFeePerGas: bigint } {
  const increment = (fee: bigint) => {
    const pctBump = (fee * BigInt(bump.gasBumpPercent)) / 100n
    const absBump = bump.gasBumpWei ?? 0n
    return fee + (pctBump > absBump ? pctBump : absBump)
  }

  const nextPriority = capFee(increment(current.maxPriorityFeePerGas), userOp.maxPriorityFeePerGas)
  const nextMaxFee = capFee(increment(current.maxFeePerGas), userOp.maxFeePerGas)
  return { maxFeePerGas: nextMaxFee, maxPriorityFeePerGas: nextPriority }
}
