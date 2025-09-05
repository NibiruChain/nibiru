import { expect, test } from "@jest/globals"
import { parseUnits, toBigInt, Wallet } from "ethers"

import { account, account2, provider, TEST_TIMEOUT, TX_WAIT_TIMEOUT } from "./setup"
import { deployContractWNIBIIfNeeded, deployContractUSDCIfNeeded } from "./utils"

test(
    "txfees tests",
    async () => {
        const { contract: contract } = await deployContractWNIBIIfNeeded("0xF8Da4a4A57e4aFBdeA4c541DCa626a47Ed874729")
        const { contract: usdc } = await deployContractUSDCIfNeeded("0x869EAa3b34B51D631FB0B6B1f9586ab658C2D25F")
        console.log("WNIBI contract address:", await contract.getAddress())
        console.log("USDC contract address:", await usdc.getAddress())
        const decimals = await contract.decimals()
        expect(decimals).toEqual(BigInt(18))
        const contractAddr = await contract.getAddress()


        // Check balances before any actions
        const walletBalWei = await provider.getBalance(account.address)
        console.log("Wallet balance (wei):", walletBalWei.toString())
        expect(walletBalWei).toBeGreaterThan(parseUnits("999", 18))
        {
            const balanceOf = await contract.balanceOf(account.address)
        }

        // Mint some USDC to the account
        {
            let tx = await usdc.mint(account, parseUnits("1000", 6))
            await tx.wait()
            const balanceOf = await usdc.balanceOf(account.address)
            expect(balanceOf).toEqual(parseUnits("1000", 6))
        }

        // Deposit via transfer of wei
        {
            const amountWei = parseUnits("420", 18)
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

        // // Send to account2 who has no unibi
        // {
        //     const amountToSend = parseUnits("10", 18) // contract tokens
        //     let tx = await contract.transfer(
        //         account2.address,
        //         amountToSend)
        //     await tx.wait()

        //     // check balances after transfer
        //     const account2Bal = await contract.balanceOf(account2.address)
        //     const wallet2Bal = await provider.getBalance(account2.address)
        //     expect(account2Bal).toEqual(amountToSend)
        //     expect(wallet2Bal).toEqual(0n)
        // }
        {
            let tx = await usdc.mint(account2, parseUnits("100000000000", 18))
            await tx.wait()
            const balanceOf = await usdc.balanceOf(account2.address)
            expect(balanceOf).toEqual(parseUnits("100000000000", 18))
            console.log("Account2 USDC balance:", balanceOf.toString())
        }
        // Account2 makes a transfer to a random address 
        {
            const amountToSend = parseUnits("1", 18) // contract tokens
            let tx = await usdc.connect(account2).transfer(
                "0x177Ac608d913f3fe4dAB9c5409D8bFB6Cf6DE202",
                amountToSend)
            await tx.wait()


            // check balances after transfer, should be less than 9^18 as there is a fee
            const account2Bal = await usdc.balanceOf(account2.address)
            console.log("Account2 balance after transfer:", account2Bal.toString())
            expect(account2Bal).toBeLessThan(parseUnits("9", 18))

        }
    },
    TEST_TIMEOUT * 2,
)
