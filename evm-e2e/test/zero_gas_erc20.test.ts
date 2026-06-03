import { newClog } from "@uniquedivine/jiyuu"
import { expect, test } from "bun:test" // eslint-disable-line import/no-unresolved
import { parseUnits, Wallet } from "ethers"

import { account, provider, TEST_TIMEOUT, TX_WAIT_TIMEOUT } from "./testdeps"
import { deployContractTestERC20 } from "./utils"
import { addZeroGasContract } from "./zero_gas_chain_helpers"

const { clog, cerr, clogCmd } = newClog(
  import.meta.url.includes("/")
    ? import.meta.url.split("/").pop()!
    : import.meta.url,
)

test(
  "fresh account can call whitelisted ERC20 with zero gas",
  async () => {
    clog("1 - Deploy ERC20 that will be treated as zero-gas once allowlisted.")
    const contract = await deployContractTestERC20()
    const zeroGasErc20Addr = await contract.getAddress()
    await addZeroGasContract(zeroGasErc20Addr)

    clog(`2 - Create a completely fresh account (no prior state, no NIBI) and a 
    random recipient.`)
    const fresh = Wallet.createRandom().connect(provider)
    const recipient = Wallet.createRandom().address

    clog(`3 - Fund the fresh account with ERC20 only (no NIBI), so it can call the contract
    while remaining gasless in terms of the native token.`)
    const ownerInitialBalance = await contract.balanceOf(account.address)
    expect(ownerInitialBalance).toBeGreaterThan(0n)
    const freshInitialTokenBalance = await contract.balanceOf(fresh.address)
    expect(freshInitialTokenBalance).toEqual(0n)
    const amountToFundFresh = parseUnits("100", 18)
    const fundTx = await contract.transfer(fresh.address, amountToFundFresh)
    await fundTx.wait(1, TX_WAIT_TIMEOUT)
    const freshTokenBalanceAfterFund = await contract.balanceOf(fresh.address)
    expect(freshTokenBalanceAfterFund).toEqual(amountToFundFresh)
    const freshNibiBefore = await provider.getBalance(fresh.address)
    expect(freshNibiBefore).toEqual(0n)
    const nonceBefore = await provider.getTransactionCount(fresh.address)

    clog(
      "4 - From the fresh account, send a zero-gas ERC20 transfer tx to the allowlisted contract.",
    )
    const contractFromFresh = contract.connect(fresh)
    const amountToSend = parseUnits("10", 18)
    const zeroGasTx = await contractFromFresh.transfer(
      recipient,
      amountToSend,
      {
        maxFeePerGas: 0n,
        maxPriorityFeePerGas: 0n,
      },
    )

    clog("Tx should succeed and be directed to the zero-gas ERC20 contract.")
    const receipt = await zeroGasTx.wait(1, TX_WAIT_TIMEOUT)
    expect(receipt.status).toEqual(1)
    expect(receipt.to?.toLowerCase()).toEqual(zeroGasErc20Addr.toLowerCase())

    clog(`5 - Zero-gas invariant: fresh account pays no NIBI for gas, but its 
    nonce advances.`)
    const freshNibiAfter = await provider.getBalance(fresh.address)
    expect(freshNibiAfter).toEqual(freshNibiBefore)
    const nonceAfter = await provider.getTransactionCount(fresh.address)
    expect(nonceAfter).toEqual(nonceBefore + 1)

    clog("6 - Application-level effect: ERC20 balances move as expected.")
    const freshTokenBalanceAfterSend = await contract.balanceOf(fresh.address)
    const recipientTokenBalance = await contract.balanceOf(recipient)
    expect(freshTokenBalanceAfterSend).toEqual(
      freshTokenBalanceAfterFund - amountToSend,
    )
    expect(recipientTokenBalance).toEqual(amountToSend)

    clog(`7 - A second transfer still pays no NIBI when the wallet supplies
    nonzero EIP-1559 fee fields.`)
    const secondRecipient = Wallet.createRandom().address
    const secondAmountToSend = parseUnits("5", 18)
    const freshNibiBeforeNonzeroFeeTx = await provider.getBalance(fresh.address)
    const nonceBeforeNonzeroFeeTx = await provider.getTransactionCount(
      fresh.address,
    )
    const nonzeroFeeTx = await contractFromFresh.transfer(
      secondRecipient,
      secondAmountToSend,
      {
        gasLimit: 780_749n,
        maxFeePerGas: 1_200_000_000_000n,
        maxPriorityFeePerGas: 1_000_000_000n,
      },
    )
    const nonzeroFeeReceipt = await nonzeroFeeTx.wait(1, TX_WAIT_TIMEOUT)
    expect(nonzeroFeeReceipt.status).toEqual(1)
    expect(nonzeroFeeReceipt.to?.toLowerCase()).toEqual(
      zeroGasErc20Addr.toLowerCase(),
    )

    const freshNibiAfterNonzeroFeeTx = await provider.getBalance(fresh.address)
    expect(freshNibiAfterNonzeroFeeTx).toEqual(freshNibiBeforeNonzeroFeeTx)
    const nonceAfterNonzeroFeeTx = await provider.getTransactionCount(
      fresh.address,
    )
    expect(nonceAfterNonzeroFeeTx).toEqual(nonceBeforeNonzeroFeeTx + 1)

    const freshTokenBalanceAfterNonzeroFeeTx = await contract.balanceOf(
      fresh.address,
    )
    const secondRecipientTokenBalance =
      await contract.balanceOf(secondRecipient)
    expect(freshTokenBalanceAfterNonzeroFeeTx).toEqual(
      freshTokenBalanceAfterSend - secondAmountToSend,
    )
    expect(secondRecipientTokenBalance).toEqual(secondAmountToSend)
  },
  TEST_TIMEOUT * 3,
)
