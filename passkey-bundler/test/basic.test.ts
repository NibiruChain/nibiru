import assert from "node:assert/strict"
import { test } from "node:test"

import { RateLimiter } from "../src/rateLimiter"
import { requiredPrefund } from "../src/userop"

test("rate limiter enforces window", () => {
  const limiter = new RateLimiter(2, 1000)
  assert.equal(limiter.allow("a"), true)
  assert.equal(limiter.allow("a"), true)
  assert.equal(limiter.allow("a"), false)
})

test("required prefund sums gas limits and maxFeePerGas", () => {
  const prefund = requiredPrefund({
    sender: "0x0000000000000000000000000000000000000001",
    nonce: 0n,
    initCode: "0x",
    callData: "0x",
    callGasLimit: 1000n,
    verificationGasLimit: 2000n,
    preVerificationGas: 3000n,
    maxFeePerGas: 10n,
    maxPriorityFeePerGas: 1n,
    paymasterAndData: "0x",
    signature: "0x",
  })
  assert.equal(prefund, 60000n)
})
