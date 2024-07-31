import { describe, expect, it } from "bun:test"; // eslint-disable-line import/no-unresolved
import { ethers, toBigInt } from "ethers";
import { SendNibiCompiled__factory } from "../types/ethers-contracts";
import { account, provider } from "./setup";

describe("Send NIBI via smart contract", async () => {
  const factory = new SendNibiCompiled__factory(account);
  const contract = await factory.deploy();
  await contract.waitForDeployment()
  expect(contract.getAddress()).resolves.toBeDefined()

  it("should send via transfer method", async () => {
    const recipient = ethers.Wallet.createRandom()
    const transferValue = toBigInt(100e6) // NIBI

    const ownerBalanceBefore = await provider.getBalance(account) // NIBI
    const recipientBalanceBefore = await provider.getBalance(recipient) // NIBI
    expect(recipientBalanceBefore).toEqual(BigInt(0))

    const tx = await contract.sendViaTransfer(recipient, {
      value: transferValue,
    })
    await tx.wait(1, 5e3)

    expect(provider.getBalance(account)).resolves.toBe(
      ownerBalanceBefore - transferValue,
    )
    expect(provider.getBalance(recipient)).resolves.toBe(transferValue)
  }, 20e3)

  it("should send via send method", async () => {
    const recipient = ethers.Wallet.createRandom()
    const transferValue = toBigInt(100e6) // NIBI

    const ownerBalanceBefore = await provider.getBalance(account) // NIBI
    const recipientBalanceBefore = await provider.getBalance(recipient) // NIBI
    expect(recipientBalanceBefore).toEqual(BigInt(0))

    const tx = await contract.sendViaSend(recipient, {
      value: transferValue,
    })
    await tx.wait(1, 5e3)

    expect(provider.getBalance(account)).resolves.toBe(
      ownerBalanceBefore - transferValue,
    )
    expect(provider.getBalance(recipient)).resolves.toBe(transferValue)
  }, 20e3)

  it("should send via transfer method", async () => {
    const recipient = ethers.Wallet.createRandom()
    const transferValue = toBigInt(100e6) // NIBI

    const ownerBalanceBefore = await provider.getBalance(account) // NIBI
    const recipientBalanceBefore = await provider.getBalance(recipient) // NIBI
    expect(recipientBalanceBefore).toEqual(BigInt(0))

    const tx = await contract.sendViaCall(recipient, {
      value: transferValue,
    })
    await tx.wait(1, 5e3)

    expect(provider.getBalance(account)).resolves.toBe(
      ownerBalanceBefore - transferValue,
    )
    expect(provider.getBalance(recipient)).resolves.toBe(transferValue)
  }, 20e3)

})
