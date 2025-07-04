import { expect, test } from "@jest/globals"
import { parseUnits, toBigInt, Wallet } from "ethers"

import { account, provider, TEST_TIMEOUT, TX_WAIT_TIMEOUT } from "./setup"
import { deployContractWNIBI, getWNIBIContract } from "./utils"

// 0xe5F54D19AA5c3c16ba70bC1E5112Fe37F1764134
// 0xF8Da4a4A57e4aFBdeA4c541DCa626a47Ed874729
test(
  "query",
  async () => {
    const contract = await getWNIBIContract("0xe5F54D19AA5c3c16ba70bC1E5112Fe37F1764134")
    

    {
      const total = await contract.totalSupply()
      console.log("totalSupply:", total.toString())
      const balanceOf = await contract.balanceOf(account.address)
      console.log("balanceOf:", balanceOf.toString())
      console.log("account.address :", account.address)
    }

  },
  TEST_TIMEOUT * 2,
)
