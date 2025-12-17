interface Bucket {
  tokens: number
  resetAt: number
}

export class RateLimiter {
  private buckets = new Map<string, Bucket>()
  private readonly maxPerWindow: number
  private readonly windowMs: number

  constructor(maxPerMinute: number, windowMs = 60_000) {
    this.maxPerWindow = maxPerMinute
    this.windowMs = windowMs
  }

  allow(key: string): boolean {
    if (this.maxPerWindow <= 0) return true
    const now = Date.now()
    const bucket = this.buckets.get(key)
    if (!bucket || bucket.resetAt < now) {
      this.buckets.set(key, { tokens: this.maxPerWindow - 1, resetAt: now + this.windowMs })
      return true
    }

    if (bucket.tokens <= 0) {
      return false
    }

    bucket.tokens -= 1
    return true
  }
}
