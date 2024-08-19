import { describe, expect, it } from "bun:test" // eslint-disable-line import/no-unresolved
import { toBigInt, Wallet } from "ethers"
import { account, provider } from "./setup"

describe("Basic Queries", () => {
  it("Simple transfer, balance check", async () => {
    const alice = Wallet.createRandom()
    const amountToSend = toBigInt(5e12) * toBigInt(1e6) // unibi

    const senderBalanceBefore = await provider.getBalance(account)
    const recipientBalanceBefore = await provider.getBalance(alice)
    expect(senderBalanceBefore).toBeGreaterThan(0)
    expect(recipientBalanceBefore).toEqual(BigInt(0))

    // Execute EVM transfer
    const transaction = {
      gasLimit: toBigInt(100e3),
      to: alice,
      value: amountToSend,
    }
    const txResponse = await account.sendTransaction(transaction)
    await txResponse.wait(1, 10e3)
    expect(txResponse).toHaveProperty("blockHash")

    const senderBalanceAfter = await provider.getBalance(account)
    const recipientBalanceAfter = await provider.getBalance(alice)

    // Assert balances with logging
    const tenPow12 = toBigInt(1e12)
    const gasUsed = 50000n // 50k gas for the transaction
    const txCostMicronibi = amountToSend / tenPow12 + gasUsed
    const txCostWei = txCostMicronibi * tenPow12
    const expectedSenderWei = senderBalanceBefore - txCostWei
    console.debug("DEBUG should send via transfer method %o:", {
      senderBalanceBefore,
      amountToSend,
      expectedSenderWei,
      senderBalanceAfter,
    })
    expect(senderBalanceAfter).toEqual(expectedSenderWei)
    expect(recipientBalanceAfter).toEqual(amountToSend)
  }, 20e3)
})
