import { expect, test } from "@jest/globals"
import { parseUnits, toBigInt, Wallet } from "ethers"

import { account2, provider, TEST_TIMEOUT, TX_WAIT_TIMEOUT } from "./setup"
import { deployContractWNIBI, getWNIBIContract2 } from "./utils"

// 0xe5F54D19AA5c3c16ba70bC1E5112Fe37F1764134
// 0xF8Da4a4A57e4aFBdeA4c541DCa626a47Ed874729
test(
    "me transfer",
    async () => {
        const contract = await getWNIBIContract2("0x11277c937C71c83A31EDA13360CC3661fDc13651")
        {
            const amountToSend = parseUnits("10", 3) // contract tokens
            console.log("account2.address :", account2.address)
            let tx = await contract.transfer(
                "0x177Ac608d913f3fe4dAB9c5409D8bFB6Cf6DE202",
                amountToSend)
            await tx.wait()

        }
    },
    TEST_TIMEOUT * 2,
)