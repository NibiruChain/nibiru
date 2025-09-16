---
order: 8
canonicalUrl: "https://nibiru.fi/docs/concepts/gas.html"
---

# Gas on Nibiru and its EVM

Gas on Nibiru represents the amount of computational effort required to
execute specific operations on the state machine. {synopsis}

<!-- DO NOT DELETE: This comment serves as useful guide on using `{prereq}`
## Pre-requisite Readings

- [Ethereum Gas](https://ethereum.org/en/developers/docs/gas/) {prereq}
- [Cosmos SDK Gas](https://docs.cosmos.network/master/basics/gas-fees.html) {prereq}
-->

| In this Section | Description |
| --- | --- |
| [Ante Handler](#ante-handler) | Understand the implementation of gas management on the Nibiru blockchain. This includes including authentication, fee deduction, and gas meter setup.  |
| [Gas on Ethereum](#gas-on-ethereum) | Brief primer on the purpose of gas on Ethereum |
| [Gas on Nibiru](#gas-on-nibiru) | Brief primer on Nibiru's gas model |
| [Nibiru EVM Gas Precision](#nibiru-evm-gas-and-wei-precision) | (🩵 Important!) Explains the precision of gas and wei in the Nibiru EVM, emphasizing the relationship between NIBI, microNIBI, and wei, and how it differs from standard Ethereum precision. |

## Ante Handler

The term "ante" originates from the game of poker, where it refers to the minimum
amount that each player must contribute to the pot before a new round of the game
can begin.

In analagous fashion, before any transaction is processed on Nibiru, it must pass
through an initial checkpoint known as the Ante Handler, which is responsible for
preliminary checks and modifications to the gas requirements of a transaction
before it is passed on before it is passed onto the actual message handling.

The Ante Handler plays a crucial role in maintaining the integrity of Nibiru by
checking the authenticity of transactions, deducting fees, and controlling
resource usage. It serves as a protective barrier, ensuring that only valid
transactions are processed, imposes costs, and discourages users from overloading
the network with unnecessary data.

#### Responsibilities of the Ante Handler

1. **Authentication:** The Ante Handler verifies that the transaction is valid by
   checking the signatures and sequence numbers. This ensures that the
transaction is indeed from the stated sender, and prevents replay attacks.

2. **Fee Deduction:** Nibiru transactions may include a fee that is deducted from
   the sender's account. The Ante Handler handles this deduction.

3. **Gas Meter Setup:** Every transaction in Nibiru consumes computational
   resources, quantified as gas. The Ante Handler sets up a gas meter for the
transaction, deducting the gas fees from the user's account.

4. **Check Memo Size:** A transaction memo is an optional piece of metadata that
   can be stored on chain with a transaction. Memos don't change message payloads
for state transitions, so someone could potentially spam the network by sending
memos that take up lots of memory. For this reason, the chain enforces a maximum
allowed size for each transaction memo. The Ante Handler verifies that this
maximum size is not exceeded.

## Gas on Ethereum

Gas was created on Ethereum to disallow the EVM (Ethereum Virtual Machine) from
running infinite loops by allocating a small amount of monetary value into the
system. A unit of gas, usually taken to be some fraction of a network's native
coin, is consumed for every operation on the EVM and requires a user to pay for
stateful operations and calling smart contracts.

## Gas on Nibiru

Exactly like Ethereum, Nibiru utilizes the concept of gas to track the resource
consumption during message execution. Operations on Nibiru are represented as
"reads" or "writes" done to the chain's key-value store. In both networks, gas
is used to make sure that operations do not require an excess amount of
computational power to complete and as a way to deter bad-acting users from
spamming the network.

On Nibiru, a fee is calculated and charged to the user during a message
execution. This fee is calculated from the sum of all gas consumed in an
message execution:

```
gas fee = gas * gas price
```

Gas is tracked in the main `GasMeter` and the `BlockGasMeter`:

- `GasMeter`: keeps track of the gas consumed during executions that lead to
  state transitions. It is reset on every transaction  execution.
- `BlockGasMeter`: keeps track of the gas consumed in a block and enforces that
  the gas does not go over a predefined limit. This limit is defined in the
  Tendermint consensus parameters and can be changed via governance parameter
  change proposals.

## Nibiru EVM Gas and Wei Precision

\[Definition\] In the Nibiru EVM, "NIBI is ether". But, the smallest unit of NIBI is microNIBI,
which is often denoted as "unibi" onchain (becuase "u" looks like $\mu$, which
stands for micro in mathematics). This has some implications for precision with
Nibiru's Ethereum transactions. 

:::tip
 One wei, pronounced "way", is defined as one quintillionth of an
  ether. Thus, `1 ether = 10^{18} wei`.  However, "NIBI is ether" (`1 ether := 1 NIBI`) and this implies that the smallest unit of NIBI
(microNIBI) is such that `1 microNIBI = 10^{-6} NIBI = 10^{12} wei`. <br>Thus, the
smallest transferrable wei value on the Nibiru EVM is $10^{12}$.
:::

### Q1: So what happens if you `sendTransaction` and specify a wei amount smaller than $10^{12}$?
A: You'll get an error on the transfer that says "wei amount is too small ..., cannot transfer less than 1 micronibi. 10^18 wei == 1 NIBI == 10^6 micronibi".

### Q2: How are wei amounts handled when they are greater than 1 microNIBI but have some "dust", like in the case of `1.05 * 10^{12}`?
A: Many teams building on the Nibiru EVM noted that throwing an error whenever a
wei amount is not divisible by $10^{12}$ adds way too much friction in getting
smart contracts right. 

To circumvent this issue and retain compatibility with Solidity variables like
"ether", we opted for a practical solution where wei values are truncated to the
highest multiple of microNIBI. For example,
```
Input  number:  123456789012345678901234567890
Parsed number:  123456789012 * 10^12
```

So it's common to write numbers like this when building user-facing appications
for Nibiru:
```js
import { parseUnits } from "ethers"
const amountWei = parseUnits("351", 12)
```

:::warning 
If you Solidity contracts validate wei values using `require(msg.value == X)`
checks or similar precision-dependent validations, it's important to make sure
this invariant is updated to follow Nibiru's invariant and not Ethereum's.
:::

### Q3: Why didn't we choose $10^{-18}$ as the smallest unit of NIBI in the first place?
Nibiru V1 was built on CometBFT and the Cosmos-SDK using fungible digital assets
that we call Bank Coins (since they're created by the Bank Module). By
convention, these tokens use a decimal precision of 6 and use a lowercase
"u"-prefixed name to refer to their smallest unit. 

You'll notice that ATOM is called "uatom" onchain and Noble USDC is called "uusdc" for this exact reason. Almost all IBC-compatible chains have a native asset with 6 digits of precision, not 18.

EVM support was an additive upgrade to Nibiru while there were already half a
million holders of the NIBI token, so we had to develop the Nibiru EVM in such a
way that all of the existing NIBI state would not be corrupted while adapting to
as many of the conventions of Ethereum that we could.

## Gas Refunds

In Ethereum, gas can be specified prior to execution and the remaining gas will
be refunded back to the user if any gas is left over - should fail with out of
gas if not enough gas was provided. 

:::tip
Note however that for EVM transactions ("`evm.MsgEthereumTx`"), Nibiru matches exactly the Ethereum gas consumption as dictated by the logic in Go-ethereum (geth).
:::

On the non-EVM side of Nibiru, the concept of gas refunds does not exist. Fees
paid to execute a transaction are not refunded back to the user due to the way
instant finality works on Tendermint consensus. The fees exacted on a transaction
will be collected by the validator and no refunds are given to the transaction
sender upon failure.

Since gas refunds are not applicable on non-EVM Nibiru transactions, it's extremely important to
specify the correct gas as upfront. 
 
To prevent overspending on fees, providing the `--gas-adjustment` flag for a
cosmos transactions will determine the fees automatically. Often, setting the
`--fee` to "auto" will simulate the transaction and give back the correct gas
cost for the transaction, however this simulation is not perfect.


## Zero Fee Transactions

In Cosmos, a minimum gas price is not enforced by the `AnteHandler` as the
`min-gas-prices` is checked against the local node/validator. In other words, the
minimum fees accepted are determined by the validators of the network, and each
validator can specify a different value for their fees. This potentially allows
end users to submit 0 fee transactions if there is at least one single validator
that is willing to include transactions with `0` gas price in their blocks
proposed.

For this same reason, it is possible to send certain transaction types on Nibiru
with `0` gas fees. 

---

## Readings to Dive Deeper

- [Ethereum Gas](https://ethereum.org/en/developers/docs/gas/)
