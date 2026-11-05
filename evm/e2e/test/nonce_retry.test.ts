import { describe, expect, it } from "bun:test"

import {
  NonceRetryManager,
  parseNonceMismatch,
} from "../passkey-sdk/src/nonce-retry"

const nestedNibiruNonceError = (actual: number, expected: number) => ({
  code: "UNKNOWN_ERROR",
  shortMessage: "could not coalesce error",
  error: {
    code: -32000,
    message: `error broadcasting tx: invalid nonce; got ${actual}, expected ${expected} or higher: internal`,
  },
  payload: { method: "eth_sendRawTransaction" },
})

describe("nonce retry", () => {
  it("parses the nested Nibiru error returned by ethers", () => {
    expect(parseNonceMismatch(nestedNibiruNonceError(24, 25))).toEqual({
      actual: 24,
      expected: 25,
    })
    expect(
      parseNonceMismatch({
        code: "NONCE_EXPIRED",
        shortMessage: "nonce has already been used",
      }),
    ).toEqual({})
    expect(parseNonceMismatch(new Error("connection reset"))).toBeNull()
  })

  it("retries a rejected transaction with the chain's expected nonce", async () => {
    const manager = new NonceRetryManager(async () => 24)
    const attempted: number[] = []

    const result = await manager.send({}, async (request) => {
      attempted.push(request.nonce)
      if (request.nonce === 24) throw nestedNibiruNonceError(24, 25)
      return "broadcast"
    })

    expect(result).toBe("broadcast")
    expect(attempted).toEqual([24, 25])
  })

  it("does not retry an unknown broadcast error", async () => {
    const manager = new NonceRetryManager(async () => 7)
    const error = new Error("socket closed after broadcast")
    let attempts = 0

    await expect(
      manager.send({}, async () => {
        attempts++
        throw error
      }),
    ).rejects.toBe(error)
    expect(attempts).toBe(1)
  })

  it("stops after the configured retry limit", async () => {
    const manager = new NonceRetryManager(async () => 3, { maxAttempts: 3 })
    const attempted: number[] = []

    await expect(
      manager.send({}, async (request) => {
        attempted.push(request.nonce)
        throw nestedNibiruNonceError(request.nonce, request.nonce + 1)
      }),
    ).rejects.toBeTruthy()
    expect(attempted).toEqual([3, 4, 5])
  })

  it("assigns distinct nonces to concurrent broadcasts", async () => {
    let pendingReads = 0
    const manager = new NonceRetryManager(async () => {
      pendingReads++
      return 10
    })
    const releaseFirst = Promise.withResolvers<void>()
    const attempted: number[] = []

    const first = manager.send({}, async (request) => {
      attempted.push(request.nonce)
      await releaseFirst.promise
      return request.nonce
    })
    const second = manager.send({}, async (request) => {
      attempted.push(request.nonce)
      return request.nonce
    })

    await Promise.resolve()
    releaseFirst.resolve()
    expect(await Promise.all([first, second])).toEqual([10, 11])
    expect(attempted).toEqual([10, 11])
    expect(pendingReads).toBe(1)
  })
})
