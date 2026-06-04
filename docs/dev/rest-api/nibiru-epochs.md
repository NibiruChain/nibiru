# REST API - nibiru/epochs

## Query Service - nibiru/epochs

### /nibiru/epochs/v1beta1/current_epoch

```bash
# You can also use wget
curl -X GET https://lcd.nibiru.fi/nibiru/epochs/v1beta1/current_epoch \
  -H 'Accept: application/json'
```
##### Summary

CurrentEpoch provide current epoch of specified identifier

##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ------ |
| identifier | query |  | No | string |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [v1QueryCurrentEpochResponse](#v1querycurrentepochresponse) |
| default | An unexpected error response. | [runtimeError](#runtimeerror) |

### /nibiru/epochs/v1beta1/epochs

```bash
# You can also use wget
curl -X GET https://lcd.nibiru.fi/nibiru/epochs/v1beta1/epochs \
  -H 'Accept: application/json'
```
##### Summary

EpochInfos provide running epochInfos

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [v1QueryEpochInfosResponse](#v1queryepochinfosresponse) |
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

#### v1EpochInfo

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| identifier | string |  | No |
| start_time | dateTime | When the epoch repetitino should start. | No |
| duration | string | How long each epoch lasts for. | No |
| current_epoch | string (uint64) | The current epoch number, starting from 1. | No |
| current_epoch_start_time | dateTime | The start timestamp of the current epoch. | No |
| epoch_counting_started | boolean | Whether or not this epoch has started. Set to true if current blocktime >= start_time. | No |
| current_epoch_start_height | string (int64) | The block height at which the current epoch started at. | No |

#### v1QueryCurrentEpochResponse

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| current_epoch | string (uint64) |  | No |

#### v1QueryEpochInfosResponse

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| epochs | [ [v1EpochInfo](#v1epochinfo) ] |  | No |
