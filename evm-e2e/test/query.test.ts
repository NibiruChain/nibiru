import { expect, test } from "@jest/globals"
import { parseUnits, toBigInt, Wallet } from "ethers"

import { account, provider, TEST_TIMEOUT, TX_WAIT_TIMEOUT } from "./setup"
import { deployContractWNIBI, getWNIBIContract } from "./utils"

test(
  "query",
  async () => {
    const contract = await getWNIBIContract("0xF8Da4a4A57e4aFBdeA4c541DCa626a47Ed874729")
    

    {
      const balanceOf = await contract.balanceOf(account.address)
      console.log("balanceOf:", balanceOf.toString())
    }

  },
  TEST_TIMEOUT * 2,
)
