import { describe, expect, it, jest } from "@jest/globals"
import {
  toBigInt,
  parseEther,
  keccak256,
  AbiCoder,
  hexlify,
  TransactionRequest,
} from "ethers"
import { account, provider } from "./setup"
import {
  INTRINSIC_TX_GAS,
  TENPOW12,
  alice,
  deployContractTestERC20,
  deployContractSendNibi,
  hexify,
  sendTestNibi,
  COMMON_TX_ARGS,
} from "./utils"

describe("Basic Queries", () => {
  jest.setTimeout(15e3)

  it("Simple transfer, balance check", async () => {
    const amountToSend = toBigInt(5e12) * toBigInt(1e6) // unibi
    const senderBalanceBefore = await provider.getBalance(account)
    const recipientBalanceBefore = await provider.getBalance(alice)
    expect(senderBalanceBefore).toBeGreaterThan(0)
    expect(recipientBalanceBefore).toEqual(BigInt(0))

    const tenPow12 = toBigInt(1e12)

    // Execute EVM transfer
    const transaction: TransactionRequest = {
      gasLimit: toBigInt(100e3),
      to: alice,
      value: amountToSend,
      maxFeePerGas: tenPow12,
    }
    const txResponse = await account.sendTransaction(transaction)
    await txResponse.wait(1, 10e3)
    expect(txResponse).toHaveProperty("blockHash")

    const senderBalanceAfter = await provider.getBalance(account)
    const recipientBalanceAfter = await provider.getBalance(alice)

    // Assert balances with logging
    const gasUsed = 50000n // 50k gas for the transaction
    const txCostMicronibi = amountToSend / tenPow12 + gasUsed
    const txCostWei = txCostMicronibi * tenPow12
    const expectedSenderWei = senderBalanceBefore - txCostWei
    console.debug("DEBUG should send via transfer method %o:", {
      senderBalanceBefore,
      amountToSend,
      expectedSenderWei,
      senderBalanceAfter,
    })
    expect(senderBalanceAfter).toEqual(expectedSenderWei)
    expect(recipientBalanceAfter).toEqual(amountToSend)
  })

  it("eth_accounts", async () => {
    const accounts = await provider.listAccounts()
    expect(accounts).not.toHaveLength(0)
  })

  it("eth_estimateGas", async () => {
    const tx = {
      from: account.address,
      to: alice,
      value: parseEther("0.01"), // Sending 0.01 Ether
    }
    const estimatedGas = await provider.estimateGas(tx)
    expect(estimatedGas).toBeGreaterThan(BigInt(0))
    expect(estimatedGas).toEqual(INTRINSIC_TX_GAS)
  })

  it("eth_feeHistory", async () => {
    const blockCount = 5 // Number of blocks in the requested history
    const newestBlock = "latest" // Can be a block number or 'latest'
    const rewardPercentiles = [25, 50, 75] // Example percentiles for priority fees

    const feeHistory = await provider.send("eth_feeHistory", [
      blockCount,
      newestBlock,
      rewardPercentiles,
    ])
    expect(feeHistory).toBeDefined()
    expect(feeHistory).toHaveProperty("baseFeePerGas")
    expect(feeHistory).toHaveProperty("gasUsedRatio")
    expect(feeHistory).toHaveProperty("oldestBlock")
    expect(feeHistory).toHaveProperty("reward")
  })

  it("eth_gasPrice", async () => {
    const gasPrice = await provider.send("eth_gasPrice", [])
    expect(gasPrice).toBeDefined()
    expect(gasPrice).toEqual(hexify(1))
  })

  it("eth_getBalance", async () => {
    const balance = await provider.getBalance(account.address)
    expect(balance).toBeGreaterThan(0)
  })

  it("eth_getBlockByNumber, eth_getBlockByHash", async () => {
    const blockNumber = 1
    const blockByNumber = await provider.send("eth_getBlockByNumber", [
      blockNumber,
      false,
    ])
    expect(blockByNumber).toBeDefined()
    expect(blockByNumber).toHaveProperty("hash")

    const blockByHash = await provider.send("eth_getBlockByHash", [
      blockByNumber.hash,
      false,
    ])
    expect(blockByHash).toBeDefined()
    expect(blockByHash.hash).toEqual(blockByNumber.hash)
    expect(blockByHash.number).toEqual(blockByNumber.number)
  })

  it("eth_getBlockTransactionCountByHash", async () => {
    const blockNumber = 1
    const block = await provider.send("eth_getBlockByNumber", [
      blockNumber,
      false,
    ])
    const txCount = await provider.send("eth_getBlockTransactionCountByHash", [
      block.hash,
    ])
    expect(parseInt(txCount)).toBeGreaterThanOrEqual(0)
  })

  it("eth_getBlockTransactionCountByNumber", async () => {
    const blockNumber = 1
    const txCount = await provider.send(
      "eth_getBlockTransactionCountByNumber",
      [blockNumber],
    )
    expect(parseInt(txCount)).toBeGreaterThanOrEqual(0)
  })

  it("eth_getCode", async () => {
    const contract = await deployContractSendNibi()
    const contractAddr = await contract.getAddress()
    const code = await provider.send("eth_getCode", [contractAddr, "latest"])
    expect(code).toBeDefined()
  })

  it("eth_getFilterChanges", async () => {
    // Deploy ERC-20 contract
    const contract = await deployContractTestERC20()
    const contractAddr = await contract.getAddress()
    const filter = {
      fromBlock: "latest",
      address: contractAddr,
    }
    // Create the filter for a contract
    const filterId = await provider.send("eth_newFilter", [filter])
    expect(filterId).toBeDefined()

    // Execute some contract TX
    const tx = await contract.transfer(
      alice,
      parseEther("0.01"),
      COMMON_TX_ARGS,
    )
    await tx.wait(1, 5e3)
    await new Promise((resolve) => setTimeout(resolve, 3000))

    // Assert logs
    const changes = await provider.send("eth_getFilterChanges", [filterId])
    expect(changes.length).toBeGreaterThan(0)
    expect(changes[0]).toHaveProperty("address")
    expect(changes[0]).toHaveProperty("data")
    expect(changes[0]).toHaveProperty("topics")

    const success = await provider.send("eth_uninstallFilter", [filterId])
    expect(success).toBeTruthy()
  })

  // Skipping as the method is not implemented
  it.skip("eth_getFilterLogs", async () => {
    // Deploy ERC-20 contract
    const contract = await deployContractTestERC20()
    const contractAddr = await contract.getAddress()
    const filter = {
      fromBlock: "latest",
      address: contractAddr,
    }
    // Execute some contract TX
    const tx = await contract.transfer(alice, parseEther("0.01"))
    await tx.wait(1, 5e3)

    // Create the filter for a contract
    const filterId = await provider.send("eth_newFilter", [filter])
    expect(filterId).toBeDefined()

    // Assert logs
    const changes = await provider.send("eth_getFilterLogs", [filterId])
    expect(changes.length).toBeGreaterThan(0)
    expect(changes[0]).toHaveProperty("address")
    expect(changes[0]).toHaveProperty("data")
    expect(changes[0]).toHaveProperty("topics")
  })

  // Skipping as the method is not implemented
  it.skip("eth_getLogs", async () => {
    // Deploy ERC-20 contract
    const contract = await deployContractTestERC20()
    const contractAddr = await contract.getAddress()
    const filter = {
      fromBlock: "latest",
      address: contractAddr,
    }
    // Execute some contract TX
    const tx = await contract.transfer(alice, parseEther("0.01"))

    // Assert logs
    const changes = await provider.send("eth_getLogs", [filter])
    expect(changes.length).toBeGreaterThan(0)
    expect(changes[0]).toHaveProperty("address")
    expect(changes[0]).toHaveProperty("data")
    expect(changes[0]).toHaveProperty("topics")
  })

  it("eth_getProof", async () => {
    // Deploy ERC-20 contract
    const contract = await deployContractTestERC20()
    const contractAddr = await contract.getAddress()

    const slot = 1 // Assuming balanceOf is at slot 1
    const storageKey = keccak256(
      AbiCoder.defaultAbiCoder().encode(
        ["address", "uint256"],
        [account.address, slot],
      ),
    )
    const proof = await provider.send("eth_getProof", [
      contractAddr,
      [storageKey],
      "latest",
    ])
    // Assert proof structure
    expect(proof).toHaveProperty("address")
    expect(proof).toHaveProperty("balance")
    expect(proof).toHaveProperty("codeHash")
    expect(proof).toHaveProperty("nonce")
    expect(proof).toHaveProperty("storageProof")

    if (proof.storageProof.length > 0) {
      expect(proof.storageProof[0]).toHaveProperty("key", storageKey)
      expect(proof.storageProof[0]).toHaveProperty("value")
      expect(proof.storageProof[0]).toHaveProperty("proof")
    }
  })

  // Skipping as the method is not implemented
  it.skip("eth_getLogs", async () => {
    // Deploy ERC-20 contract
    const contract = await deployContractTestERC20()
    const contractAddr = await contract.getAddress()
    const filter = {
      fromBlock: "latest",
      address: contractAddr,
    }
    // Execute some contract TX
    const tx = await contract.transfer(alice, parseEther("0.01"))
    await tx.wait(1, 5e3)

    // Assert logs
    const logs = await provider.send("eth_getLogs", [filter])
    expect(logs.length).toBeGreaterThan(0)
    expect(logs[0]).toHaveProperty("address")
    expect(logs[0]).toHaveProperty("data")
    expect(logs[0]).toHaveProperty("topics")
  })

  it("eth_getProof", async () => {
    const contract = await deployContractTestERC20()
    const contractAddr = await contract.getAddress()

    const slot = 1 // Assuming balanceOf is at slot 1
    const storageKey = keccak256(
      AbiCoder.defaultAbiCoder().encode(
        ["address", "uint256"],
        [account.address, slot],
      ),
    )
    const proof = await provider.send("eth_getProof", [
      contractAddr,
      [storageKey],
      "latest",
    ])
    // Assert proof structure
    expect(proof).toHaveProperty("address")
    expect(proof).toHaveProperty("balance")
    expect(proof).toHaveProperty("codeHash")
    expect(proof).toHaveProperty("nonce")
    expect(proof).toHaveProperty("storageProof")

    if (proof.storageProof.length > 0) {
      expect(proof.storageProof[0]).toHaveProperty("key", storageKey)
      expect(proof.storageProof[0]).toHaveProperty("value")
      expect(proof.storageProof[0]).toHaveProperty("proof")
    }
  })

  it("eth_getStorageAt", async () => {
    const contract = await deployContractTestERC20()
    const contractAddr = await contract.getAddress()

    const value = await provider.getStorage(contractAddr, 1)
    expect(value).toBeDefined()
  })

  it("eth_getTransactionByBlockHashAndIndex, eth_getTransactionByBlockNumberAndIndex", async () => {
    // Execute EVM transfer
    const txResponse = await sendTestNibi()
    const block = await txResponse.getBlock()

    const txByBlockHash = await provider.send(
      "eth_getTransactionByBlockHashAndIndex",
      [block.hash, "0x0"],
    )
    expect(txByBlockHash).toBeDefined()
    expect(txByBlockHash).toHaveProperty("from")
    expect(txByBlockHash).toHaveProperty("to")
    expect(txByBlockHash).toHaveProperty("blockHash")
    expect(txByBlockHash).toHaveProperty("blockNumber")
    expect(txByBlockHash).toHaveProperty("value")

    const txByBlockNumber = await provider.send(
      "eth_getTransactionByBlockNumberAndIndex",
      [block.number, "0x0"],
    )

    expect(txByBlockNumber).toBeDefined()
    expect(txByBlockNumber["from"]).toEqual(txByBlockHash["from"])
    expect(txByBlockNumber["to"]).toEqual(txByBlockHash["to"])
    expect(txByBlockNumber["value"]).toEqual(txByBlockHash["value"])
  })

  it("eth_getTransactionByHash", async () => {
    const txResponse = await sendTestNibi()
    const txByHash = await provider.getTransaction(txResponse.hash)
    expect(txByHash).toBeDefined()
    expect(txByHash.hash).toEqual(txResponse.hash)
  })

  it("eth_getTransactionCount", async () => {
    const txCount = await provider.getTransactionCount(account.address)
    expect(txCount).toBeGreaterThanOrEqual(0)
  })

  it("eth_getTransactionReceipt", async () => {
    const txResponse = await sendTestNibi()
    const txReceipt = await provider.getTransactionReceipt(txResponse.hash)
    expect(txReceipt).toBeDefined()
    expect(txReceipt.hash).toEqual(txResponse.hash)
  })

  it("eth_getUncleCountByBlockHash", async () => {
    const latestBlock = await provider.getBlockNumber()
    const block = await provider.getBlock(latestBlock)
    const uncleCount = await provider.send("eth_getUncleCountByBlockHash", [
      block.hash,
    ])
    expect(parseInt(uncleCount)).toBeGreaterThanOrEqual(0)
  })

  it("eth_getUncleCountByBlockNumber", async () => {
    const latestBlock = await provider.getBlockNumber()
    const uncleCount = await provider.send("eth_getUncleCountByBlockNumber", [
      latestBlock,
    ])
    expect(parseInt(uncleCount)).toBeGreaterThanOrEqual(0)
  })

  it("eth_maxPriorityFeePerGas", async () => {
    const maxPriorityGas = await provider.send("eth_maxPriorityFeePerGas", [])
    expect(parseInt(maxPriorityGas)).toBeGreaterThanOrEqual(0)
  })

  it("eth_newBlockFilter", async () => {
    const filterId = await provider.send("eth_newBlockFilter", [])
    expect(filterId).toBeDefined()
  })

  it("eth_newPendingTransactionFilter", async () => {
    const filterId = await provider.send("eth_newPendingTransactionFilter", [])
    expect(filterId).toBeDefined()
  })

  it("eth_syncing", async () => {
    const syncing = await provider.send("eth_syncing", [])
    expect(syncing).toBeFalsy()
  })
})
