import assert from "node:assert/strict"
import fs from "node:fs"
import os from "node:os"
import path from "node:path"
import { test } from "node:test"

import { RateLimiter } from "../src/rateLimiter"
import { SqliteStore } from "../src/store"
import { requiredPrefund } from "../src/userop"
import { NonceManager } from "../src/submission"
import { calldataGasCost, estimatePreVerificationGas } from "../src/validation"

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

test("nonce manager reuses failed nonce and avoids gaps", async () => {
  const wallet = {
    getNonce: async () => 5,
  } as any
  const logger = {
    info: () => {},
    warn: () => {},
  } as any

  const nm = new NonceManager(wallet, logger)
  await nm.init()

  const n1 = await nm.reserve()
  const n2 = await nm.reserve()
  assert.equal(n1, 5n)
  assert.equal(n2, 6n)

  await nm.onSendFailure(n1, new Error("send failed"))

  const n3 = await nm.reserve()
  const n4 = await nm.reserve()
  assert.equal(n3, 5n)
  assert.equal(n4, 7n)
})

test("nonce manager does not rewind below chain pending nonce", async () => {
  let pending = 10
  const wallet = {
    getNonce: async () => pending,
  } as any
  const logger = {
    info: () => {},
    warn: () => {},
  } as any

  const nm = new NonceManager(wallet, logger)
  await nm.init()

  const n1 = await nm.reserve()
  assert.equal(n1, 10n)

  pending = 12
  await nm.onSendFailure(n1, new Error("send failed"))
  const n2 = await nm.reserve()
  assert.equal(n2, 12n)
})

test("calldata gas cost counts 0/nonzero bytes", () => {
  assert.equal(calldataGasCost("0x"), 0n)
  assert.equal(calldataGasCost("0x00"), 4n)
  assert.equal(calldataGasCost("0x01"), 16n)
  assert.equal(calldataGasCost("0x0001"), 20n)
})

test("preVerificationGas estimate includes calldata and overhead", () => {
  const pre = estimatePreVerificationGas({
    beneficiary: "0x0000000000000000000000000000000000000000",
    userOp: {
      sender: "0x0000000000000000000000000000000000000001",
      nonce: 0n,
      initCode: "0x",
      callData: "0x",
      callGasLimit: 0n,
      verificationGasLimit: 0n,
      preVerificationGas: 0n,
      maxFeePerGas: 0n,
      maxPriorityFeePerGas: 0n,
      paymasterAndData: "0x",
      signature: "0x",
    },
  })
  assert.ok(pre > 30_000n)
})

test("sqlite store persists userOp records and receipts", async () => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "passkey-bundler-"))
  const dbPath = path.join(dir, "bundler.sqlite")
  const store = new SqliteStore({ dbPath, receiptLimit: 10, maxLogs: 10 })

  const userOpHash = "0x" + "11".repeat(32)
  await store.upsertUserOp({
    userOpHash,
    entryPoint: "0x0000000000000000000000000000000000000002",
    sender: "0x0000000000000000000000000000000000000001",
    nonce: "0x0",
    receivedAt: 1,
    lastUpdated: 2,
    status: "queued",
  })
  assert.equal((await store.getUserOp(userOpHash))?.status, "queued")
  assert.equal(await store.getReceipt(userOpHash), null)

  await store.saveReceipt({
    userOpHash,
    entryPoint: "0x0000000000000000000000000000000000000002",
    sender: "0x0000000000000000000000000000000000000001",
    nonce: "0x0",
    success: false,
    revertReason: "Execution reverted",
    receipt: { transactionHash: "0x" + "22".repeat(32) },
    receivedAt: 1,
    lastUpdated: 3,
  })

  const receipt = await store.getReceipt(userOpHash)
  assert.ok(receipt)
  assert.equal(receipt?.receipt.transactionHash, "0x" + "22".repeat(32))
  assert.equal(receipt?.success, false)

  const included = await store.listUserOpsByStatus(["included"], 10)
  assert.equal(included.length, 1)

  fs.rmSync(dir, { recursive: true, force: true })
})
