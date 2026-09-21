import {
  AbstractSigner,
  BaseWallet,
  Provider,
  TransactionRequest,
  TransactionResponse,
  TypedDataDomain,
  TypedDataField,
} from "ethers"

const NIBIRU_NONCE_MISMATCH =
  /invalid nonce;\s*got\s+(\d+),\s*expected\s+(\d+)\s+or higher/i
const ETHERS_NONCE_MESSAGES = /nonce (?:has already been used|too low)/i

export type NonceMismatch = {
  actual?: number
  expected?: number
}

const nestedErrorValues = (value: unknown): unknown[] => {
  if (typeof value !== "object" || value === null) return []

  const record = value as Record<string, unknown>
  return [
    record.code,
    record.message,
    record.shortMessage,
    record.error,
    record.info,
    record.cause,
    record.data,
  ]
}

/** Classify only errors that prove the submitted nonce is no longer usable. */
export const parseNonceMismatch = (error: unknown): NonceMismatch | null => {
  const pending: unknown[] = [error]
  const seen = new Set<object>()
  let ethersNonceExpired = false
  let ethersNonceMessage = false

  while (pending.length > 0) {
    const value = pending.pop()
    if (typeof value === "string") {
      const match = NIBIRU_NONCE_MISMATCH.exec(value)
      if (match !== null) {
        return { actual: Number(match[1]), expected: Number(match[2]) }
      }
      ethersNonceExpired ||= value === "NONCE_EXPIRED"
      ethersNonceMessage ||= ETHERS_NONCE_MESSAGES.test(value)
      continue
    }

    if (typeof value !== "object" || value === null || seen.has(value)) {
      continue
    }
    seen.add(value)
    pending.push(...nestedErrorValues(value))
  }

  return ethersNonceExpired && ethersNonceMessage ? {} : null
}

type NonceRequest = { nonce?: number | bigint | null }

export type NonceRetryOptions = {
  maxAttempts?: number
  onRetry?: (details: {
    attempt: number
    maxAttempts: number
    rejectedNonce: number
    retryNonce: number
  }) => void
}

/**
 * Allocate and repair nonces for one signer.
 *
 * The lock covers nonce selection through broadcast. Unknown broadcast errors
 * are never retried because the node may have accepted the transaction.
 */
export class NonceRetryManager {
  private readonly getPendingNonce: () => Promise<number>
  private readonly maxAttempts: number
  private readonly onRetry?: NonceRetryOptions["onRetry"]
  private nextNonce: number | null = null
  private tail: Promise<void> = Promise.resolve()

  constructor(
    getPendingNonce: () => Promise<number>,
    options: NonceRetryOptions = {},
  ) {
    this.getPendingNonce = getPendingNonce
    this.maxAttempts = options.maxAttempts ?? 3
    this.onRetry = options.onRetry
    if (!Number.isInteger(this.maxAttempts) || this.maxAttempts < 1) {
      throw new Error("maxAttempts must be a positive integer")
    }
  }

  async send<TRequest extends NonceRequest, TResult>(
    request: TRequest,
    broadcast: (request: TRequest & { nonce: number }) => Promise<TResult>,
  ): Promise<TResult> {
    return this.runExclusive(async () => {
      let nonce =
        request.nonce === undefined || request.nonce === null
          ? await this.allocateNonce()
          : Number(request.nonce)

      for (let attempt = 1; attempt <= this.maxAttempts; attempt++) {
        try {
          const result = await broadcast({ ...request, nonce })
          this.nextNonce = Math.max(this.nextNonce ?? 0, nonce + 1)
          return result
        } catch (error) {
          const mismatch = parseNonceMismatch(error)
          if (mismatch === null || attempt === this.maxAttempts) {
            this.nextNonce = null
            throw error
          }

          const rejectedNonce = nonce
          nonce = mismatch.expected ?? (await this.getPendingNonce())
          this.nextNonce = nonce
          this.onRetry?.({
            attempt,
            maxAttempts: this.maxAttempts,
            rejectedNonce,
            retryNonce: nonce,
          })
        }
      }

      throw new Error("unreachable nonce retry state")
    })
  }

  private async allocateNonce(): Promise<number> {
    if (this.nextNonce === null) {
      this.nextNonce = await this.getPendingNonce()
    }
    return this.nextNonce
  }

  private async runExclusive<TResult>(run: () => Promise<TResult>) {
    const previous = this.tail
    let release = () => {}
    this.tail = new Promise<void>((resolve) => {
      release = resolve
    })

    await previous
    try {
      return await run()
    } finally {
      release()
    }
  }
}

/** Ethers signer that coordinates and repairs transaction nonces. */
export class NonceRetryingSigner extends AbstractSigner<Provider> {
  readonly address: string
  private readonly wallet: BaseWallet
  private readonly manager: NonceRetryManager
  private readonly options: NonceRetryOptions

  constructor(wallet: BaseWallet, options: NonceRetryOptions = {}) {
    if (wallet.provider === null) {
      throw new Error("NonceRetryingSigner requires a connected wallet")
    }
    super(wallet.provider)
    this.wallet = wallet
    this.address = wallet.address
    this.options = options
    this.manager = new NonceRetryManager(
      () => this.provider.getTransactionCount(this.address, "pending"),
      options,
    )
  }

  getAddress(): Promise<string> {
    return Promise.resolve(this.address)
  }

  connect(provider: null | Provider): NonceRetryingSigner {
    if (provider === null) {
      throw new Error("NonceRetryingSigner requires a connected provider")
    }
    return new NonceRetryingSigner(this.wallet.connect(provider), this.options)
  }

  signTransaction(transaction: TransactionRequest): Promise<string> {
    return this.wallet.signTransaction(transaction)
  }

  signMessage(message: string | Uint8Array): Promise<string> {
    return this.wallet.signMessage(message)
  }

  signTypedData(
    domain: TypedDataDomain,
    types: Record<string, Array<TypedDataField>>,
    value: Record<string, unknown>,
  ): Promise<string> {
    return this.wallet.signTypedData(domain, types, value)
  }

  sendTransaction(
    transaction: TransactionRequest,
  ): Promise<TransactionResponse> {
    return this.manager.send(transaction, (request) =>
      this.wallet.sendTransaction(request),
    )
  }
}
