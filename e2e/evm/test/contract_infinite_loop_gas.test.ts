import { describe, it, expect, beforeAll } from "bun:test" // eslint-disable-line import/no-unresolved
import { deployContract } from "./setup"
import { InfiniteLoopGasCompiled } from "../types/ethers-contracts"

describe("Infinite loop gas contract", () => {
  let contract: InfiniteLoopGasCompiled

  beforeAll(async () => {
    contract = (await deployContract(
      "InfiniteLoopGasCompiled.json",
    )) as InfiniteLoopGasCompiled
  })

  it("should fail due to out of gas error", async () => {
    const initialCounter = await contract.counter()
    expect(initialCounter).toBe(BigInt(0))

    try {
      const tx = await contract.forever({ gasLimit: 1000000 })
      await tx.wait()
      throw "The transaction should have failed but did not."
    } catch (error) {
      expect(error.message).toContain("transaction execution reverted")
    }
    const finalCounter = await contract.counter()
    expect(finalCounter).toEqual(initialCounter)
  }, 20000)
})
