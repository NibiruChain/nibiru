import { SubmissionJob } from "./types"

export interface QueueOpts {
  concurrency: number
  maxSize: number
  onDepthChange?: (depth: number) => void
  processor: (job: QueuedJob) => Promise<void>
}

export interface QueuedJob extends SubmissionJob {
  enqueuedAt: number
}

export class UserOpQueue {
  private readonly opts: QueueOpts
  private queue: QueuedJob[] = []
  private active = 0

  constructor(opts: QueueOpts) {
    this.opts = opts
  }

  enqueue(job: SubmissionJob): QueuedJob {
    if (this.queue.length + this.active >= this.opts.maxSize) {
      throw new Error("Bundler queue is full")
    }
    const queued: QueuedJob = { ...job, enqueuedAt: Date.now() }
    this.queue.push(queued)
    this.drain()
    return queued
  }

  depth(): number {
    return this.queue.length + this.active
  }

  private drain() {
    this.opts.onDepthChange?.(this.depth())
    while (this.active < this.opts.concurrency && this.queue.length > 0) {
      const nextIndex = this.pickNextIndex()
      const [job] = this.queue.splice(nextIndex, 1)
      this.active += 1
      this.opts.onDepthChange?.(this.depth())
      this.opts
        .processor(job)
        .catch(() => {
          // errors are logged in the processor; nothing to do here
        })
        .finally(() => {
          this.active -= 1
          this.drain()
        })
    }
  }

  private pickNextIndex(): number {
    let bestIdx = 0
    for (let i = 1; i < this.queue.length; i += 1) {
      const candidate = this.queue[i]
      const best = this.queue[bestIdx]
      if (candidate.enqueuedAt < best.enqueuedAt) {
        bestIdx = i
      } else if (candidate.enqueuedAt === best.enqueuedAt) {
        // FIFO, then prefer higher priority fee
        const candPriority = BigInt(candidate.userOp.maxPriorityFeePerGas)
        const bestPriority = BigInt(best.userOp.maxPriorityFeePerGas)
        if (candPriority > bestPriority) {
          bestIdx = i
        }
      }
    }
    return bestIdx
  }
}
