const {ethers} = require("ethers");
const {config} = require('dotenv')
const fs = require("fs");

config()

const rpcEndpoint = process.env.JSON_RPC_ENDPOINT
const mnemonic = process.env.MNEMONIC

const provider = ethers.getDefaultProvider(rpcEndpoint)
const wallet = ethers.Wallet.fromPhrase(mnemonic)
const account = wallet.connect(provider)

const deployContract = async (path) => {
    const contractJSON = JSON.parse(
        fs.readFileSync(`contracts/${path}`).toString()
    )
    const bytecode = contractJSON['bytecode']
    const abi = contractJSON['abi']

    const contractFactory = new ethers.ContractFactory(abi, bytecode, account)
    const contract = await contractFactory.deploy()
    await contract.waitForDeployment()
    return contract
}

module.exports = {
    provider,
    account,
    deployContract,
}
