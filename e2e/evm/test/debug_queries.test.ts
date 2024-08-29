import { describe, expect, it, beforeAll } from "@jest/globals"
import { parseEther } from "ethers"
import { provider } from "./setup"
import { alice, deployERC20 } from "./utils"

describe("debug queries", () => {
  let contractAddress
  let txHash
  let txIndex
  let blockNumber
  let blockHash

  beforeAll(async () => {
    // Deploy ERC-20 contract
    const contract = await deployERC20()
    contractAddress = await contract.getAddress()

    // Execute some contract TX
    const txResponse = await contract.transfer(alice, parseEther("0.01"))
    await txResponse.wait(1, 5e3)

    const receipt = await provider.getTransactionReceipt(txResponse.hash)
    txHash = txResponse.hash
    txIndex = txResponse.index
    blockNumber = receipt.blockNumber
    blockHash = receipt.blockHash
  }, 20e3)

  it("debug_traceBlockByNumber", async () => {
    const traceResult = await provider.send("debug_traceBlockByNumber", [
      blockNumber,
    ])
    expectTrace(traceResult)
  })

  it("debug_traceBlockByHash", async () => {
    const traceResult = await provider.send("debug_traceBlockByHash", [
      blockHash,
    ])
    expectTrace(traceResult)
  })

  it("debug_traceTransaction", async () => {
    const traceResult = await provider.send("debug_traceTransaction", [txHash])
    expectTrace([{ result: traceResult }])
  })

  // TODO: implement that in EVM
  it.skip("debug_getBadBlocks", async () => {
    const traceResult = await provider.send("debug_getBadBlocks", [txHash])
    expect(traceResult).toBeDefined()
  })

  // TODO: implement that in EVM
  it.skip("debug_storageRangeAt", async () => {
    const traceResult = await provider.send("debug_storageRangeAt", [
      blockNumber,
      txIndex,
      contractAddress,
      "0x0",
      100,
    ])
    expect(traceResult).toBeDefined()
  })
})

const expectTrace = (traceResult: any[]) => {
  expect(traceResult).toBeDefined()
  expect(traceResult.length).toBeGreaterThan(0)

  const trace = traceResult[0]["result"]
  expect(trace).toHaveProperty("failed", false)
  expect(trace).toHaveProperty("gas")
  expect(trace).toHaveProperty("returnValue")
  expect(trace).toHaveProperty("structLogs")
}
