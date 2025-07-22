import { expect, test } from "@jest/globals"
import { parseUnits, toBigInt, Wallet } from "ethers"

import { account, account2, provider, TEST_TIMEOUT, TX_WAIT_TIMEOUT } from "./setup"
import { deployContractWNIBIIfNeeded } from "./utils"

test(
    "txfees tests",
    async () => {
        const { contract } = await deployContractWNIBIIfNeeded("0xF8Da4a4A57e4aFBdeA4c541DCa626a47Ed874729")

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

        // Send to account2 who has no unibi
        {
            const amountToSend = parseUnits("10", 18) // contract tokens
            let tx = await contract.transfer(
                account2.address,
                amountToSend)
            await tx.wait()

            // check balances after transfer
            const account2Bal = await contract.balanceOf(account2.address)
            const wallet2Bal = await provider.getBalance(account2.address)
            expect(account2Bal).toEqual(amountToSend)
            expect(wallet2Bal).toEqual(0n)
        }

        // Account2 makes a transfer to a random address 
        {
            const amountToSend = parseUnits("1", 18) // contract tokens
            let tx = await contract.connect(account2).transfer(
                "0x177Ac608d913f3fe4dAB9c5409D8bFB6Cf6DE202",
                amountToSend)
            await tx.wait()

            // check balances after transfer, should be less than 9^18 as there is a fee
            const account2Bal = await contract.balanceOf(account2.address)
            expect(account2Bal).toBeLessThan(parseUnits("9", 18))
        }
    },
    TEST_TIMEOUT * 2,
)
