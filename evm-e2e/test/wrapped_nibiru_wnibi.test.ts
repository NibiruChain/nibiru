import { expect, test } from "@jest/globals"
import { parseUnits, toBigInt, Wallet } from "ethers"

import { account, provider, TEST_TIMEOUT, TX_WAIT_TIMEOUT } from "./setup"
import { deployContractWNIBI } from "./utils"

test(
  "WNIBI.sol deposits via ether transfer",
  async () => {
    const { contract } = await deployContractWNIBI()
    const decimals = await contract.decimals()
    expect(decimals).toEqual(BigInt(18))
    const contractAddr = await contract.getAddress()

    // Check balances before any actions
    const walletBalWei = await provider.getBalance(account.address)
    expect(walletBalWei).toBeGreaterThan(parseUnits("999", 18))
    {
      const balanceOf = await contract.balanceOf(account.address)
      expect(balanceOf).toEqual(BigInt(0))
    }

    // Deposit via transfer of wei
    {
      const amountWei = parseUnits("420", 12)
      const txResp = await account.sendTransaction({
        to: contractAddr,
        value: amountWei,
      })
      await txResp.wait()

      // Check balances after deposit
      const contractBalWei = await provider.getBalance(contractAddr)
      const contractBalWNIBI = await contract.balanceOf(contractAddr)
      const walletBalWNIBI = await contract.balanceOf(account.address)
      expect(contractBalWei).toEqual(amountWei)
      expect(contractBalWNIBI).toEqual(0n)
      expect(walletBalWNIBI).toEqual(amountWei)
    }

    // Withdraw
    {
      const amountWei = parseUnits("351", 12)
      const txResp = await contract.withdraw(amountWei)
      await txResp.wait()

      // Check balanaces after withdraw
      const contractBalWei = await provider.getBalance(contractAddr)
      const contractBalWNIBI = await contract.balanceOf(contractAddr)
      const walletBalWNIBI = await contract.balanceOf(account.address)
      expect(contractBalWei).toEqual(parseUnits("69", 12))
      expect(contractBalWNIBI).toEqual(0n)
      expect(walletBalWNIBI).toEqual(parseUnits("69", 12))
    }

    // Deposit via method call
    {
      const balanceBefore = await contract.balanceOf(account.address)
      const amountToSend = parseUnits("911", 12)
      let tx = await contract.deposit({
        value: amountToSend,
      })
      await tx.wait(1, TX_WAIT_TIMEOUT)

      // Check balanaces after deposit
      const balanceAfter = await contract.balanceOf(account.address)
      expect(balanceAfter).toEqual(balanceBefore + amountToSend)
    }

    // Transfer via method call and total supply check
    {
      const alice = Wallet.createRandom()
      const amountToSend = parseUnits("200", 12) // WNIBI tokens for alice

      let tx = await contract.transfer(
        alice.address,
        amountToSend,
      )
      await tx.wait(1, TX_WAIT_TIMEOUT)

      // Check balances after transfer and correct total supply
      const aliceBalance = await contract.balanceOf(alice)
      expect(aliceBalance).toEqual(amountToSend)

      const accountBalance = await contract.balanceOf(account.address)

      const totalSupply = await contract.totalSupply()
      expect(totalSupply).toEqual(
        accountBalance + aliceBalance,
      )
    }
  },
  TEST_TIMEOUT * 2,
)
