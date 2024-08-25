import { describe, expect, it } from "@jest/globals"
import { parseUnits, toBigInt, Wallet } from "ethers"
import { account } from "./setup"
import { COMMON_TX_ARGS, deployContractTestERC20 } from "./utils"

describe("ERC-20 contract tests", () => {
  it("should send properly", async () => {
    const contract = await deployContractTestERC20()
    expect(contract.getAddress()).resolves.toBeDefined()

    const ownerInitialBalance = parseUnits("1000000", 18)
    const alice = Wallet.createRandom()

    expect(contract.balanceOf(account)).resolves.toEqual(ownerInitialBalance)
    expect(contract.balanceOf(alice)).resolves.toEqual(toBigInt(0))

    // send to alice
    const amountToSend = parseUnits("1000", 18) // contract tokens
    let tx = await contract.transfer(alice, amountToSend, COMMON_TX_ARGS)
    await tx.wait()

    expect(contract.balanceOf(account)).resolves.toEqual(
      ownerInitialBalance - amountToSend,
    )
    expect(contract.balanceOf(alice)).resolves.toEqual(amountToSend)
  }, 20e3)
})
