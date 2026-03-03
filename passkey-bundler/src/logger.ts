import pino from "pino"

import { BundlerLogEntry, LogLevel } from "./types"

export class BundlerLogger {
  private logger: pino.Logger
  private onLog?: (entry: BundlerLogEntry) => void
  private level: LogLevel

  constructor(level: LogLevel, onLog?: (entry: BundlerLogEntry) => void, bindings?: Record<string, unknown>) {
    this.logger = pino({ level })
    this.onLog = onLog
    this.level = level
    if (bindings) {
      this.logger = this.logger.child(bindings)
    }
  }

  child(bindings: Record<string, unknown>): BundlerLogger {
    const childLogger = new BundlerLogger(this.level, this.onLog)
    childLogger.logger = this.logger.child(bindings)
    childLogger.onLog = this.onLog
    childLogger.level = this.level
    return childLogger
  }

  debug(msg: string, meta?: Record<string, unknown>) {
    this.emit("debug", msg, meta)
  }

  info(msg: string, meta?: Record<string, unknown>) {
    this.emit("info", msg, meta)
  }

  warn(msg: string, meta?: Record<string, unknown>) {
    this.emit("warn", msg, meta)
  }

  error(msg: string, meta?: Record<string, unknown>) {
    this.emit("error", msg, meta)
  }

  private emit(level: LogLevel, msg: string, meta?: Record<string, unknown>) {
    const logEntry: BundlerLogEntry = { ts: Date.now(), level, message: msg, meta }
    this.onLog?.(logEntry)
    switch (level) {
      case "debug":
        this.logger.debug(meta ?? {}, msg)
        break
      case "info":
        this.logger.info(meta ?? {}, msg)
        break
      case "warn":
        this.logger.warn(meta ?? {}, msg)
        break
      case "error":
        this.logger.error(meta ?? {}, msg)
        break
    }
  }
}
