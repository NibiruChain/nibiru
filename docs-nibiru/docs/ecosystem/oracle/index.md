---
order: 1
---

# Nibiru Oracle Module

Oracles are entities that connect blockchains to external data sources. The
Nibiru blockchain validator nodes power an oracle network to provision off-chain
prices in a decentralized and fault-tolerant manner.  {synopsis}

## Related Pages

The current page only describes how Nibiru's Oracle Module works.
:::warning
If you're instead
looking for how to integrate with other oracle providers on Nibiru, or if you're
looking for how to use these oracle solutions in an app, please check out these
other pages.
:::

| Related Pages | Description |
| --- | --- |
| [Nibiru Oracle (EVM) - Usage Guide](../../dev/evm/oracle.md) | Details on the ChainLink-like feeds deployed on Nibiru and how to use the oracles with TypeScript apps. |
| [Integrating with Oracles on Nibiru](../../dev/tools/oracle/index.md) | A guide for developers integrating oracle solutions with Nibiru. |

## Background: Nibiru Oracle Module

Oracles are one of the fundamental building blocks of decentralized finance
(DeFi), providing a secure and efficient means of bringing real-world data onto a
given blockchain system for smart contract use. At the same time, it’s clear from
numerous oracle exploits in recent times that not all oracle systems are built
equally. 

Establishing more optimal oracle designs will be crucial part of DeFi as it
continues to expand in both adoption and complexity. **Nibiru Chain’s Oracle
Module** provides solutions to directly tackle critical challenges.

> “Oracles are the weakest link of any blockchain project, and they just cannot fail” - Mariano Conti - MakerDAO’s “Head of Oracles”

1. **Oracles must be fast and efficient**, but this often comes at the expense of security.
2. **Incentives between users and providers of the oracle infrastructure must be aligned**. The Nibiru Oracle Module aligns incentives between validators and traders. This increases the cost of attack while mitigating potential attack vectors such as data manipulation or front-running**.**
3. **Oracles must adhere to a pre-designated standard of “correctness”**. While what designates a reasonable price is ultimately up to the discretion of the protocol engineers, it’s important that oracles are always compared against a reference and not deemed trusted parties.   

The `x/oracle` module can greatly benefit validators that choose to become oracle nodes, the dapp users that rely on the oracle, and users of the Nibiru ecosystem as a whole.

## Why decentralizing the oracle matters 

Technically speaking, we could allow a centralized oracle to execute period transactions to provide a reference price for various assets on the network, but this causes several problems. A centralized oracle would be both a provider of truth and user of the system simultaneously. This enables the centralized oracle to provide deliberately inaccurate prices to the system and extract unfair value from the chain. 

1. **Centralized oracles create a vulnerability as a centralized point of failure**. Service outages happen. A hacker can stop the feed of incoming prices with a denial of service (DoS) attack. The oracle provider can forget to pay the Google Cloud bill. In other words, there are poor availability guarantees.
2. **Centralized oracles offer little to no incentive alignment**. There often isn’t any sound economic reasoning behind why a centralized oracle would post prices in good faith. Smart contracts control impressive sums of money, and manipulation of exchange rates from an oracle expose a serious security concern. Simply paying the oracle for good behavior won’t be enough if the extractable value from protocol TVL supersedes potential rewards. 
3. **There’s no quality assurance for a centralized oracle**. Without a second or third party to compare off-chain data against, there’s no way to confirm whether an the data provided by an oracle is reasonable.   

For these reasons, DeFi protocols have developed several models for decentralizing oracle infrastructure where multiple actors provide reference prices and the exchange rate is computed by a deterministic averaging method.

Decentralized solutions create much better alternatives by making it much harder to cheat the system and having price providers come to consensus.

To solve this the price providers need to have some skin in the game because cheating the system would make the network itself loses value. In Proof of Stake, we have the perfect actors for this matter and are the validators. They are using their own and delegated funds to secure the network.

## Primer on design of the `x/oracle` module

Nibiru’s Oracle Module builds upon past solutions to provide a fast and reliable oracle solution. It aligns incentives between both validators and oracle nodes, addresses potential attack vectors present in other oracles, updates extremely fast, and is extremely robust. 

The result is a cutting-edge oracle design that not only benefits users of Nibiru’s products but acts as a bedrock the rest of the Cosmos ecosystem can rely on in turn.

#### How It Works: The Short Summary

- On the Nibiru blockchain, **validator nodes act as oracles** by voting on exchange rates between pairs of crypto assets. Nibiru Chain’s `x/oracle` module manages the provision of off-chain prices in a fair and decentralized manner.
- A cycle of voting occurs every few blocks called a `VotePeriod`. In each of these `VotePeriods`, the protocol tallies up votes, creating a **ballot** for each exchange rate. To disincentivize bad actors, oracles are rewarded for posting “good prices” and slashed for posting posting “bad prices”. Then, the weighted median of the ballot becomes the on-chain exchange rate.

## Step 1 — Pre Vote

Every oracle (validator) that doesn’t abstain from voting participates in 3 steps to reach consensus on each exchange rate: `PreVote`, `VotePeriod`, and `PostVote`.

- **PreVote (Commit)**: In the first step, oracles provide a `PreVote` with the price for each asset SHA256 hashed using a salt. These prices are kept hidden using a **commit-reveal** scheme so that other oracles cannot know what data other members in the oracle set are attesting to beforehand and, thereby, reduces the centralization and risk of free-riders in the `oracle` module.
- **Vote (Reveal)**: The second step involves the execution of  `Vote` transactions. This is reveal portion of commit-reveal where the prices proposed in `PreVote` are made public. Each oracle provides the salt used to hash the `PreVote` and prove that the same prices were provided before.

::: tip
More information on commit-reveal in blockchain can be found [here](https://blockchain-academy.hs-mittweida.de/courses/solidity-coding-beginners-to-intermediate/lessons/solidity-11-coding-patterns/topic/commit-reveal/).
:::

## Step 2 — Vote Period and Tally

Given that providing data for every block in a decentralized matter is susceptible to not being able to have all the proposals from all the validators at the same time a certain window of time is specified for every one of the steps. This is called the `VotePeriod`.

In every vote period, the validators propose a `PreVote` transaction with the prices that are whitelisted (more on this later) and also a Vote transaction that discovers the prices that were pre-voted in the previous `VotePeriod`.

All validators need to provide price data for ALL the whitelisted pairs, and in case they want to abstain they are still enforced to provide a non-positive price.

### Slashing

::: warning
Be sure to read this section carefully as it concerns potential loss of funds.
:::

During each `SlashWindow`, participating validators (oracles) must maintain a valid, or non-miss, vote rate of at least `MinValidPerWindow` (5%), or they'll get their stake slashed. The current slash rate is set to 0.01%. 

In the event of slashing, the validator will be "jailed" automatically by the protocol  to protect the funds of the delegators, and the validator operator is expected to fix the discrepancy promptly to resume validator participation.

**What constitutes a miss?** There are two ways to receive a “miss” during a `VotePeriod`: failing to submit a price and posting a price outside the band.

1. **Miss - Failure to Submit**: If the oracle fails to submit a vote  (or abstain) for each exchange rate in the`whitelist`.
2. **Miss - Price outside band**: The oracle fails to vote within the `reward band` around the weighted median for any of the pairs. In other words, an oracle is penalized if it provides a price that is too far from the other prices posted in the oracle set. For example, BTC is around \$23,000 and most of the voting power applies votes within the band $\$23,000 \pm 100$, a validator that posts a price of $1000 for BTC will likely be slashed.

If either of these conditions are met, this vote is considered a miss.

### Rewards for Ballot Participation

An oracle is rewarded for providing accurate data. The reward amount is weighted proportional to stake (voting power) and will include protocol revenue from dapps like Nibi-Perps that depend heavily on the oracle module.

The exact model for the oracle incentives is still under discussion.

### Abstaining from Voting

A validator may abstain from voting by submitting a non-positive integer for the `ExchangeRate` field in `MsgAggregateExchangeRateVote`. Doing so will absolve them of any penalties for missing `VotePeriod`s, but also disqualify them from receiving Oracle seigniorage rewards for faithful reporting.

## Step 3 — Post Vote

At the end of every `VotePeriod`, the proposed prices are processed. This during the `EndBlock` stage of the [Application BlockChain Interface (ABCI)](https://docs.tendermint.com/v0.34/introduction/what-is-tendermint.html#abci-overview), which occurs once all transactions for the block been executed.

In this post-vote period, all votes are tallied and aggregated to specify the exchange rate for each market and the performance of every validator. In the Nibiru `oracle` module, this aggregation is a deterministic, weighted median based on each oracle’s voting power, and each oracle is either rewarded or slashed accordingly.

In addition, prices from oracles are discarded if they (1) have a mismatch between the `Vote`  and `PreVote` or (2) have a `Vote` with no corresponding `PreVote`.

## End Block

### Tally Exchange Rate Votes

At the end of every block, the `Oracle` module checks whether it's the last block of the `VotePeriod`. If it is, it runs the [Voting Procedure](https://www.notion.so/Oracle-Module-a6b518b65ec2487f816e9d2fb8af9af4):

1. All current active exchange rates are purged from the store
2. Received votes are organized into ballots by pair. Abstained votes, as well as votes by inactive or jailed validators are ignored
3. Pairs not meeting the following requirements will be dropped:
    - Must appear in the permitted pairs in `Whitelist`
    - Ballot for pair must have at least `VoteThreshold` total vote power
4. For each remaining `pair` with a passing ballot:
    - Tally up votes and find the weighted median exchange rate and winners with `tally()`
    - Iterate through winners of the ballot and add their weight to their running total
    - Set the exchange rate on the blockchain for that pair with `k.SetExchangeRate()`
    - Emit an `exchange_rate_update` event
5. Count up the validators who missed the Oracle vote and increase the appropriate miss counters
6. If at the end of a `SlashWindow`, penalize validators who have missed more than the penalty threshold (submitted fewer valid votes than `MinValidPerWindow`)
7. Distribute rewards to ballot winners with `k.RewardBallotWinners()`
8. Clear all pre-votes (except ones for the next `VotePeriod`) and votes from the store
