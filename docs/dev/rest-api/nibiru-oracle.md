# REST API - nibiru/oracle

## Query Service - nibiru/oracle

### /nibiru/oracle/v1beta1/exchange_rate

```bash
# You can also use wget
curl -X GET https://lcd.nibiru.fi/nibiru/oracle/v1beta1/exchange_rate \
  -H 'Accept: application/json'
```
##### Summary

ExchangeRate returns exchange rate of a pair along with the block height and
block time that the exchange rate was set by the oracle module.

##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ------ |
| pair | query | pair defines the pair to query for. | No | string |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [v1QueryExchangeRateResponse](#v1queryexchangerateresponse) |
| default | An unexpected error response. | [runtimeError](#runtimeerror) |

### /nibiru/oracle/v1beta1/exchange_rate_twap

```bash
# You can also use wget
curl -X GET https://lcd.nibiru.fi/nibiru/oracle/v1beta1/exchange_rate_twap \
  -H 'Accept: application/json'
```
##### Summary

ExchangeRateTwap returns twap exchange rate of a pair

##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ------ |
| pair | query | pair defines the pair to query for. | No | string |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [v1QueryExchangeRateResponse](#v1queryexchangerateresponse) |
| default | An unexpected error response. | [runtimeError](#runtimeerror) |

### /nibiru/oracle/v1beta1/pairs/actives

```bash
# You can also use wget
curl -X GET https://lcd.nibiru.fi/nibiru/oracle/v1beta1/pairs/actives \
  -H 'Accept: application/json'
```
##### Summary

Actives returns all active pairs

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [v1QueryActivesResponse](#v1queryactivesresponse) |
| default | An unexpected error response. | [runtimeError](#runtimeerror) |

### /nibiru/oracle/v1beta1/pairs/exchange_rates

```bash
# You can also use wget
curl -X GET https://lcd.nibiru.fi/nibiru/oracle/v1beta1/pairs/exchange_rates \
  -H 'Accept: application/json'
```
##### Summary

ExchangeRates returns exchange rates of all pairs

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [v1QueryExchangeRatesResponse](#v1queryexchangeratesresponse) |
| default | An unexpected error response. | [runtimeError](#runtimeerror) |

### /nibiru/oracle/v1beta1/pairs/vote_targets

```bash
# You can also use wget
curl -X GET https://lcd.nibiru.fi/nibiru/oracle/v1beta1/pairs/vote_targets \
  -H 'Accept: application/json'
```
##### Summary

VoteTargets returns all vote target for pairs

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [v1QueryVoteTargetsResponse](#v1queryvotetargetsresponse) |
| default | An unexpected error response. | [runtimeError](#runtimeerror) |

### /nibiru/oracle/v1beta1/params

```bash
# You can also use wget
curl -X GET https://lcd.nibiru.fi/nibiru/oracle/v1beta1/params \
  -H 'Accept: application/json'
```
##### Summary

Params queries all parameters.

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [v1QueryParamsResponse](#v1queryparamsresponse) |
| default | An unexpected error response. | [runtimeError](#runtimeerror) |

### /nibiru/oracle/v1beta1/valdiators/{validator_addr}/aggregate_vote

```bash
# You can also use wget
curl -X GET https://lcd.nibiru.fi/nibiru/oracle/v1beta1/valdiators/{validator_addr}/aggregate_vote \
  -H 'Accept: application/json'
```
##### Summary

AggregateVote returns an aggregate vote of a validator

##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ------ |
| validator_addr | path | validator defines the validator address to query for. | Yes | string |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [v1QueryAggregateVoteResponse](#v1queryaggregatevoteresponse) |
| default | An unexpected error response. | [runtimeError](#runtimeerror) |

### /nibiru/oracle/v1beta1/validators/aggregate_prevotes

```bash
# You can also use wget
curl -X GET https://lcd.nibiru.fi/nibiru/oracle/v1beta1/validators/aggregate_prevotes \
  -H 'Accept: application/json'
```
##### Summary

AggregatePrevotes returns aggregate prevotes of all validators

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [v1QueryAggregatePrevotesResponse](#v1queryaggregateprevotesresponse) |
| default | An unexpected error response. | [runtimeError](#runtimeerror) |

### /nibiru/oracle/v1beta1/validators/aggregate_votes

```bash
# You can also use wget
curl -X GET https://lcd.nibiru.fi/nibiru/oracle/v1beta1/validators/aggregate_votes \
  -H 'Accept: application/json'
```
##### Summary

AggregateVotes returns aggregate votes of all validators

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [v1QueryAggregateVotesResponse](#v1queryaggregatevotesresponse) |
| default | An unexpected error response. | [runtimeError](#runtimeerror) |

### /nibiru/oracle/v1beta1/validators/{validator_addr}/aggregate_prevote

```bash
# You can also use wget
curl -X GET https://lcd.nibiru.fi/nibiru/oracle/v1beta1/validators/{validator_addr}/aggregate_prevote \
  -H 'Accept: application/json'
```
##### Summary

AggregatePrevote returns an aggregate prevote of a validator

##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ------ |
| validator_addr | path | validator defines the validator address to query for. | Yes | string |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [v1QueryAggregatePrevoteResponse](#v1queryaggregateprevoteresponse) |
| default | An unexpected error response. | [runtimeError](#runtimeerror) |

### /nibiru/oracle/v1beta1/validators/{validator_addr}/feeder

```bash
# You can also use wget
curl -X GET https://lcd.nibiru.fi/nibiru/oracle/v1beta1/validators/{validator_addr}/feeder \
  -H 'Accept: application/json'
```
##### Summary

FeederDelegation returns feeder delegation of a validator

##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ------ |
| validator_addr | path | validator defines the validator address to query for. | Yes | string |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [v1QueryFeederDelegationResponse](#v1queryfeederdelegationresponse) |
| default | An unexpected error response. | [runtimeError](#runtimeerror) |

### /nibiru/oracle/v1beta1/validators/{validator_addr}/miss

```bash
# You can also use wget
curl -X GET https://lcd.nibiru.fi/nibiru/oracle/v1beta1/validators/{validator_addr}/miss \
  -H 'Accept: application/json'
```
##### Summary

MissCounter returns oracle miss counter of a validator

##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ------ |
| validator_addr | path | validator defines the validator address to query for. | Yes | string |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [v1QueryMissCounterResponse](#v1querymisscounterresponse) |
| default | An unexpected error response. | [runtimeError](#runtimeerror) |

---
### Models

#### protobufAny

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| type_url | string |  | No |
| value | byte |  | No |

#### runtimeError

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| error | string |  | No |
| code | integer |  | No |
| message | string |  | No |
| details | [ [protobufAny](#protobufany) ] |  | No |

#### v1AggregateExchangeRatePrevote

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| hash | string |  | No |
| voter | string |  | No |
| submit_block | string (uint64) |  | No |

#### v1AggregateExchangeRateVote

MsgAggregateExchangeRateVote - struct for voting on
the exchange rates different assets.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| exchange_rate_tuples | [ [v1ExchangeRateTuple](#v1exchangeratetuple) ] |  | No |
| voter | string |  | No |

#### v1ExchangeRateTuple

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| pair | string |  | No |
| exchange_rate | string |  | No |

#### v1Params

Params defines the module parameters for the x/oracle module.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| vote_period | string (uint64) | VotePeriod defines the number of blocks during which voting takes place. | No |
| vote_threshold | string | VoteThreshold specifies the minimum proportion of votes that must be received for a ballot to pass. | No |
| reward_band | string |  | No |
| whitelist | [ string ] |  | No |
| slash_fraction | string | SlashFraction returns the proportion of an oracle's stake that gets slashed in the event of slashing. `SlashFraction` specifies the exact penalty for failing a voting period. | No |
| slash_window | string (uint64) | SlashWindow returns the number of voting periods that specify a "slash window". After each slash window, all oracles that have missed more than the penalty threshold are slashed. Missing the penalty threshold is synonymous with submitting fewer valid votes than `MinValidPerWindow`. | No |
| min_valid_per_window | string |  | No |
| twap_lookback_window | string | Amount of time to look back for TWAP calculations. Ex: "900.000000069s" corresponds to 900 seconds and 69 nanoseconds in JSON. | No |
| min_voters | string (uint64) | The minimum number of voters (i.e. oracle validators) per pair for it to be considered a passing ballot. Recommended at least 4. | No |
| validator_fee_ratio | string | The validator fee ratio that is given to validators every epoch. | No |
| expiration_blocks | string (uint64) |  | No |

#### v1QueryActivesResponse

QueryActivesResponse is response type for the
Query/Actives RPC method.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| actives | [ string ] | actives defines a list of the pair which oracle prices agreed upon. | No |

#### v1QueryAggregatePrevoteResponse

QueryAggregatePrevoteResponse is response type for the
Query/AggregatePrevote RPC method.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| aggregate_prevote | [v1AggregateExchangeRatePrevote](#v1aggregateexchangerateprevote) |  | No |

#### v1QueryAggregatePrevotesResponse

QueryAggregatePrevotesResponse is response type for the
Query/AggregatePrevotes RPC method.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| aggregate_prevotes | [ [v1AggregateExchangeRatePrevote](#v1aggregateexchangerateprevote) ] |  | No |

#### v1QueryAggregateVoteResponse

QueryAggregateVoteResponse is response type for the
Query/AggregateVote RPC method.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| aggregate_vote | [v1AggregateExchangeRateVote](#v1aggregateexchangeratevote) |  | No |

#### v1QueryAggregateVotesResponse

QueryAggregateVotesResponse is response type for the
Query/AggregateVotes RPC method.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| aggregate_votes | [ [v1AggregateExchangeRateVote](#v1aggregateexchangeratevote) ] |  | No |

#### v1QueryExchangeRateResponse

QueryExchangeRateResponse is response type for the
Query/ExchangeRate RPC method.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| exchange_rate | string |  | No |
| block_timestamp_ms | string (int64) | Block timestamp for the block where the oracle came to consensus for this price. This timestamp is a conventional Unix millisecond time, i.e. the number of milliseconds elapsed since January 1, 1970 UTC. | No |
| block_height | string (uint64) | Block height when the oracle came to consensus for this price. | No |
| is_vintage | boolean | True if this exchange rate has passed its expiration window. | No |

#### v1QueryExchangeRatesResponse

QueryExchangeRatesResponse is response type for the
Query/ExchangeRates RPC method.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| exchange_rates | [ [v1ExchangeRateTuple](#v1exchangeratetuple) ] | exchange_rates defines a list of the exchange rate for all whitelisted pairs. | No |

#### v1QueryFeederDelegationResponse

QueryFeederDelegationResponse is response type for the
Query/FeederDelegation RPC method.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| feeder_addr | string |  | No |

#### v1QueryMissCounterResponse

QueryMissCounterResponse is response type for the
Query/MissCounter RPC method.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| miss_counter | string (uint64) |  | No |

#### v1QueryParamsResponse

QueryParamsResponse is the response type for the Query/Params RPC method.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| params | [v1Params](#v1params) | params defines the parameters of the module. | No |

#### v1QueryVoteTargetsResponse

QueryVoteTargetsResponse is response type for the
Query/VoteTargets RPC method.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| vote_targets | [ string ] | vote_targets defines a list of the pairs in which everyone should vote in the current vote period. | No |

