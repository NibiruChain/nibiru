const {ethers} = require('ethers')
const {account, provider, deployContract} = require('./setup')

describe('Basic Queries', () => {

    it('Simple transfer, balance check', async () => {
        const randomAddress = ethers.Wallet.createRandom().address
        const amountToSend = 1000n // unibi
        const gasLimit = 100_000n // unibi

        const senderBalanceBefore = await provider.getBalance(account.address)
        const recipientBalanceBefore = await provider.getBalance(randomAddress)

        expect(senderBalanceBefore).toBeGreaterThan(0)
        expect(recipientBalanceBefore).toEqual(0n)

        // Execute EVM transfer
        const transaction = {
            gasLimit: gasLimit,
            to: randomAddress,
            value: amountToSend
        }
        const txResponse = await account.sendTransaction(transaction)
        await txResponse.wait()
        expect(txResponse).toHaveProperty('blockHash')

        const senderBalanceAfter = await provider.getBalance(account.address)
        const recipientBalanceAfter = await provider.getBalance(randomAddress)

        // TODO: https://github.com/NibiruChain/nibiru/issues/1902
        // gas is not deducted regardless the gas limit, check this
        const expectedSenderBalance = senderBalanceBefore - amountToSend
        expect(senderBalanceAfter).toBeLessThanOrEqual(expectedSenderBalance)
        expect(recipientBalanceAfter).toEqual(amountToSend)
    }, 20_000)
})
