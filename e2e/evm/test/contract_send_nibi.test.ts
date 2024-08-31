/**
 * @file sendNibi.test.ts
 *
 * This test suite is designed to validate the functionality of the for sending
 * NIBI via various mechanisms like transfer, send, and call. The tests ensure
 * that the correct amount of NIBI is transferred and that balances are updated
 * accordingly.
 *
 * The methods tested are from the smart contract,
 * "e2e/evm/contracts/SendReceiveNibi.sol".
 */
import { describe, expect, it } from "@jest/globals"
import { toBigInt, Wallet } from "ethers"
import { account, provider } from "./setup"
import { deployContractSendNibi } from "./utils"

async function testSendNibi(
  method: "sendViaTransfer" | "sendViaSend" | "sendViaCall",
  weiToSend: bigint,
) {
  const contract = await deployContractSendNibi()
  const recipient = Wallet.createRandom()

  const ownerBalanceBefore = await provider.getBalance(account)
  const recipientBalanceBefore = await provider.getBalance(recipient)
  expect(recipientBalanceBefore).toEqual(BigInt(0))

  const tx = await contract[method](recipient, { value: weiToSend })
  const receipt = await tx.wait(1, 5e3)

  const tenPow12 = toBigInt(1e12)
  const txCostMicronibi = weiToSend / tenPow12 + receipt.gasUsed
  const txCostWei = txCostMicronibi * tenPow12
  const expectedOwnerWei = ownerBalanceBefore - txCostWei

  console.debug(`DEBUG method ${method} %o:`, {
    ownerBalanceBefore,
    weiToSend,
    gasUsed: receipt.gasUsed,
    gasPrice: `${receipt.gasPrice.toString()} micronibi`,
    expectedOwnerWei,
  })

  await expect(provider.getBalance(account)).resolves.toBe(expectedOwnerWei)
  await expect(provider.getBalance(recipient)).resolves.toBe(weiToSend)
}

describe("Send NIBI via smart contract", () => {
  const TIMEOUT_MS = 20e3
  it(
    "method sendViaTransfer",
    async () => {
      const weiToSend: bigint = toBigInt(5e12) * toBigInt(1e6)
      await testSendNibi("sendViaTransfer", weiToSend)
    },
    TIMEOUT_MS,
  )

  it(
    "method sendViaSend",
    async () => {
      const weiToSend: bigint = toBigInt(100e12) * toBigInt(1e6)
      await testSendNibi("sendViaSend", weiToSend)
    },
    TIMEOUT_MS,
  )

  it(
    "method sendViaCall",
    async () => {
      const weiToSend: bigint = toBigInt(100e12) * toBigInt(1e6)
      await testSendNibi("sendViaCall", weiToSend)
    },
    TIMEOUT_MS,
  )
})
