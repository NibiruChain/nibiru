import { describe, it, expect } from "bun:test" // eslint-disable-line import/no-unresolved
import { ethers } from "ethers"
import { account, deployContract } from "./setup"
import { FunTokenCompiled } from "../types/ethers-contracts"

describe("ERC-20 contract tests", () => {
  it("send, balanceOf", async () => {
    const contract = (await deployContract(
      "FunTokenCompiled.json",
    )) as FunTokenCompiled
    const contractAddress = await contract.getAddress()
    expect(contractAddress).toBeDefined()

    // Execute contract: ERC20 transfer
    const shrimpAddress = ethers.Wallet.createRandom().address
    let ownerInitialBalance = ethers.parseUnits("1000000", 18)

    const amountToSend = ethers.parseUnits("1000", 18) // contract tokens

    let ownerBalance = await contract.balanceOf(account.address)
    let shrimpBalance = await contract.balanceOf(shrimpAddress)

    expect(ownerBalance).toEqual(ownerInitialBalance)
    expect(shrimpBalance).toEqual(ethers.toBigInt(0))

    let tx = await contract.transfer(shrimpAddress, amountToSend)
    await tx.wait()

    ownerBalance = await contract.balanceOf(account.address)
    shrimpBalance = await contract.balanceOf(shrimpAddress)

    expect(ownerBalance).toEqual(ownerInitialBalance - amountToSend)
    expect(shrimpBalance).toEqual(amountToSend)
  }, 20000)
})
