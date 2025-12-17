import { Counter, Gauge, Histogram, Registry, collectDefaultMetrics } from "prom-client"

export class Metrics {
  readonly registry: Registry
  readonly rpcRequests: Counter
  readonly rpcFailures: Counter
  readonly userOpSuccess: Counter
  readonly userOpFailed: Counter
  readonly prefundAttempts: Counter
  readonly prefundFailures: Counter
  readonly queueDepth: Gauge
  readonly submissionDuration: Histogram

  constructor() {
    this.registry = new Registry()
    collectDefaultMetrics({ register: this.registry })

    this.rpcRequests = new Counter({
      name: "bundler_rpc_requests_total",
      help: "Total JSON-RPC requests received",
      registers: [this.registry],
      labelNames: ["method"],
    })

    this.rpcFailures = new Counter({
      name: "bundler_rpc_failures_total",
      help: "Total JSON-RPC requests that failed validation or auth",
      registers: [this.registry],
      labelNames: ["method"],
    })

    this.userOpSuccess = new Counter({
      name: "bundler_user_ops_success_total",
      help: "User operations successfully included",
      registers: [this.registry],
    })

    this.userOpFailed = new Counter({
      name: "bundler_user_ops_failed_total",
      help: "User operations that failed during submission or execution",
      registers: [this.registry],
      labelNames: ["reason"],
    })

    this.prefundAttempts = new Counter({
      name: "bundler_prefund_attempts_total",
      help: "Prefund attempts performed before handleOps",
      registers: [this.registry],
    })

    this.prefundFailures = new Counter({
      name: "bundler_prefund_failures_total",
      help: "Prefund attempts that failed",
      registers: [this.registry],
    })

    this.queueDepth = new Gauge({
      name: "bundler_queue_depth",
      help: "Current queue depth for pending user operations",
      registers: [this.registry],
    })

    this.submissionDuration = new Histogram({
      name: "bundler_submission_duration_ms",
      help: "Time spent submitting a user operation through handleOps",
      registers: [this.registry],
      buckets: [50, 100, 250, 500, 1000, 2500, 5000, 10000],
    })
  }

  setQueueDepth(value: number) {
    this.queueDepth.set(value)
  }

  render(): Promise<string> {
    return this.registry.metrics()
  }
}
