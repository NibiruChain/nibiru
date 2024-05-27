const {ethers} = require('ethers')
const {config} = require('dotenv')
const fs = require('fs')

config()

describe('Ethereum JSON-RPC Interface Tests', () => {
    let provider
    let wallet
    let account

    beforeAll(async () => {
        const rpcEndpoint = process.env.JSON_RPC_ENDPOINT
        const mnemonic = process.env.MNEMONIC
        provider = ethers.getDefaultProvider(rpcEndpoint)
        wallet = ethers.Wallet.fromPhrase(mnemonic)
        account = wallet.connect(provider)
    })

    test('Simple Transfer, balance check', async () => {
        const randomAddress = ethers.Wallet.createRandom().address
        const amountToSend = ethers.toBigInt(1000) // unibi
        const gasLimit = ethers.toBigInt(100_000) // unibi

        const senderBalanceBefore = await provider.getBalance(wallet.address)
        const recipientBalanceBefore = await provider.getBalance(randomAddress)

        expect(senderBalanceBefore).toBeGreaterThan(0)
        expect(recipientBalanceBefore).toEqual(ethers.toBigInt(0))

        // Execute EVM transfer
        const transaction = {
            gasLimit: gasLimit,
            to: randomAddress,
            value: amountToSend
        }
        const txResponse = await account.sendTransaction(transaction)
        await txResponse.wait()
        expect(txResponse).toHaveProperty('blockHash')

        const senderBalanceAfter = await provider.getBalance(wallet.address)
        const recipientBalanceAfter = await provider.getBalance(randomAddress)

        // TODO: gas is not deducted regardless the gas limit, check this
        const expectedSenderBalance = senderBalanceBefore - amountToSend
        expect(senderBalanceAfter).toBeLessThanOrEqual(expectedSenderBalance)
        expect(recipientBalanceAfter).toEqual(amountToSend)
    }, 20_000)

    test('Smart Contract', async () => {
        // Read contract ABI and bytecode
        const contractJSON = JSON.parse(
            fs.readFileSync('contracts/FunTokenCompiled.json').toString()
        )
        const bytecode = contractJSON['bytecode']
        const abi = contractJSON['abi']

        // Deploy contract
        const contractFactory = new ethers.ContractFactory(abi, bytecode, account)
        const contract = await contractFactory.deploy()
        await contract.waitForDeployment()
        const contractAddress = await contract.getAddress()
        expect(contractAddress).toBeDefined()

        // Execute contract: ERC20 transfer
        const shrimpAddress = ethers.Wallet.createRandom().address
        let ownerInitialBalance = ethers.parseUnits("1000000", 18)

        const amountToSend = ethers.parseUnits("1000", 18) // contract tokens

        let ownerBalance = await contract.balanceOf(account.address)
        let shrimpBalance = await contract.balanceOf(shrimpAddress)

        expect(ownerBalance).toEqual(ownerInitialBalance)
        expect(shrimpBalance).toEqual(ethers.toBigInt(0))

        let tx = await contract.transfer(shrimpAddress, amountToSend)
        await tx.wait()

        ownerBalance = await contract.balanceOf(account.address)
        shrimpBalance = await contract.balanceOf(shrimpAddress)

        expect(ownerBalance).toEqual(ownerInitialBalance - amountToSend)
        expect(shrimpBalance).toEqual(amountToSend)
    }, 20000)
})
