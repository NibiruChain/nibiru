import { Contract, Interface, JsonRpcProvider, Wallet, toBeHex } from "ethers"

import { BundlerConfig, BundlerReceipt, SubmissionJob } from "./types"
import { BundlerLogger } from "./logger"
import { Metrics } from "./metrics"
import { BundlerStore } from "./store"
import { getUserOpHash, requiredPrefund } from "./userop"

const ENTRY_POINT_ABI = [
  "function handleOps((address sender,uint256 nonce,bytes initCode,bytes callData,uint256 callGasLimit,uint256 verificationGasLimit,uint256 preVerificationGas,uint256 maxFeePerGas,uint256 maxPriorityFeePerGas,bytes paymasterAndData,bytes signature)[] ops, address payable beneficiary)",
  "function depositTo(address) payable",
  "function balanceOf(address) view returns (uint256)",
]

class NonceManager {
  private nextNonce?: bigint
  private lock: Promise<bigint> = Promise.resolve(0n)
  private readonly wallet: Wallet
  private readonly logger: BundlerLogger

  constructor(wallet: Wallet, logger: BundlerLogger) {
    this.wallet = wallet
    this.logger = logger
  }

  async init(): Promise<void> {
    this.nextNonce = BigInt(await this.wallet.getNonce("pending"))
    this.logger.info("Initialized bundler nonce", { nonce: this.nextNonce.toString() })
  }

  async reserve(): Promise<bigint> {
    this.lock = this.lock.then(async () => {
      if (this.nextNonce === undefined) {
        await this.init()
      }
      const current = this.nextNonce as bigint
      this.nextNonce = current + 1n
      return current
    })
    return this.lock
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
    this.entryPointInterface = new Interface(ENTRY_POINT_ABI)
    this.entryPoint = new Contract(this.config.entryPoint, this.entryPointInterface, this.wallet)
    this.nonceManager = new NonceManager(this.wallet, this.logger)
  }

  async start(): Promise<void> {
    await this.nonceManager.init()
  }

  async process(job: SubmissionJob): Promise<void> {
    const start = Date.now()
    try {
      const hash = getUserOpHash(job.userOp, this.config.entryPoint, this.config.chainId)
      if (hash.toLowerCase() !== job.userOpHash.toLowerCase()) {
        throw new Error("UserOp hash mismatch after validation")
      }
      if (this.config.prefundEnabled) {
        await this.ensurePrefund(job)
      }

      const receipt = await this.submitWithRetries(job)
      await this.store.saveReceipt({
        userOpHash: job.userOpHash,
        entryPoint: this.config.entryPoint,
        sender: job.userOp.sender,
        nonce: toBeHex(job.userOp.nonce),
        actualGasCost: receipt.actualGasCost,
        actualGasUsed: receipt.actualGasUsed,
        success: receipt.success,
        revertReason: receipt.revertReason,
        receipt: { transactionHash: receipt.txHash },
        receivedAt: job.receivedAt,
        lastUpdated: Date.now(),
      })
      if (receipt.success) {
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
      await this.store.saveReceipt({
        userOpHash: job.userOpHash,
        entryPoint: this.config.entryPoint,
        sender: job.userOp.sender,
        nonce: toBeHex(job.userOp.nonce),
        success: false,
        revertReason: message,
        receipt: { transactionHash: "0x" },
        receivedAt: job.receivedAt,
        lastUpdated: Date.now(),
      })
    } finally {
      this.metrics.submissionDuration.observe(Date.now() - start)
    }
  }

  private async ensurePrefund(job: SubmissionJob) {
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
    try {
      const nonce = await this.nonceManager.reserve()
      const tx = await this.entryPoint.depositTo(job.userOp.sender, {
        value: topUp,
        nonce,
      })
      await tx.wait()
    } catch (err) {
      this.metrics.prefundFailures.inc()
      throw err
    }
  }

  private async submitWithRetries(
    job: SubmissionJob,
  ): Promise<{ txHash: string; success: boolean; actualGasUsed?: string; actualGasCost?: string; revertReason?: string }> {
    const beneficiary = this.config.beneficiary ?? this.wallet.address
    const data = this.entryPointInterface.encodeFunctionData("handleOps", [[job.userOp], beneficiary])
    const gasLimit =
      job.userOp.callGasLimit + job.userOp.verificationGasLimit + job.userOp.preVerificationGas + 200_000n
    const nonce = await this.nonceManager.reserve()

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
        this.logger.info("Broadcasted handleOps", { txHash: tx.hash, userOpHash: job.userOpHash, attempt: i + 1 })
        const receipt = await this.provider.waitForTransaction(tx.hash, this.config.finalityBlocks, this.config.submissionTimeoutMs)
        if (!receipt) {
          throw new Error("Timed out waiting for handleOps receipt")
        }
        const actualGasUsed = receipt.gasUsed ? toBeHex(receipt.gasUsed) : undefined
        const actualGasCost =
          receipt.gasUsed && (tx.maxFeePerGas ?? tx.gasPrice)
            ? toBeHex(receipt.gasUsed * (tx.maxFeePerGas ?? tx.gasPrice!))
            : undefined
        if (receipt.status === 1) {
          return { txHash: tx.hash, success: true, actualGasCost, actualGasUsed }
        }
        return { txHash: tx.hash, success: false, actualGasCost, actualGasUsed, revertReason: "Execution reverted" }
      } catch (err) {
        const message = (err as Error).message ?? "handleOps failed"
        if (i === attempts - 1) {
          return { txHash: "0x", success: false, revertReason: message }
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
    return { txHash: "0x", success: false, revertReason: "handleOps retries exhausted" }
  }
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
