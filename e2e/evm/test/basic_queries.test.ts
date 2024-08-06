import { describe, expect, it } from "bun:test"; // eslint-disable-line import/no-unresolved
import { toBigInt, Wallet } from "ethers";
import { account, provider } from "./setup";

describe("Basic Queries", () => {
  it("Simple transfer, balance check", async () => {
    const alice = Wallet.createRandom()
    const amountToSend = toBigInt(1e3) // unibi

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
    await txResponse.wait()
    expect(txResponse).toHaveProperty("blockHash")

    const senderBalanceAfter = await provider.getBalance(account)
    const recipientBalanceAfter = await provider.getBalance(alice)

    const expectedSenderBalance = senderBalanceBefore - amountToSend - 50000n // 50k gas for the transaction
    expect(senderBalanceAfter).toEqual(expectedSenderBalance)
    expect(recipientBalanceAfter).toEqual(amountToSend)
  }, 20e3)
})
