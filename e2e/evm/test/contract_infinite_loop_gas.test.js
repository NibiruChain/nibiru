const {deployContract} = require('./setup')

describe('Infinite loop gas contract', () => {
    let contract

    beforeAll(async () => {
        contract = await deployContract('InfiniteLoopGasCompiled.json')
    })

    it('should fail due to out of gas error', async () => {
        const initialCounter = await contract.counter()
        expect(initialCounter).toBe(0n)

        try {
            const tx = await contract.forever({gasLimit: 1000000})
            await tx.wait()
            fail("The transaction should have failed but did not.")
        } catch (error) {
            expect(error.message).toContain("transaction execution reverted")
        }
        const finalCounter = await contract.counter()
        expect(finalCounter).toEqual(initialCounter)
    }, 20000)
})
