const {ethers} = require('ethers')
const {account, provider, deployContract} = require('./setup')

let contract

const doContractSend = async (sendMethod) => {
    const recipientAddress = ethers.Wallet.createRandom().address
    const transferValue = 100n * 10n ** 6n // NIBI

    const ownerBalanceBefore = await provider.getBalance(account.address) // NIBI
    const recipientBalanceBefore = await provider.getBalance(recipientAddress) // NIBI
    expect(recipientBalanceBefore).toEqual(0n)

    let tx = await contract[sendMethod](recipientAddress, {value: transferValue})
    await tx.wait()

    const ownerBalanceAfter = await provider.getBalance(account.address) // NIBI
    const recipientBalanceAfter = await provider.getBalance(recipientAddress) // NIBI

    expect(ownerBalanceAfter).toBeLessThanOrEqual(ownerBalanceBefore - transferValue)
    expect(recipientBalanceAfter).toEqual(transferValue)
}

describe('Send NIBI from smart contract', () => {

    beforeAll(async () => {
        contract = await deployContract('SendNibiCompiled.json')
    })

    it.each([
        ['sendViaTransfer'],
        ['sendViaSend'],
        ['sendViaCall'],
    ])('send nibi via %p method', async (sendMethod) => {
        await doContractSend(sendMethod)
    }, 20000);
})
