# REST API - nibiru/devgas

## Query Service - nibiru/devgas

### /nibiru/devgas/v1/fee_shares/{contract_address}

```bash
# You can also use wget
curl -X GET https://lcd.nibiru.fi/nibiru/devgas/v1/fee_shares/{contract_address} \
  -H 'Accept: application/json'
```
##### Summary

FeeShare retrieves a registered FeeShare for a given contract address

##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ------ |
| contract_address | path | contract_address of a registered contract in bech32 format | Yes | string |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [v1QueryFeeShareResponse](#v1queryfeeshareresponse) |
| default | An unexpected error response. | [runtimeError](#runtimeerror) |

### /nibiru/devgas/v1/fee_shares/{deployer}

```bash
# You can also use wget
curl -X GET https://lcd.nibiru.fi/nibiru/devgas/v1/fee_shares/{deployer} \
  -H 'Accept: application/json'
```
##### Summary

FeeShares retrieves all FeeShares that a deployer has
registered

##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ------ |
| deployer | path | TODO feat(devgas): re-implement the paginated version TODO feat(colletions): add automatic pagination generation | Yes | string |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [v1QueryFeeSharesResponse](#v1queryfeesharesresponse) |
| default | An unexpected error response. | [runtimeError](#runtimeerror) |

### /nibiru/devgas/v1/fee_shares/{withdrawer_address}

```bash
# You can also use wget
curl -X GET https://lcd.nibiru.fi/nibiru/devgas/v1/fee_shares/{withdrawer_address} \
  -H 'Accept: application/json'
```
##### Summary

FeeSharesByWithdrawer retrieves all FeeShares with a given withdrawer
address

##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ------ |
| withdrawer_address | path | withdrawer_address in bech32 format | Yes | string |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [v1QueryFeeSharesByWithdrawerResponse](#v1queryfeesharesbywithdrawerresponse) |
| default | An unexpected error response. | [runtimeError](#runtimeerror) |

### /nibiru/devgas/v1/params

```bash
# You can also use wget
curl -X GET https://lcd.nibiru.fi/nibiru/devgas/v1/params \
  -H 'Accept: application/json'
```
##### Summary

Params retrieves the module params

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [v1QueryParamsResponse](#v1queryparamsresponse) |
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

#### v1FeeShare

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| contract_address | string |  | No |
| deployer_address | string | deployer_address is the bech32 address of message sender. It must be the same as the contracts admin address. | No |
| withdrawer_address | string | withdrawer_address is the bech32 address of account receiving the transaction fees. | No |

#### v1ModuleParams

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| enable_fee_share | boolean |  | No |
| developer_shares | string |  | No |
| allowed_denoms | [ string ] | allowed_denoms defines the list of denoms that are allowed to be paid to the contract withdraw addresses. If said denom is not in the list, the fees will ONLY be sent to the community pool. If this list is empty, all denoms are allowed. | No |

#### v1QueryFeeShareResponse

QueryFeeShareResponse is the response type for the Query/FeeShare RPC method.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| feeshare | [v1FeeShare](#v1feeshare) |  | No |

#### v1QueryFeeSharesByWithdrawerResponse

QueryFeeSharesByWithdrawerResponse is the response type for the
Query/FeeSharesByWithdrawer RPC method.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| feeshare | [ [v1FeeShare](#v1feeshare) ] |  | No |

#### v1QueryFeeSharesResponse

QueryFeeSharesResponse is the response type for the Query/FeeShares RPC
method.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| feeshare | [ [v1FeeShare](#v1feeshare) ] |  | No |

#### v1QueryParamsResponse

QueryParamsResponse is the response type for the Query/Params RPC method.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| params | [v1ModuleParams](#v1moduleparams) |  | No |

