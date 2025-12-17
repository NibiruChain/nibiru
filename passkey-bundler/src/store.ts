import fs from "node:fs"
import path from "node:path"

import Database from "better-sqlite3"

import { BundlerLogEntry, BundlerReceipt, UserOpRecord, UserOpStatus } from "./types"

export interface BundlerStore {
  upsertUserOp(record: UserOpRecord): Promise<void>
  getUserOp(hash: string): Promise<UserOpRecord | null>
  listUserOpsByStatus(statuses: UserOpStatus[], limit: number): Promise<UserOpRecord[]>
  saveReceipt(receipt: BundlerReceipt): Promise<void>
  getReceipt(hash: string): Promise<BundlerReceipt | null>
  appendLog(entry: BundlerLogEntry): Promise<void>
  getLogs(limit: number): Promise<BundlerLogEntry[]>
}

export class InMemoryStore implements BundlerStore {
  private readonly userOps = new Map<string, UserOpRecord>()
  private receipts = new Map<string, BundlerReceipt>()
  private receiptLimit: number
  private logs: BundlerLogEntry[] = []
  private maxLogs: number

  constructor(receiptLimit: number, maxLogs = 500) {
    this.receiptLimit = receiptLimit
    this.maxLogs = maxLogs
  }

  async upsertUserOp(record: UserOpRecord): Promise<void> {
    const existing = this.userOps.get(record.userOpHash)
    const merged: UserOpRecord = {
      ...(existing ?? {}),
      ...record,
      rpcUserOp: record.rpcUserOp ?? existing?.rpcUserOp,
    }
    this.userOps.set(record.userOpHash, merged)
  }

  async getUserOp(hash: string): Promise<UserOpRecord | null> {
    return this.userOps.get(hash) ?? null
  }

  async listUserOpsByStatus(statuses: UserOpStatus[], limit: number): Promise<UserOpRecord[]> {
    if (!statuses.length || limit <= 0) return []
    const items: UserOpRecord[] = []
    for (const rec of this.userOps.values()) {
      if (statuses.includes(rec.status)) items.push(rec)
    }
    items.sort((a, b) => a.lastUpdated - b.lastUpdated)
    return items.slice(0, limit)
  }

  async saveReceipt(receipt: BundlerReceipt): Promise<void> {
    this.receipts.set(receipt.userOpHash, receipt)
    await this.upsertUserOp({
      userOpHash: receipt.userOpHash,
      entryPoint: receipt.entryPoint,
      sender: receipt.sender,
      nonce: receipt.nonce,
      receivedAt: receipt.receivedAt,
      lastUpdated: receipt.lastUpdated,
      status: "included",
      txHash: receipt.receipt.transactionHash,
      actualGasCost: receipt.actualGasCost,
      actualGasUsed: receipt.actualGasUsed,
      success: receipt.success,
      revertReason: receipt.revertReason,
    })
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

export class SqliteStore implements BundlerStore {
  private readonly db: Database.Database
  private readonly receiptLimit: number
  private readonly maxLogs: number

  constructor(opts: { dbPath: string; receiptLimit: number; maxLogs?: number }) {
    this.receiptLimit = opts.receiptLimit
    this.maxLogs = opts.maxLogs ?? 2000

    const resolvedPath = path.resolve(opts.dbPath)
    fs.mkdirSync(path.dirname(resolvedPath), { recursive: true })

    this.db = new Database(resolvedPath)
    this.db.pragma("journal_mode = WAL")
    this.db.pragma("busy_timeout = 5000")
    this.initSchema()
  }

  async upsertUserOp(record: UserOpRecord): Promise<void> {
    const stmt = this.db.prepare(`
      INSERT INTO user_ops (
        user_op_hash, entry_point, sender, nonce,
        rpc_user_op_json,
        received_at, last_updated, status,
        tx_hash, actual_gas_cost, actual_gas_used,
        success, revert_reason, request_id, remote_address
      ) VALUES (
        @userOpHash, @entryPoint, @sender, @nonce,
        @rpcUserOpJson,
        @receivedAt, @lastUpdated, @status,
        @txHash, @actualGasCost, @actualGasUsed,
        @success, @revertReason, @requestId, @remoteAddress
      )
      ON CONFLICT(user_op_hash) DO UPDATE SET
        entry_point=excluded.entry_point,
        sender=excluded.sender,
        nonce=excluded.nonce,
        rpc_user_op_json=COALESCE(excluded.rpc_user_op_json, user_ops.rpc_user_op_json),
        received_at=excluded.received_at,
        last_updated=excluded.last_updated,
        status=excluded.status,
        tx_hash=excluded.tx_hash,
        actual_gas_cost=excluded.actual_gas_cost,
        actual_gas_used=excluded.actual_gas_used,
        success=excluded.success,
        revert_reason=excluded.revert_reason,
        request_id=excluded.request_id,
        remote_address=excluded.remote_address
    `)

    stmt.run({
      userOpHash: record.userOpHash,
      entryPoint: record.entryPoint,
      sender: record.sender,
      nonce: record.nonce,
      rpcUserOpJson: record.rpcUserOp ? JSON.stringify(record.rpcUserOp) : null,
      receivedAt: record.receivedAt,
      lastUpdated: record.lastUpdated,
      status: record.status,
      txHash: record.txHash ?? null,
      actualGasCost: record.actualGasCost ?? null,
      actualGasUsed: record.actualGasUsed ?? null,
      success: typeof record.success === "boolean" ? (record.success ? 1 : 0) : null,
      revertReason: record.revertReason ?? null,
      requestId: record.requestId !== undefined && record.requestId !== null ? String(record.requestId) : null,
      remoteAddress: record.remoteAddress ?? null,
    })
  }

  async getUserOp(hash: string): Promise<UserOpRecord | null> {
    const row = this.db
      .prepare(`SELECT * FROM user_ops WHERE user_op_hash = ?`)
      .get(hash) as any
    if (!row) return null
    return this.rowToUserOp(row)
  }

  async listUserOpsByStatus(statuses: UserOpStatus[], limit: number): Promise<UserOpRecord[]> {
    if (!statuses.length || limit <= 0) return []
    const placeholders = statuses.map(() => "?").join(",")
    const rows = this.db
      .prepare(
        `SELECT * FROM user_ops WHERE status IN (${placeholders}) ORDER BY last_updated ASC LIMIT ?`,
      )
      .all(...statuses, limit) as any[]
    return rows.map((r) => this.rowToUserOp(r))
  }

  async saveReceipt(receipt: BundlerReceipt): Promise<void> {
    await this.upsertUserOp({
      userOpHash: receipt.userOpHash,
      entryPoint: receipt.entryPoint,
      sender: receipt.sender,
      nonce: receipt.nonce,
      receivedAt: receipt.receivedAt,
      lastUpdated: receipt.lastUpdated,
      status: "included",
      txHash: receipt.receipt.transactionHash,
      actualGasCost: receipt.actualGasCost,
      actualGasUsed: receipt.actualGasUsed,
      success: receipt.success,
      revertReason: receipt.revertReason,
    })

    this.pruneIncludedReceipts()
  }

  async getReceipt(hash: string): Promise<BundlerReceipt | null> {
    const row = this.db
      .prepare(`SELECT * FROM user_ops WHERE user_op_hash = ? AND status = 'included'`)
      .get(hash) as any
    if (!row) return null
    return this.rowToReceipt(row)
  }

  async appendLog(entry: BundlerLogEntry): Promise<void> {
    this.db
      .prepare(`INSERT INTO logs (ts, level, message, meta_json) VALUES (?, ?, ?, ?)`)
      .run(entry.ts, entry.level, entry.message, entry.meta ? JSON.stringify(entry.meta) : null)

    this.pruneLogs()
  }

  async getLogs(limit: number): Promise<BundlerLogEntry[]> {
    if (limit <= 0) return []
    const rows = this.db.prepare(`SELECT ts, level, message, meta_json FROM logs ORDER BY id DESC LIMIT ?`).all(limit) as any[]
    rows.reverse()
    return rows.map((r) => ({
      ts: r.ts,
      level: r.level,
      message: r.message,
      meta: r.meta_json ? JSON.parse(r.meta_json) : undefined,
    }))
  }

  private initSchema() {
    this.db.exec(`
      CREATE TABLE IF NOT EXISTS user_ops (
        user_op_hash TEXT PRIMARY KEY,
        entry_point TEXT NOT NULL,
        sender TEXT NOT NULL,
        nonce TEXT NOT NULL,
        rpc_user_op_json TEXT,
        received_at INTEGER NOT NULL,
        last_updated INTEGER NOT NULL,
        status TEXT NOT NULL,
        tx_hash TEXT,
        actual_gas_cost TEXT,
        actual_gas_used TEXT,
        success INTEGER,
        revert_reason TEXT,
        request_id TEXT,
        remote_address TEXT
      );
      CREATE INDEX IF NOT EXISTS idx_user_ops_status_updated ON user_ops(status, last_updated);
      CREATE INDEX IF NOT EXISTS idx_user_ops_tx_hash ON user_ops(tx_hash);

      CREATE TABLE IF NOT EXISTS logs (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        ts INTEGER NOT NULL,
        level TEXT NOT NULL,
        message TEXT NOT NULL,
        meta_json TEXT
      );
      CREATE INDEX IF NOT EXISTS idx_logs_ts ON logs(ts);
    `)

    // Backwards-compatible migration for existing DBs.
    try {
      this.db.exec(`ALTER TABLE user_ops ADD COLUMN rpc_user_op_json TEXT`)
    } catch {
      // ignore
    }
  }

  private pruneLogs() {
    const count = this.db.prepare(`SELECT COUNT(1) AS n FROM logs`).get() as any
    const n = Number(count?.n ?? 0)
    if (n <= this.maxLogs) return
    const toDelete = n - this.maxLogs
    this.db.prepare(`DELETE FROM logs WHERE id IN (SELECT id FROM logs ORDER BY id ASC LIMIT ?)`).run(toDelete)
  }

  private pruneIncludedReceipts() {
    if (this.receiptLimit <= 0) return
    const count = this.db
      .prepare(`SELECT COUNT(1) AS n FROM user_ops WHERE status = 'included'`)
      .get() as any
    const n = Number(count?.n ?? 0)
    if (n <= this.receiptLimit) return
    const toDelete = n - this.receiptLimit
    this.db
      .prepare(
        `DELETE FROM user_ops WHERE user_op_hash IN (
          SELECT user_op_hash FROM user_ops WHERE status = 'included' ORDER BY received_at ASC LIMIT ?
        )`,
      )
      .run(toDelete)
  }

  private rowToUserOp(row: any): UserOpRecord {
    return {
      userOpHash: row.user_op_hash,
      entryPoint: row.entry_point,
      sender: row.sender,
      nonce: row.nonce,
      rpcUserOp: row.rpc_user_op_json ? safeJsonParse(row.rpc_user_op_json) : undefined,
      receivedAt: row.received_at,
      lastUpdated: row.last_updated,
      status: row.status,
      txHash: row.tx_hash ?? undefined,
      actualGasCost: row.actual_gas_cost ?? undefined,
      actualGasUsed: row.actual_gas_used ?? undefined,
      success: row.success === null || row.success === undefined ? undefined : Boolean(row.success),
      revertReason: row.revert_reason ?? undefined,
      requestId: row.request_id ?? undefined,
      remoteAddress: row.remote_address ?? undefined,
    }
  }

  private rowToReceipt(row: any): BundlerReceipt {
    return {
      userOpHash: row.user_op_hash,
      entryPoint: row.entry_point,
      sender: row.sender,
      nonce: row.nonce,
      actualGasCost: row.actual_gas_cost ?? undefined,
      actualGasUsed: row.actual_gas_used ?? undefined,
      success: Boolean(row.success),
      revertReason: row.revert_reason ?? undefined,
      receipt: { transactionHash: row.tx_hash },
      receivedAt: row.received_at,
      lastUpdated: row.last_updated,
    }
  }
}

function safeJsonParse(value: string): any {
  try {
    return JSON.parse(value)
  } catch {
    return undefined
  }
}
