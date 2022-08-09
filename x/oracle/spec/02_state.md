<!--
order: 2
-->

# State

## ExchangeRatePrevote

`ExchangeRatePrevote` containing validator voter's prevote for a given pair for the current `VotePeriod`.

- ExchangeRatePrevote: `0x01<pair_Bytes><valAddress_Bytes> -> amino(ExchangeRatePrevote)`

```go
type ValAddress []byte
type VoteHash []byte

type ExchangeRatePrevote struct {
 Hash        VoteHash       // Vote hex hash to protect centralize data source problem
 Pair       string         // Ticker name of target fiat currency
 Voter       sdk.ValAddress // Voter val address
 SubmitBlock int64
}
```

## ExchangeRateVote

`ExchangeRateVote` containing validator voter's vote for a given pair for the current `VotePeriod`.

- ExchangeRateVote: `0x02<pair_Bytes><valAddress_Bytes> -> amino(ExchangeRateVote)`

```go
type ExchangeRateVote struct {
 ExchangeRate sdk.Dec        // ExchangeRate of pair
 Pair        string         // Ticker name of target fiat currency
 Voter        sdk.ValAddress // voter val address of validator
}
```

## ExchangeRate

An `sdk.Dec` that stores the current exchange rate against a given pair.

You can get the active list of pairs (exchange rates with votes past `VoteThreshold`) with `k.GetActivePairs()`.

- ExchangeRate: `0x03<pair_Bytes> -> amino(sdk.Dec)`

## FeederDelegation

An `sdk.AccAddress` (`nibi-` account) address of `operator`'s delegated price feeder.

- FeederDelegation: `0x04<valAddress_Bytes> -> amino(sdk.AccAddress)`

## MissCounter

An `int64` representing the number of `VotePeriods` that validator `operator` missed during the current `SlashWindow`.

- MissCounter: `0x05<valAddress_Bytes> -> amino(int64)`

## AggregateExchangeRatePrevote

`AggregateExchangeRatePrevote` containing validator voter's aggregated prevote for all pairs for the current `VotePeriod`.

- AggregateExchangeRatePrevote: `0x06<valAddress_Bytes> -> amino(AggregateExchangeRatePrevote)`

```go
// AggregateVoteHash is hash value to hide vote exchange rates
// which is formatted as hex string in SHA256("{salt}:({exchange rate},{pair})|...|({exchange rate},{pair}):{voter}")
type AggregateVoteHash []byte

type AggregateExchangeRatePrevote struct {
 Hash        AggregateVoteHash // Vote hex hash to protect centralize data source problem
 Voter       sdk.ValAddress    // Voter val address
 SubmitBlock int64
}
```

## AggregateExchangeRateVote

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