import { ethers } from "ethers"
import { describe, it, expect } from "bun:test" // eslint-disable-line import/no-unresolved
import { account, provider } from "./setup"

describe("Basic Queries", () => {
  it("Simple transfer, balance check", async () => {
    const randomAddress = ethers.Wallet.createRandom().address
    const amountToSend = BigInt(1000) // unibi
    const gasLimit = BigInt(100_000) // unibi

    const senderBalanceBefore = await provider.getBalance(account.address)
    const recipientBalanceBefore = await provider.getBalance(randomAddress)

    expect(senderBalanceBefore).toBeGreaterThan(0)
    expect(recipientBalanceBefore).toEqual(BigInt(0))

    // Execute EVM transfer
    const transaction = {
      gasLimit: gasLimit,
      to: randomAddress,
      value: amountToSend,
    }
    const txResponse = await account.sendTransaction(transaction)
    await txResponse.wait()
    expect(txResponse).toHaveProperty("blockHash")

    const senderBalanceAfter = await provider.getBalance(account.address)
    const recipientBalanceAfter = await provider.getBalance(randomAddress)

    // TODO: https://github.com/NibiruChain/nibiru/issues/1902
    // gas is not deducted regardless the gas limit, check this
    const expectedSenderBalance = senderBalanceBefore - amountToSend
    expect(senderBalanceAfter).toBeLessThanOrEqual(expectedSenderBalance)
    expect(recipientBalanceAfter).toEqual(amountToSend)
  }, 20_000)
})
