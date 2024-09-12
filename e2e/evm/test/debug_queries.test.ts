import { describe, expect, it, beforeAll } from "@jest/globals"
import { TransactionReceipt, parseEther } from "ethers"
import { provider } from "./setup"
import { alice, deployContractTestERC20, hexify } from "./utils"
import { TestERC20Compiled__factory } from "../types/ethers-contracts"

describe("debug queries", () => {
  let contractAddress: string
  let txHash: string
  let txIndex: number
  let blockNumber: number
  let blockHash: string

  beforeAll(async () => {
    // Deploy ERC-20 contract
    const contract = await deployContractTestERC20()
    contractAddress = await contract.getAddress()

    // Execute some contract TX
    const txResponse = await contract.transfer(alice, parseEther("0.01"))
    await txResponse.wait(1, 5e3)

    const receipt: TransactionReceipt = await provider.getTransactionReceipt(
      txResponse.hash,
    )
    txHash = txResponse.hash
    txIndex = txResponse.index
    blockNumber = receipt.blockNumber
    blockHash = receipt.blockHash
  }, 20e3)

  it("debug_traceBlockByNumber", async () => {
    const traceResult = await provider.send("debug_traceBlockByNumber", [
      blockNumber,
      {
        tracer: "callTracer",
        timeout: "3000s",
        tracerConfig: { onlyTopCall: false },
      },
    ])
    expectTrace(traceResult)
  })

  it("debug_traceBlockByHash", async () => {
    const traceResult = await provider.send("debug_traceBlockByHash", [
      blockHash,
      {
        tracer: "callTracer",
        timeout: "3000s",
        tracerConfig: { onlyTopCall: false },
      },
    ])
    expectTrace(traceResult)
  })

  it("debug_traceTransaction", async () => {
    const traceResult = await provider.send("debug_traceTransaction", [
      txHash,
      {
        tracer: "callTracer",
        timeout: "3000s",
        tracerConfig: { onlyTopCall: false },
      },
    ])
    expectTrace([{ result: traceResult }])
  })

  it("debug_traceCall", async () => {
    const contractInterface = TestERC20Compiled__factory.createInterface()
    const callData = contractInterface.encodeFunctionData("totalSupply")
    const tx = {
      to: contractAddress,
      data: callData,
      gas: hexify(1000_000),
    }
    const traceResult = await provider.send("debug_traceCall", [
      tx,
      "latest",
      {
        tracer: "callTracer",
        timeout: "3000s",
        tracerConfig: { onlyTopCall: false },
      },
    ])
    expectTrace([{ result: traceResult }])
  })

  // TODO: impl in EVM
  it("debug_getBadBlocks", async () => {
    try {
      const traceResult = await provider.send("debug_getBadBlocks", [txHash])
      console.debug("DEBUG %o:", { traceResult })
      expect(traceResult).toBeDefined()
    } catch (err) {
      expect(err.message).toContain(
        "the method debug_getBadBlocks does not exist",
      )
    }
  })

  // TODO: impl in EVM
  it("debug_storageRangeAt", async () => {
    try {
      const traceResult = await provider.send("debug_storageRangeAt", [
        blockNumber,
        txIndex,
        contractAddress,
        "0x0",
        100,
      ])
      console.debug("DEBUG %o:", { traceResult })
      expect(traceResult).toBeDefined()
    } catch (err) {
      expect(err.message).toContain(
        "the method debug_storageRangeAt does not exist",
      )
    }
  })
})

const expectTrace = (traceResult: any[]) => {
  expect(traceResult).toBeDefined()
  expect(traceResult.length).toBeGreaterThan(0)

  const trace = traceResult[0]["result"]
  expect(trace).toHaveProperty("from")
  expect(trace).toHaveProperty("to")
  expect(trace).toHaveProperty("gas")
  expect(trace).toHaveProperty("gasUsed")
  expect(trace).toHaveProperty("input")
  expect(trace).toHaveProperty("output")
  expect(trace).toHaveProperty("type", "CALL")
}
