export interface RpcUserOperation {
  sender: string
  nonce: string
  initCode: string
  callData: string
  callGasLimit: string
  verificationGasLimit: string
  preVerificationGas: string
  maxFeePerGas: string
  maxPriorityFeePerGas: string
  paymasterAndData: string
  signature: string
}

export interface UserOperation {
  sender: string
  nonce: bigint
  initCode: string
  callData: string
  callGasLimit: bigint
  verificationGasLimit: bigint
  preVerificationGas: bigint
  maxFeePerGas: bigint
  maxPriorityFeePerGas: bigint
  paymasterAndData: string
  signature: string
}

export interface BundlerConfig {
  mode: "dev" | "testnet"
  rpcUrl: string
  entryPoint: string
  chainId: bigint
  bundlerPrivateKey: string
  beneficiary?: string
  port: number
  metricsPort?: number
  maxBodyBytes: number
  authRequired: boolean
  rateLimitPerMinute: number
  apiKeys: string[]
  maxQueue: number
  queueConcurrency: number
  logLevel: LogLevel
  enablePasskeyHelpers: boolean
  dbUrl?: string
  validationEnabled: boolean
  gasBumpPercent: number
  gasBumpWei?: bigint
  submissionTimeoutMs: number
  finalityBlocks: number
  receiptLimit: number
  receiptPollIntervalMs: number
}

export interface BundlerReceipt {
  userOpHash: string
  entryPoint: string
  sender: string
  nonce: string
  actualGasCost?: string
  actualGasUsed?: string
  success: boolean
  revertReason?: string
  receipt: {
    transactionHash: string
  }
  receivedAt: number
  lastUpdated: number
}

export type LogLevel = "debug" | "info" | "warn" | "error"

export interface BundlerLogEntry {
  ts: number
  level: LogLevel
  message: string
  meta?: Record<string, unknown>
}

export interface SubmissionJob {
  rpcUserOp: RpcUserOperation
  userOp: UserOperation
  userOpHash: string
  receivedAt: number
  requestId: string | number | null
  remoteAddress?: string
}

export type UserOpStatus = "queued" | "processing" | "submitted" | "included" | "rejected" | "failed"

export interface UserOpRecord {
  userOpHash: string
  entryPoint: string
  sender: string
  nonce: string
  rpcUserOp?: RpcUserOperation
  receivedAt: number
  lastUpdated: number
  status: UserOpStatus
  txHash?: string
  actualGasCost?: string
  actualGasUsed?: string
  success?: boolean
  revertReason?: string
  requestId?: string | number | null
  remoteAddress?: string
}

export interface HealthStatus {
  status: "ok" | "error"
  details?: Record<string, unknown>
}
