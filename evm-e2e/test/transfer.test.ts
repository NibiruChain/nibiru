import { expect, test } from "@jest/globals"
import { parseUnits, toBigInt, Wallet } from "ethers"

import { account, provider, TEST_TIMEOUT, TX_WAIT_TIMEOUT } from "./setup"
import { deployContractWNIBI, getWNIBIContract } from "./utils"

// 0xe5F54D19AA5c3c16ba70bC1E5112Fe37F1764134
// 0xF8Da4a4A57e4aFBdeA4c541DCa626a47Ed874729
test(
    "transfer for me",
    async () => {
        const contract = await getWNIBIContract("0x11277c937C71c83A31EDA13360CC3661fDc13651")

        {
            const amountToSend = parseUnits("10", 18) // contract tokens

            let tx = await contract.transfer(
                "0xe35fdcAde710De4E7A1889dD15dE6c656aba21f0",
                amountToSend)
            await tx.wait()

        }
    },
    TEST_TIMEOUT * 2,
)
