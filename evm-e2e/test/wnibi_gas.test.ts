import { describe, expect, it } from "@jest/globals"
import { parseUnits, toBigInt, Wallet } from "ethers"

import { account, account2, provider, TEST_TIMEOUT, TX_WAIT_TIMEOUT } from "./setup"
import { getWNIBI } from "./utils"

describe("WNIBI used as gas tests", () => {
    it(
        "Interaction of account with zero native balance but some WNIBI balance",
        async () => {
            const { contract: wnibi } = await getWNIBI()
            const walletBalWei = await provider.getBalance(account2.address)

            // Make sure there is no balance initially
            expect(walletBalWei).toEqual(0n)

            // Send some WNIBI to account2 to use as gas token
            {
                let tx = await wnibi.transfer(account2.address, parseUnits("10", 18))
                await tx.wait()
                const wnibiBal = await wnibi.balanceOf(account2.address)
                expect(wnibiBal).toEqual(parseUnits("10", 18))
            }

            {
                const alice = Wallet.createRandom()
                // Ensure non-zero fees so WNIBI balance drops below 9 after transfer
                const feeOverrides = {
                    maxFeePerGas: parseUnits("1", "gwei"),
                    maxPriorityFeePerGas: parseUnits("0", "gwei"),
                }
                let tx = await wnibi.connect(account2).transfer(alice.address, parseUnits("1", 18), feeOverrides)
                await tx.wait()
                const wnibiBal = await wnibi.balanceOf(alice.address)
                expect(wnibiBal).toEqual(parseUnits("1", 18))
                const wnibiBal2 = await wnibi.balanceOf(account2.address)
                // Less than 9 WNIBI left after transfer and gas
                expect(wnibiBal2).toBeLessThan(parseUnits("9", 18))
            }
        },
        TEST_TIMEOUT,
    )
})