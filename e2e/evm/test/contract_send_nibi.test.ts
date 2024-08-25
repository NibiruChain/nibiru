import { describe, expect, it } from "@jest/globals"
import { toBigInt, Wallet } from "ethers"
import { account, provider } from "./setup"
import { COMMON_TX_ARGS, deployContractSendNibi } from "./utils"

describe("Send NIBI via smart contract", () => {
  it("should send via transfer method", async () => {
    const contract = await deployContractSendNibi()
    const recipient = Wallet.createRandom()
    const weiToSend = toBigInt(5e12) * toBigInt(1e6) // 5 micro NIBI

    const ownerBalanceBefore = await provider.getBalance(account) // NIBI
    const recipientBalanceBefore = await provider.getBalance(recipient) // NIBI
    expect(recipientBalanceBefore).toEqual(BigInt(0))

    const tx = await contract.sendViaTransfer(recipient, {
      value: weiToSend,
      ...COMMON_TX_ARGS,
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
    const contract = await deployContractSendNibi()
    const recipient = Wallet.createRandom()
    const weiToSend = toBigInt(100e12) * toBigInt(1e6) // 100 NIBi

    const ownerBalanceBefore = await provider.getBalance(account) // NIBI
    const recipientBalanceBefore = await provider.getBalance(recipient) // NIBI
    expect(recipientBalanceBefore).toEqual(BigInt(0))

    const tx = await contract.sendViaSend(recipient, {
      value: weiToSend,
      ...COMMON_TX_ARGS,
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
    const contract = await deployContractSendNibi()
    const recipient = Wallet.createRandom()
    const weiToSend = toBigInt(100e12) * toBigInt(1e6) // 100 NIBI

    const ownerBalanceBefore = await provider.getBalance(account) // NIBI
    const recipientBalanceBefore = await provider.getBalance(recipient) // NIBI
    expect(recipientBalanceBefore).toEqual(BigInt(0))

    const tx = await contract.sendViaCall(recipient, {
      value: weiToSend,
      ...COMMON_TX_ARGS,
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
