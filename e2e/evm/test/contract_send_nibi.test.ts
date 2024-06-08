import { describe, it, expect, beforeAll } from "bun:test" // eslint-disable-line import/no-unresolved
import { ethers } from "ethers"
import { account, provider, deployContract } from "./setup"
import { SendNibiCompiled } from "../types/ethers-contracts"

const doContractSend = async (
  sendMethod: string,
  contract: SendNibiCompiled,
) => {
  const recipientAddress = ethers.Wallet.createRandom().address
  const transferValue = 100n * 10n ** 6n // NIBI

  const ownerBalanceBefore = await provider.getBalance(account.address) // NIBI
  const recipientBalanceBefore = await provider.getBalance(recipientAddress) // NIBI
  expect(recipientBalanceBefore).toEqual(BigInt(0))

  const tx = await contract[sendMethod](recipientAddress, {
    value: transferValue,
  })
  await tx.wait()

  const ownerBalanceAfter = await provider.getBalance(account.address) // NIBI
  const recipientBalanceAfter = await provider.getBalance(recipientAddress) // NIBI

  expect(ownerBalanceAfter).toBeLessThanOrEqual(
    ownerBalanceBefore - transferValue,
  )
  expect(recipientBalanceAfter).toEqual(transferValue)
}

describe("Send NIBI from smart contract", () => {
  let contract: SendNibiCompiled
  beforeAll(async () => {
    contract = (await deployContract(
      "SendNibiCompiled.json",
    )) as SendNibiCompiled
  })

  it.each([["sendViaTransfer"], ["sendViaSend"], ["sendViaCall"]])(
    "send nibi via %p method",
    async (sendMethod) => {
      await doContractSend(sendMethod, contract)
    },
    20000,
  )
})
