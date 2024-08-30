import { describe, expect, it } from "@jest/globals"
import { toBigInt } from "ethers"
import { deployContractInfiniteLoopGas } from "./utils"

describe("Infinite loop gas contract", () => {
  it("should fail due to out of gas error", async () => {
    const contract = await deployContractInfiniteLoopGas()

    expect(contract.counter()).resolves.toBe(toBigInt(0))

    try {
      const tx = await contract.forever({ gasLimit: 1e6 })
      await tx.wait()
      throw new Error("The transaction should have failed but did not.")
    } catch (error) {
      expect(error.message).toContain("transaction execution reverted")
    }

    expect(contract.counter()).resolves.toBe(toBigInt(0))
  }, 20e3)
})
