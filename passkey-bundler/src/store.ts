import { BundlerLogEntry, BundlerReceipt } from "./types"

export interface BundlerStore {
  saveReceipt(receipt: BundlerReceipt): Promise<void>
  getReceipt(hash: string): Promise<BundlerReceipt | null>
  appendLog(entry: BundlerLogEntry): Promise<void>
  getLogs(limit: number): Promise<BundlerLogEntry[]>
}

export class InMemoryStore implements BundlerStore {
  private receipts = new Map<string, BundlerReceipt>()
  private receiptLimit: number
  private logs: BundlerLogEntry[] = []
  private maxLogs: number

  constructor(receiptLimit: number, maxLogs = 500) {
    this.receiptLimit = receiptLimit
    this.maxLogs = maxLogs
  }

  async saveReceipt(receipt: BundlerReceipt): Promise<void> {
    this.receipts.set(receipt.userOpHash, receipt)
    while (this.receipts.size > this.receiptLimit) {
      const oldest = this.receipts.keys().next().value
      if (!oldest) break
      this.receipts.delete(oldest)
    }
  }

  async getReceipt(hash: string): Promise<BundlerReceipt | null> {
    return this.receipts.get(hash) ?? null
  }

  async appendLog(entry: BundlerLogEntry): Promise<void> {
    this.logs.push(entry)
    if (this.logs.length > this.maxLogs) {
      this.logs.splice(0, this.logs.length - this.maxLogs)
    }
  }

  async getLogs(limit: number): Promise<BundlerLogEntry[]> {
    if (limit <= 0) return []
    return this.logs.slice(-limit)
  }
}
