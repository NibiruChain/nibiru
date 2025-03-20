import { describe, expect, it } from "@jest/globals"
import { toBigInt } from "ethers"

import { TEST_TIMEOUT } from "./setup"
import { deployContractInfiniteLoopGas } from "./utils"

describe("Infinite loop gas contract", () => {
  it(
    "should fail due to out of gas error",
    async () => {
      const contract = await deployContractInfiniteLoopGas()

      await expect(contract.counter()).resolves.toBe(toBigInt(0))

      try {
        const tx = await contract.forever({ gasLimit: 1e6 })
        await tx.wait()
        throw new Error("The transaction should have failed but did not.")
      } catch (error) {
        expect(error.message).toContain("transaction execution reverted")
      }

      await expect(contract.counter()).resolves.toBe(toBigInt(0))
    },
    TEST_TIMEOUT,
  )
})
