import { describe, expect, it } from "bun:test"; // eslint-disable-line import/no-unresolved
import { toBigInt } from "ethers";
import { InfiniteLoopGasCompiled__factory } from "../types/ethers-contracts";
import { account } from "./setup";

describe("Infinite loop gas contract", () => {
  it("should fail due to out of gas error", async () => {
    const factory = new InfiniteLoopGasCompiled__factory(account);
    const contract = await factory.deploy();
    await contract.waitForDeployment()
    
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
