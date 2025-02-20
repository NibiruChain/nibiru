import { describe, it, expect, beforeAll } from "bun:test" // eslint-disable-line import/no-unresolved
import { AddressLike, ethers } from "ethers"
import { account, provider, deployContract } from "./setup"
import { SendNibiCompiled } from "../types/ethers-contracts"
import { TypedContractMethod } from "../types/ethers-contracts/common"

type SendMethod = TypedContractMethod<[_to: AddressLike], [void], "payable">

const doContractSend = async (sendMethod: SendMethod) => {
  const recipientAddress = ethers.Wallet.createRandom().address
  const transferValue = 100n * 10n ** 6n // NIBI

  const ownerBalanceBefore = await provider.getBalance(account.address) // NIBI
  const recipientBalanceBefore = await provider.getBalance(recipientAddress) // NIBI
  expect(recipientBalanceBefore).toEqual(BigInt(0))

  const tx = await sendMethod(recipientAddress, {
    value: transferValue,
  })
  const [blockConfirmations, timeout] = [1, 5_000]
  await tx.wait(blockConfirmations, timeout)

  const ownerBalanceAfter = await provider.getBalance(account.address) // NIBI
  const recipientBalanceAfter = await provider.getBalance(recipientAddress) // NIBI

  expect(ownerBalanceAfter).toBeLessThanOrEqual(
    ownerBalanceBefore - transferValue,
  )
  expect(recipientBalanceAfter).toEqual(transferValue)
}

describe("Send NIBI from smart contract", async () => {
  let contract: SendNibiCompiled
  contract = (await deployContract("SendNibiCompiled.json")) as SendNibiCompiled

  expect(contract).toBeDefined()
  const sendMethods: SendMethod[] = [
    contract.sendViaTransfer,
    contract.sendViaSend,
    contract.sendViaCall,
  ]
  sendMethods.forEach((m) => expect(m).toBeFunction())
  // Contract initialized properly.

  const testCases = sendMethods.map((sendMethod) => ({
    testName: sendMethod.name,
    sendMethod,
  }))
  testCases.forEach(({ testName, sendMethod }) => {
    it(`send nibi via ${testName} method`, async () => {
      await doContractSend(sendMethod)
    }, 20000)
  })
})
