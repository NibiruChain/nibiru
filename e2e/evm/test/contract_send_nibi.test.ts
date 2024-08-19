import { describe, expect, it } from "bun:test" // eslint-disable-line import/no-unresolved
import { toBigInt, Wallet } from "ethers"
import { SendNibiCompiled__factory } from "../types/ethers-contracts"
import { account, provider } from "./setup"

describe("Send NIBI via smart contract", async () => {
  const factory = new SendNibiCompiled__factory(account)
  const contract = await factory.deploy()
  await contract.waitForDeployment()
  expect(contract.getAddress()).resolves.toBeDefined()

  it("should send via transfer method", async () => {
    const recipient = Wallet.createRandom()
    const weiToSend = toBigInt(5e12) * toBigInt(1e6) // 5 micro NIBI

    const ownerBalanceBefore = await provider.getBalance(account) // NIBI
    const recipientBalanceBefore = await provider.getBalance(recipient) // NIBI
    expect(recipientBalanceBefore).toEqual(BigInt(0))

    const tx = await contract.sendViaTransfer(recipient, {
      value: weiToSend,
    })
    const receipt = await tx.wait(1, 5e3)

    // Assert balances with logging
    const tenPow12 = toBigInt(1e12)
    const txCostMicronibi = weiToSend / tenPow12 + receipt.gasUsed
    const txCostWei = txCostMicronibi * tenPow12
    const expectedOwnerWei = ownerBalanceBefore - txCostWei
    console.debug("DEBUG should send via transfer method %o:", {
      ownerBalanceBefore,
      weiToSend,
      gasUsed: receipt.gasUsed,
      gasPrice: `${receipt.gasPrice.toString()} micronibi`,
      expectedOwnerWei,
    })
    expect(provider.getBalance(account)).resolves.toBe(expectedOwnerWei)
    expect(provider.getBalance(recipient)).resolves.toBe(weiToSend)
  }, 20e3)

  it("should send via send method", async () => {
    const recipient = Wallet.createRandom()
    const weiToSend = toBigInt(100e12) * toBigInt(1e6) // 100 NIBi

    const ownerBalanceBefore = await provider.getBalance(account) // NIBI
    const recipientBalanceBefore = await provider.getBalance(recipient) // NIBI
    expect(recipientBalanceBefore).toEqual(BigInt(0))

    const tx = await contract.sendViaSend(recipient, {
      value: weiToSend,
    })
    const receipt = await tx.wait(1, 5e3)

    // Assert balances with logging
    const tenPow12 = toBigInt(1e12)
    const txCostMicronibi = weiToSend / tenPow12 + receipt.gasUsed
    const txCostWei = txCostMicronibi * tenPow12
    const expectedOwnerWei = ownerBalanceBefore - txCostWei
    console.debug("DEBUG send via send method %o:", {
      ownerBalanceBefore,
      weiToSend,
      gasUsed: receipt.gasUsed,
      gasPrice: `${receipt.gasPrice.toString()} micronibi`,
      expectedOwnerWei,
    })
    expect(provider.getBalance(account)).resolves.toBe(expectedOwnerWei)
    expect(provider.getBalance(recipient)).resolves.toBe(weiToSend)
  }, 20e3)

  it("should send via transfer method", async () => {
    const recipient = Wallet.createRandom()
    const weiToSend = toBigInt(100e12) * toBigInt(1e6) // 100 NIBI

    const ownerBalanceBefore = await provider.getBalance(account) // NIBI
    const recipientBalanceBefore = await provider.getBalance(recipient) // NIBI
    expect(recipientBalanceBefore).toEqual(BigInt(0))

    const tx = await contract.sendViaCall(recipient, {
      value: weiToSend,
    })
    const receipt = await tx.wait(1, 5e3)

    // Assert balances with logging
    const tenPow12 = toBigInt(1e12)
    const txCostMicronibi = weiToSend / tenPow12 + receipt.gasUsed
    const txCostWei = txCostMicronibi * tenPow12
    const expectedOwnerWei = ownerBalanceBefore - txCostWei
    console.debug("DEBUG should send via transfer method %o:", {
      ownerBalanceBefore,
      weiToSend,
      gasUsed: receipt.gasUsed,
      gasPrice: `${receipt.gasPrice.toString()} micronibi`,
      expectedOwnerWei,
    })
    expect(provider.getBalance(account)).resolves.toBe(expectedOwnerWei)
    expect(provider.getBalance(recipient)).resolves.toBe(weiToSend)
  }, 20e3)
})
