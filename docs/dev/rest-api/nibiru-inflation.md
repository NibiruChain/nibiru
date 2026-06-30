# REST API - nibiru/inflation

## Query Service - nibiru/inflation

### /nibiru/inflation/v1/circulating_supply

```bash
# You can also use wget
curl -X GET https://lcd.nibiru.fi/nibiru/inflation/v1/circulating_supply \
  -H 'Accept: application/json'
```
##### Summary

CirculatingSupply retrieves the total number of tokens that are in
circulation (i.e. excluding unvested tokens).

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [v1QueryCirculatingSupplyResponse](#v1querycirculatingsupplyresponse) |
| default | An unexpected error response. | [runtimeError](#runtimeerror) |

### /nibiru/inflation/v1/epoch_mint_provision

```bash
# You can also use wget
curl -X GET https://lcd.nibiru.fi/nibiru/inflation/v1/epoch_mint_provision \
  -H 'Accept: application/json'
```
##### Summary

EpochMintProvision retrieves current minting epoch provision value.

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [v1QueryEpochMintProvisionResponse](#v1queryepochmintprovisionresponse) |
| default | An unexpected error response. | [runtimeError](#runtimeerror) |

### /nibiru/inflation/v1/inflation_rate

```bash
# You can also use wget
curl -X GET https://lcd.nibiru.fi/nibiru/inflation/v1/inflation_rate \
  -H 'Accept: application/json'
```
##### Summary

InflationRate retrieves the inflation rate of the current period.

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [v1QueryInflationRateResponse](#v1queryinflationrateresponse) |
| default | An unexpected error response. | [runtimeError](#runtimeerror) |

### /nibiru/inflation/v1/params

```bash
# You can also use wget
curl -X GET https://lcd.nibiru.fi/nibiru/inflation/v1/params \
  -H 'Accept: application/json'
```
##### Summary

Params retrieves the total set of minting parameters.

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [v1QueryParamsResponse](#v1queryparamsresponse) |
| default | An unexpected error response. | [runtimeError](#runtimeerror) |

### /nibiru/inflation/v1/period

```bash
# You can also use wget
curl -X GET https://lcd.nibiru.fi/nibiru/inflation/v1/period \
  -H 'Accept: application/json'
```
##### Summary

Period retrieves current period.

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [v1QueryPeriodResponse](#v1queryperiodresponse) |
| default | An unexpected error response. | [runtimeError](#runtimeerror) |

### /nibiru/inflation/v1/skipped_epochs

```bash
# You can also use wget
curl -X GET https://lcd.nibiru.fi/nibiru/inflation/v1/skipped_epochs \
  -H 'Accept: application/json'
```
##### Summary

SkippedEpochs retrieves the total number of skipped epochs.

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [v1QuerySkippedEpochsResponse](#v1queryskippedepochsresponse) |
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

#### v1InflationDistribution

InflationDistribution defines the distribution in which inflation is
allocated through minting on each epoch (staking, community, strategic). It
excludes the team vesting distribution.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| staking_rewards | string |  | No |
| community_pool | string |  | No |
| strategic_reserves | string |  | No |

#### v1Params

Params holds parameters for the inflation module.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| inflation_enabled | boolean |  | No |
| polynomial_factors | [ string ] |  | No |
| inflation_distribution | [v1InflationDistribution](#v1inflationdistribution) |  | No |
| epochs_per_period | string (uint64) |  | No |
| periods_per_year | string (uint64) |  | No |
| max_period | string (uint64) | max_period is the maximum number of periods that have inflation being  paid off. After this period, inflation will be disabled. | No |
| has_inflation_started | boolean |  | No |

#### v1QueryCirculatingSupplyResponse

QueryCirculatingSupplyResponse is the response type for the
Query/CirculatingSupply RPC method.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| circulating_supply | [v1beta1DecCoin](#v1beta1deccoin) |  | No |

#### v1QueryEpochMintProvisionResponse

QueryEpochMintProvisionResponse is the response type for the
Query/EpochMintProvision RPC method.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| epoch_mint_provision | [v1beta1DecCoin](#v1beta1deccoin) | epoch_mint_provision is the current minting per epoch provision value. | No |

#### v1QueryInflationRateResponse

QueryInflationRateResponse is the response type for the Query/InflationRate
RPC method.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| inflation_rate | string |  | No |

#### v1QueryParamsResponse

QueryParamsResponse is the response type for the Query/Params RPC method.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| params | [v1Params](#v1params) | params defines the parameters of the module. | No |

#### v1QueryPeriodResponse

QueryPeriodResponse is the response type for the Query/Period RPC method.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| period | string (uint64) | period is the current minting per epoch provision value. | No |

#### v1QuerySkippedEpochsResponse

QuerySkippedEpochsResponse is the response type for the Query/SkippedEpochs
RPC method.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| skipped_epochs | string (uint64) | skipped_epochs is the number of epochs that the inflation module has been disabled. | No |

#### v1beta1DecCoin

DecCoin defines a token with a denomination and a decimal amount.

NOTE: The amount field is an Dec which implements the custom method
signatures required by gogoproto.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| denom | string |  | No |
| amount | string |  | No |
