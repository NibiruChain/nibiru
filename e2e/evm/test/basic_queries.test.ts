import { describe, expect, it } from "@jest/globals"
import { toBigInt, Wallet, parseEther } from "ethers"
import { account, provider } from "./setup"
import {
  SendNibiCompiled__factory,
  TestERC20Compiled__factory,
} from "../types/ethers-contracts"
import { deployERC20 } from "./utils"

describe("Basic Queries", () => {
  const alice = Wallet.createRandom()

  it("Simple transfer, balance check", async () => {
    const amountToSend = toBigInt(5e12) * toBigInt(1e6) // unibi
    const senderBalanceBefore = await provider.getBalance(account)
    const recipientBalanceBefore = await provider.getBalance(alice)
    expect(senderBalanceBefore).toBeGreaterThan(0)
    expect(recipientBalanceBefore).toEqual(BigInt(0))

    // Execute EVM transfer
    const transaction = {
      gasLimit: toBigInt(100e3),
      to: alice,
      value: amountToSend,
    }
    const txResponse = await account.sendTransaction(transaction)
    await txResponse.wait(1, 10e3)
    expect(txResponse).toHaveProperty("blockHash")

    const senderBalanceAfter = await provider.getBalance(account)
    const recipientBalanceAfter = await provider.getBalance(alice)

    // Assert balances with logging
    const tenPow12 = toBigInt(1e12)
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
  }, 20e3)

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
    const factory = new SendNibiCompiled__factory(account)
    const contract = await factory.deploy()
    await contract.waitForDeployment()
    const contractAddr = await contract.getAddress()
    const code = await provider.send("eth_getCode", [contractAddr, "latest"])
    expect(code).toBeDefined()
  })

  it("eth_getFilterChanges", async () => {
    // Deploy ERC-20 contract
    const contract = await deployERC20()
    const address = await contract.getAddress()
    const filter = {
      fromBlock: "latest",
      address: address,
    }
    // Create the filter for a contract
    const filterId = await provider.send("eth_newFilter", [filter])
    expect(filterId).toBeDefined()

    // Execute some contract TX
    const tx = await contract.transfer(alice, parseEther("0.01"))
    await tx.wait(1, 5e3)

    // Assert logs
    const changes = await provider.send("eth_getFilterChanges", [filterId])
    expect(changes.length).toBeGreaterThan(0)
    expect(changes[0]).toHaveProperty("address")
    expect(changes[0]).toHaveProperty("data")
    expect(changes[0]).toHaveProperty("topics")
  }, 20e3)

  it("eth_getFilterLogs", async () => {
    // Deploy ERC-20 contract
    const contract = await deployERC20()
    const address = await contract.getAddress()
    const filter = {
      fromBlock: "latest",
      address: address,
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
  }, 20e3)

  it("eth_getLogs", async () => {
    // Deploy ERC-20 contract
    const contract = await deployERC20()
    const address = await contract.getAddress()
    const filter = {
      fromBlock: "latest",
      address: address,
    }
    // Execute some contract TX
    const tx = await contract.transfer(alice, parseEther("0.01"))
    await tx.wait(1, 5e3)

    // Assert logs
    const changes = await provider.send("eth_getLogs", [filter])
    expect(changes.length).toBeGreaterThan(0)
    expect(changes[0]).toHaveProperty("address")
    expect(changes[0]).toHaveProperty("data")
    expect(changes[0]).toHaveProperty("topics")
  }, 20e3)
})
