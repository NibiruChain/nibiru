# Oracle

The Oracle module provides the Nibiru blockchain with an up-to-date and accurate price feed of exchange rates of trading pairs.

As price information is extrinsic to the blockchain, the Nibiru network relies on validators to periodically vote on the current exchange rates, with the protocol tallying up the results once per `VotePeriod` and updating the on-chain exchange rate as the weighted median of the ballot.

> Since the Oracle service is powered by validators, you may find it interesting to look at the [Staking](https://github.com/cosmos/cosmos-sdk/tree/master/x/staking/spec/README.md) module, which covers the logic for staking and validators.

## Contents         <!-- omit in toc -->
- [Oracle](#oracle)
  - [Concepts](#concepts)
    - [Voting Procedure](#voting-procedure)
    - [Reward Band](#reward-band)
    - [Slashing](#slashing)
    - [Abstaining from Voting](#abstaining-from-voting)
    - [Messages](#messages)
  - [Module Parameters](#module-parameters)
  - [State](#state)
    - [ExchangeRate](#exchangerate)
    - [FeederDelegation](#feederdelegation)
    - [MissCounter](#misscounter)
    - [AggregateExchangeRatePrevote](#aggregateexchangerateprevote)
    - [AggregateExchangeRateVote](#aggregateexchangeratevote)
  - [End Block](#end-block)
    - [Tally Exchange Rate Votes](#tally-exchange-rate-votes)
  - [Messages](#messages-1)
    - [MsgAggregateExchangeRatePrevote](#msgaggregateexchangerateprevote)
    - [MsgAggregateExchangeRateVote](#msgaggregateexchangeratevote)
    - [MsgDelegateFeedConsent](#msgdelegatefeedconsent)
  - [Events](#events)
    - [EndBlocker](#endblocker)
    - [Events for MsgExchangeRatePrevote](#events-for-msgexchangerateprevote)
    - [Events for MsgExchangeRateVote](#events-for-msgexchangeratevote)
    - [Events for MsgDelegateFeedConsent](#events-for-msgdelegatefeedconsent)
    - [Events for MsgAggregateExchangeRatePrevote](#events-for-msgaggregateexchangerateprevote)
    - [Events for MsgAggregateExchangeRateVote](#events-for-msgaggregateexchangeratevote)

---

## Concepts

See [docs.nibiru.fi/ecosystem/oracle](https://docs.nibiru.fi/ecosystem/oracle/).

### Voting Procedure

During each `VotePeriod`, the Oracle module obtains consensus on the exchange rate of pairs specified in `Whitelist` by requiring all members of the validator set to submit a vote for exchange rates before the end of the interval.

Validators must first pre-commit to a exchange rate, then in the subsequent `VotePeriod` submit and reveal their exchange rate alongside a proof that they had pre-commited at that price. This scheme forces the voter to commit to a submission before knowing the votes of others and thereby reduces centralization and free-rider risk in the Oracle.

* Prevote and Vote

    Let `P_t` be the current time interval of duration defined by `VotePeriod` (currently set to 30 seconds) during which validators must submit two messages:

  * A `MsgAggregateExchangeRatePrevote`, containing the SHA256 hash of the exchange rates of pairs. A prevote must be submitted for all pairs.
  * A `MsgAggregateExchangeRateVote`, containing the salt used to create the hash for the aggregate prevote submitted in the previous interval `P_t-1`.

* Vote Tally

    At the end of `P_t`, the submitted votes are tallied.

    The submitted salt of each vote is used to verify consistency with the prevote submitted by the validator in `P_t-1`. If the validator has not submitted a prevote, or the SHA256 resulting from the salt does not match the hash from the prevote, the vote is dropped.

    For each pair, if the total voting power of submitted votes exceeds 50%, the weighted median of the votes is recorded on-chain as the effective exchange rate for the following `VotePeriod` `P_t+1`.

    Exchange rates receiving fewer than `VoteThreshold` total voting power have their exchange rates deleted from the store, and no exchange rate will exist for the next VotePeriod `P_t+1`.

* Ballot Rewards

    After the votes are tallied, the winners of the ballots are determined with `tally()`.

    Voters that have managed to vote within a narrow band around the weighted median, are rewarded with a portion of the collected seigniorage. See `k.RewardBallotWinners()` for more details.

### Reward Band

Let `M` be the weighted median, `𝜎` be the standard deviation of the votes in the ballot, and  be the RewardBand parameter. The band around the median is set to be `𝜀 = max(𝜎, R/2)`. All valid (i.e. bonded and non-jailed) validators that submitted an exchange rate vote in the interval `[M - 𝜀, M + 𝜀]` should be included in the set of winners, weighted by their relative vote power.

### Slashing

> Be sure to read this section carefully as it concerns potential loss of funds.

A `VotePeriod` during which either of the following events occur is considered a "miss":

* The validator fails to submits a vote for an exchange rate against **each and every** pair specified in `Whitelist`.

* The validator fails to vote within the `reward band` around the weighted median for one or more pairs.

During every `SlashWindow`, participating validators must maintain a valid vote rate of at least `MinValidPerWindow` (5%), lest they get their stake slashed (currently set to 0.01%). The slashed validator is automatically temporarily "jailed" by the protocol (to protect the funds of delegators), and the operator is expected to fix the discrepancy promptly to resume validator participation.

### Abstaining from Voting

A validator may abstain from voting by submitting a non-positive integer for the `ExchangeRate` field in `MsgAggregateExchangeRateVote`. Doing so will absolve them of any penalties for missing `VotePeriod`s, but also disqualify them from receiving Oracle seigniorage rewards for faithful reporting.

### Messages

> The control flow for vote-tallying, exchange rate updates, ballot rewards and slashing happens at the end of every `VotePeriod`, and is found at the [end-block ABCI](#end-block) function rather than inside message handlers.

---

## Module Parameters

The oracle module contains the following parameters:

| Module Param (type)       | Description |
| ------------------------- | ----------- |
| `VotePeriod` (uint64)     | Defines the number of blocks during which voting takes place. Ex. "5". |
| `VoteThreshold` (Dec) | VoteThreshold specifies the minimum proportion of votes that must be received for a ballot to pass. Ex. "0.5" |
| `RewardBand` (Dec)    | Defines a maxium divergence that a price vote can have from the weighted median in the ballot. If a vote lies within the valid range defined by: `μ := weightedMedian`, `validRange := μ ± (μ * rewardBand / 2)`, then rewards are added to the validator performance. Note that if the reward band is smaller than 1 standard deviation, the band is taken to be 1 standard deviation.a price. Ex. "0.02" |
| `Whitelist` (set[String]) | The set of whitelisted markets, or asset pairs, for the module. Ex. '["unibi:uusd","ubtc:uusd"]' |
| `SlashFraction` (Dec) | The proportion of an oracle's stake that gets slashed in the event of slashing. `SlashFraction` specifies the exact penalty for failing a voting period. |
| `SlashWindow` (uint64)    | The number of voting periods that specify a "slash window". After each slash window, all oracles that have missed more than the penalty threshold are slashed. Missing the penalty threshold is synonymous with submitting fewer valid votes than `MinValidPerWindow`. |
| `MinValidPerWindow` (Dec)   | The oracle slashing threshold. Ex. "0.05". |
| `TwapLookbackWindow` (Duration) | Lookback window for time-weighted average price (TWAP) calculations.

---

## State

### ExchangeRate

An `sdk.Dec` that stores the current exchange rate against a given pair.

You can get the active list of pairs (exchange rates with votes past `VoteThreshold`) with `k.GetActivePairs()`.

- ExchangeRate: `0x03<pair_Bytes> -> amino(sdk.Dec)`

### FeederDelegation

An `sdk.AccAddress` (`nibi-` account) address of `operator`'s delegated price feeder.

- FeederDelegation: `0x04<valAddress_Bytes> -> amino(sdk.AccAddress)`

### MissCounter

An `int64` representing the number of `VotePeriods` that validator `operator` missed during the current `SlashWindow`.

- MissCounter: `0x05<valAddress_Bytes> -> amino(int64)`

### AggregateExchangeRatePrevote

`AggregateExchangeRatePrevote` containing validator voter's aggregated prevote for all pairs for the current `VotePeriod`.

- AggregateExchangeRatePrevote: `0x06<valAddress_Bytes> -> amino(AggregateExchangeRatePrevote)`

```go
// AggregateVoteHash is hash value to hide vote exchange rates
// which is formatted as hex string in SHA256("{salt}:({pair},{exchange_rate})|...|({pair},{exchange_rate}):{voter}")
type AggregateVoteHash []byte

type AggregateExchangeRatePrevote struct {
 Hash        AggregateVoteHash // Vote hex hash to protect centralize data source problem
 Voter       sdk.ValAddress    // Voter val address
 SubmitBlock int64
}
```

### AggregateExchangeRateVote

`AggregateExchangeRateVote` containing validator voter's aggregate vote for all pairs for the current `VotePeriod`.

- AggregateExchangeRateVote: `0x07<valAddress_Bytes> -> amino(AggregateExchangeRateVote)`

```go
type ExchangeRateTuple struct {
 Pair        string  `json:"pair"`
 ExchangeRate sdk.Dec `json:"exchange_rate"`
}

type ExchangeRateTuples []ExchangeRateTuple

type AggregateExchangeRateVote struct {
 ExchangeRateTuples ExchangeRateTuples // ExchangeRates of pairs
 Voter              sdk.ValAddress     // voter val address of validator
}
```

---

## End Block

### Tally Exchange Rate Votes

At the end of every block, the `Oracle` module checks whether it's the last block of the `VotePeriod`. If it is, it runs the [Voting Procedure](#Voting_Procedure):

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

5. Count up the validators who [missed](#Slashing) the Oracle vote and increase the appropriate miss counters

6. If at the end of a `SlashWindow`, penalize validators who have missed more than the penalty threshold (submitted fewer valid votes than `MinValidPerWindow`)

7. Distribute rewards to ballot winners with `k.RewardBallotWinners()`

8. Clear all prevotes (except ones for the next `VotePeriod`) and votes from the store

---

## Messages

### MsgAggregateExchangeRatePrevote

`Hash` is a hex string generated by the leading 20 bytes of the SHA256 hash (hex string) of a string of the format `{salt}:({pair},{exchange_rate})|...|({pair},{exchange_rate}):{voter}`, the metadata of the actual `MsgAggregateExchangeRateVote` to follow in the next `VotePeriod`. You can use the `GetAggregateVoteHash()` function to help encode this hash. Note that since in the subsequent `MsgAggregateExchangeRateVote`, the salt will have to be revealed, the salt used must be regenerated for each prevote submission.

```go
// MsgAggregateExchangeRatePrevote - struct for aggregate prevoting on the ExchangeRateVote.
// The purpose of aggregate prevote is to hide vote exchange rates with hash
// which is formatted as hex string in SHA256("{salt}:({pair},{exchange_rate})|...|({pair},{exchange_rate}):{voter}")
type MsgAggregateExchangeRatePrevote struct {
 Hash      AggregateVoteHash 
 Feeder    sdk.AccAddress    
 Validator sdk.ValAddress    
}
```

`Feeder` (`nibi-` address) is used if the validator wishes to delegate oracle vote signing to a separate key (who "feeds" the price in lieu of the operator) to de-risk exposing their validator signing key.

`Validator` is the validator address (`nibivaloper-` address) of the original validator.

### MsgAggregateExchangeRateVote

The `MsgAggregateExchangeRateVote` contains the actual exchange rates vote. The `Salt` parameter must match the salt used to create the prevote, otherwise the voter cannot be rewarded.

```go
// MsgAggregateExchangeRateVote - struct for voting on the exchange rates of pairs.
type MsgAggregateExchangeRateVote struct {
 Salt          string
 ExchangeRates string
 Feeder        sdk.AccAddress 
 Validator     sdk.ValAddress 
}
```

### MsgDelegateFeedConsent

Validators may also elect to delegate voting rights to another key to prevent the block signing key from being kept online. To do so, they must submit a `MsgDelegateFeedConsent`, delegating their oracle voting rights to a `Delegate` that sign `MsgAggregateExchangeRatePrevote` and `MsgAggregateExchangeRateVote` on behalf of the validator.

> Delegate validators will likely require you to deposit some funds (in NIBI) which they can use to pay fees, sent in a separate MsgSend. This agreement is made off-chain and not enforced by the Nibiru protocol.

The `Operator` field contains the operator address of the validator (prefixed `nibivaloper-`). The `Delegate` field is the account address (prefixed `nibi-`) of the delegate account that will be submitting exchange rate related votes and prevotes on behalf of the `Operator`.

```go
// MsgDelegateFeedConsent - struct for delegating oracle voting rights to another address.
type MsgDelegateFeedConsent struct {
 Operator sdk.ValAddress 
 Delegate sdk.AccAddress 
}
```

---

## Events

The oracle module emits the following events:

### EndBlocker

| Type                 | Attribute Key | Attribute Value |
|----------------------|---------------|-----------------|
| exchange_rate_update | pair          | {pair}          |
| exchange_rate_update | exchange_rate | {exchangeRate}  |  


### Events for MsgExchangeRatePrevote

| Type    | Attribute Key | Attribute Value     |
|---------|---------------|---------------------|
| prevote | pair          | {pair}              |
| prevote | voter         | {validatorAddress}  |
| prevote | feeder        | {feederAddress}     |
| message | module        | oracle              |
| message | action        | exchangerateprevote |
| message | sender        | {senderAddress}     |

### Events for MsgExchangeRateVote

| Type    | Attribute Key | Attribute Value    |
|---------|---------------|--------------------|
| vote    | pair          | {pair}             |
| vote    | voter         | {validatorAddress} |
| vote    | exchange_rate | {exchangeRate}     |
| vote    | feeder        | {feederAddress}    |
| message | module        | oracle             |
| message | action        | exchangeratevote   |
| message | sender        | {senderAddress}    |

### Events for MsgDelegateFeedConsent


| Type          | Attribute Key | Attribute Value    |
|---------------|---------------|--------------------|
| feed_delegate | operator      | {validatorAddress} |
| feed_delegate | feeder        | {feederAddress}    |
| message       | module        | oracle             |
| message       | action        | delegatefeeder     |
| message       | sender        | {senderAddress}    |

### Events for MsgAggregateExchangeRatePrevote

| Type              | Attribute Key | Attribute Value              |
|-------------------|---------------|------------------------------|
| aggregate_prevote | voter         | {validatorAddress}           |
| aggregate_prevote | feeder        | {feederAddress}              |
| message           | module        | oracle                       |
| message           | action        | aggregateexchangerateprevote |
| message           | sender        | {senderAddress}              |

### Events for MsgAggregateExchangeRateVote

| Type           | Attribute Key  | Attribute Value           |
|----------------|----------------|---------------------------|
| aggregate_vote | voter          | {validatorAddress}        |
| aggregate_vote | exchange_rates | {exchangeRates}           |
| aggregate_vote | feeder         | {feederAddress}           |
| message        | module         | oracle                    |
| message        | action         | aggregateexchangeratevote |
| message        | sender         | {senderAddress}           |

---
