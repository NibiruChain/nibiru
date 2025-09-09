import { describe, expect, it } from "@jest/globals"
import { parseUnits, toBigInt, Wallet } from "ethers"

import { account, account2, account3, provider, TEST_TIMEOUT, TX_WAIT_TIMEOUT } from "./setup"
import { deployContractWNIBIIfNeeded, deployContractUSDCIfNeeded } from "./utils"

describe("Gastoken tests", () => {
    it(
        "Send tx with WNIBI token",
        async () => {
            const { contract: wnibi } = await deployContractWNIBIIfNeeded("0xF8Da4a4A57e4aFBdeA4c541DCa626a47Ed874729")
            const { contract: usdc } = await deployContractUSDCIfNeeded("0x869EAa3b34B51D631FB0B6B1f9586ab658C2D25F")
            console.log("account2:", account2.address)

            const walletBalWei = await provider.getBalance(account2.address)
            const wnibiBal = await wnibi.balanceOf(account2.address)
            const usdcBal = await usdc.balanceOf(account2.address)

            // Make sure there is no balance initially
            expect(walletBalWei).toEqual(0n)
            expect(wnibiBal).toEqual(0n)
            expect(usdcBal).toEqual(0n)

            // Send some WNIBI to account2 to use as gas token
            {
                let tx = await wnibi.transfer(account2, parseUnits("1", 18))
                await tx.wait()
                const wnibiBal = await wnibi.balanceOf(account2.address)
                expect(wnibiBal).toEqual(parseUnits("1", 18))
            }

            // Send half of the WNIBI to account
            {
                const amountToSend = parseUnits("0.5", 18)
                let tx = await wnibi.connect(account2).transfer(account, amountToSend)
                await tx.wait()
                const wnibiBal = await wnibi.balanceOf(account2.address)
                expect(wnibiBal).toBeLessThan(parseUnits("0.5", 18))
            }
        },
        TEST_TIMEOUT,
    )

    it(
        "Send tx with USDC token",
        async () => {
            const { contract: wnibi } = await deployContractWNIBIIfNeeded("0xF8Da4a4A57e4aFBdeA4c541DCa626a47Ed874729")
            const { contract: usdc } = await deployContractUSDCIfNeeded("0x869EAa3b34B51D631FB0B6B1f9586ab658C2D25F")

            const walletBalWei = await provider.getBalance(account3.address)
            const wnibiBal = await wnibi.balanceOf(account3.address)
            const usdcBal = await usdc.balanceOf(account3.address)

            // Make sure there is no balance initially
            expect(walletBalWei).toEqual(0n)
            expect(wnibiBal).toEqual(0n)
            expect(usdcBal).toEqual(0n)

            // Mint some USDC to account3 to use as gas token
            {
                let tx = await usdc.transfer(account3, parseUnits("1", 18))
                await tx.wait()
                const usdcBal = await usdc.balanceOf(account3.address)
                expect(usdcBal).toEqual(parseUnits("1", 18))
            }

            // Send half of the USDC to account
            {
                const amountToSend = parseUnits("0.5", 18)
                let tx = await usdc.connect(account3).transfer(account, amountToSend)
                await tx.wait()
                const usdcBal = await usdc.balanceOf(account3.address)
                expect(usdcBal).toBeLessThan(parseUnits("0.5", 18))
            }
        },
        TEST_TIMEOUT,
    )
})