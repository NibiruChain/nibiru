import { expect, test } from "@jest/globals"
import { parseUnits, toBigInt, Wallet } from "ethers"

import { account, provider, TEST_TIMEOUT, TX_WAIT_TIMEOUT } from "./setup"
import { deployContractWNIBI } from "./utils"

test(
  "deploy WNIBI and deposit",
  async () => {
    const { contract } = await deployContractWNIBI()
    
    const decimals = await contract.decimals()
    expect(decimals).toEqual(BigInt(18))
    const contractAddr = await contract.getAddress()
    console.log("WNIBI contract address:", contractAddr)
    console.log("user :", account.address)

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
  },
  TEST_TIMEOUT * 2,
)
