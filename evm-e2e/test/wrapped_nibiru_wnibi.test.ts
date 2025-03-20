import { expect, test } from "@jest/globals"
import { parseUnits, toBigInt } from "ethers"

import { account, provider, TEST_TIMEOUT } from "./setup"
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

    // TODO: test: Deposit via method call
    // TODO: test: totalSupply and transfer tests
  },
  TEST_TIMEOUT,
)
