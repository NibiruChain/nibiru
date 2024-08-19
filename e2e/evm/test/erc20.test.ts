import { describe, expect, it } from "bun:test"; // eslint-disable-line import/no-unresolved
import { parseUnits, toBigInt, Wallet } from "ethers";
import { TestERC20Compiled__factory } from "../types/ethers-contracts";
import { account } from "./setup";

describe("ERC-20 contract tests", () => {
  it("should send properly", async () => {
    const factory = new TestERC20Compiled__factory(account);
    const contract = await factory.deploy();
    await contract.waitForDeployment()
    expect(contract.getAddress()).resolves.toBeDefined()

    const ownerInitialBalance = parseUnits("1000000", 18)
    const alice = Wallet.createRandom()

    expect(contract.balanceOf(account)).resolves.toEqual(ownerInitialBalance)
    expect(contract.balanceOf(alice)).resolves.toEqual(toBigInt(0))
    
    // send to alice
    const amountToSend = parseUnits("1000", 18) // contract tokens
    let tx = await contract.transfer(alice, amountToSend)
    await tx.wait()

    expect(contract.balanceOf(account)).resolves.toEqual(ownerInitialBalance - amountToSend)
    expect(contract.balanceOf(alice)).resolves.toEqual(amountToSend)
  }, 20e3)
})
