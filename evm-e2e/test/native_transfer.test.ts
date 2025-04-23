import { describe, expect, it } from "@jest/globals"
import { parseEther, toBigInt } from "ethers"

import { account, provider, TEST_TIMEOUT, TX_WAIT_TIMEOUT } from "./setup"
import { alice } from "./utils"

describe("native transfer", () => {
  it(
    "simple transfer, balance check",
    async () => {
      const amountToSend = toBigInt(5e12) * toBigInt(1e6) // unibi
      const senderBalanceBefore = await provider.getBalance(account)
      const recipientBalanceBefore = await provider.getBalance(alice)
      expect(senderBalanceBefore).toBeGreaterThan(0)

      // Execute EVM transfer
      const transaction = {
        gasLimit: toBigInt(100e3),
        to: alice,
        value: amountToSend,
      }
      const txResponse = await account.sendTransaction(transaction)
      await txResponse.wait(1, TX_WAIT_TIMEOUT)
      expect(txResponse).toHaveProperty("blockHash")

      const senderBalanceAfter = await provider.getBalance(account)
      const recipientBalanceAfter = await provider.getBalance(alice)

      // Assert balances with logging
      const tenPow12 = toBigInt(1e12)
      const gasUsed = transaction.gasLimit
      const txCostMicronibi = amountToSend / tenPow12 + gasUsed
      const txCostWei = txCostMicronibi * tenPow12
      const expectedSenderWei = senderBalanceBefore - txCostWei
      console.debug("DEBUG should send via transfer method %o:", {
        senderBalanceBefore,
        amountToSend,
        expectedSenderWei,
        senderBalanceAfter,
        txResponse,
      })
      expect(recipientBalanceAfter).toEqual(recipientBalanceBefore + amountToSend)
      const delta = senderBalanceAfter - expectedSenderWei
      const deltaFromExpectation = delta >= 0 ? delta : -delta
      expect(deltaFromExpectation).toBeLessThan(parseEther("0.1"))
    },
    TEST_TIMEOUT,
  )
})
