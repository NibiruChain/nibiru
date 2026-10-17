import * as fs from "fs"
import { describe, expect, it } from "bun:test"
import { Contract, Wallet } from "ethers"

import { account, provider, TEST_TIMEOUT } from "./testdeps"
import { txWait } from "./utils"

// Load the full ABI from the artifact file
const contractArtifact = JSON.parse(
  fs.readFileSync("./artifacts/contracts/IFunToken.sol/IFunToken.json", "utf8"),
)

const FUNTOKEN_PRECOMPILE = "0x0000000000000000000000000000000000000800"
const SENDER_SLOT_LIMIT_ERROR = "EVM mempool sender slot limit reached"
const SENDER_SLOT_RETRY_ATTEMPTS = 4
const BLOCK_POLL_INTERVAL_MS = 250

const waitForNextBlock = async (blockNumber: number): Promise<number> => {
  const startedAt = Date.now()
  let latestBlock = blockNumber

  while (Date.now() - startedAt < TEST_TIMEOUT) {
    await new Promise((resolve) => setTimeout(resolve, BLOCK_POLL_INTERVAL_MS))
    latestBlock = await provider.getBlockNumber()
    if (latestBlock > blockNumber) {
      return latestBlock
    }
  }

  throw new Error(
    `timed out waiting for next block after ${blockNumber}; latestBlock=${latestBlock}`,
  )
}

const sendWithSenderSlotRetry = async <T>(
  send: () => Promise<T>,
  label: string,
): Promise<T> => {
  for (let attempt = 1; attempt <= SENDER_SLOT_RETRY_ATTEMPTS; attempt++) {
    try {
      return await send()
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error)
      if (
        !message.includes(SENDER_SLOT_LIMIT_ERROR) ||
        attempt === SENDER_SLOT_RETRY_ATTEMPTS
      ) {
        throw error
      }

      const blockNumber = await provider.getBlockNumber()
      console.log(
        `[senderSlotRetry:${label}] sender slots full; waiting after block ${blockNumber} before attempt ${attempt + 1}/${SENDER_SLOT_RETRY_ATTEMPTS}`,
      )
      await waitForNextBlock(blockNumber)
    }
  }

  throw new Error(`unreachable sender slot retry state for ${label}`)
}

describe("StateDB corruption test", () => {
  it(
    "concurrent simulations don't corrupt StateDB during transactions",
    async () => {
      const contract = new Contract(
        FUNTOKEN_PRECOMPILE,
        contractArtifact.abi,
        account,
      )
      const recipient = Wallet.createRandom()

      // Get initial balances
      const senderBalanceBefore = await provider.getBalance(account.address)
      const recipientBalanceBefore = await provider.getBalance(
        recipient.address,
      )

      const SIMULATION_COUNT = 100
      const TX_COUNT = 10
      const TX_AMOUNT = 1 // 1 unibi
      const SIMULATION_AMOUNT = 1000 // 1000 unibi

      // Run aggressive simulations
      const runSimulations = async (): Promise<void> => {
        const promises = []

        for (let i = 0; i < SIMULATION_COUNT; i++) {
          if (i % 2 === 0) {
            promises.push(
              contract.bankMsgSend
                .estimateGas(recipient.address, "unibi", SIMULATION_AMOUNT)
                .catch(() => {}),
            )
          } else {
            promises.push(
              contract.bankMsgSend
                .staticCall(recipient.address, "unibi", SIMULATION_AMOUNT)
                .catch(() => {}),
            )
          }
        }

        await Promise.all(promises)
      }

      // Start continuous simulations
      let simulationRunning = true
      const simulationPromise = (async () => {
        while (simulationRunning) {
          await runSimulations()
          await new Promise((resolve) => setTimeout(resolve, 1))
        }
      })()

      // Wait for simulations to start
      await new Promise((resolve) => setTimeout(resolve, 50))

      // Send real transactions
      const currentNonce = await provider.getTransactionCount(
        account.address,
        "pending",
      )
      const txPromises = []

      for (let i = 0; i < TX_COUNT; i++) {
        const tx = sendWithSenderSlotRetry(
          () =>
            contract.bankMsgSend(recipient.address, "unibi", TX_AMOUNT, {
              gasLimit: 1000000,
              nonce: currentNonce + i,
            }),
          `bankMsgSend ${i}`,
        )

        txPromises.push(tx)
      }

      const transactions = await Promise.all(txPromises)
      const receipts = await Promise.all(
        transactions.map((tx, idx) =>
          txWait(tx, { label: `precompile_statedb bankMsgSend ${idx}` }),
        ),
      )

      // Stop simulations
      simulationRunning = false
      await simulationPromise

      // Get final balances
      const senderBalanceAfter = await provider.getBalance(account.address)
      const recipientBalanceAfter = await provider.getBalance(recipient.address)

      // Assert balances - expecting 10 unibi = 10 * 10^12 wei
      const totalSentWei = BigInt(TX_AMOUNT * TX_COUNT) * BigInt(10 ** 12) // 10 unibi in wei
      expect(recipientBalanceAfter - recipientBalanceBefore).toEqual(
        totalSentWei,
      )

      // Sender balance should be reduced by 10 * 10^12 wei + gas fees
      expect(senderBalanceBefore - senderBalanceAfter).toBeGreaterThan(
        totalSentWei,
      )
    },
    TEST_TIMEOUT * 3,
  )
})
