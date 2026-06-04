# REST API - nibiru/tokenfactory

## Query Service - nibiru/tokenfactory

### /nibiru/tokenfactory/v1/denom-info/{denom}

```bash
# You can also use wget
curl -X GET https://lcd.nibiru.fi/nibiru/tokenfactory/v1/denom-info/{denom} \
  -H 'Accept: application/json'
```
##### Summary

DenomInfo retrieves the denom metadata and admin info

##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ------ |
| denom | path |  | Yes | string |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [v1QueryDenomInfoResponse](#v1querydenominforesponse) |
| default | An unexpected error response. | [runtimeError](#runtimeerror) |

### /nibiru/tokenfactory/v1/denoms/{creator}

```bash
# You can also use wget
curl -X GET https://lcd.nibiru.fi/nibiru/tokenfactory/v1/denoms/{creator} \
  -H 'Accept: application/json'
```
##### Summary

Denoms retrieves all registered denoms for a given creator

##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ------ |
| creator | path |  | Yes | string |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [v1QueryDenomsResponse](#v1querydenomsresponse) |
| default | An unexpected error response. | [runtimeError](#runtimeerror) |

### /nibiru/tokenfactory/v1/params

```bash
# You can also use wget
curl -X GET https://lcd.nibiru.fi/nibiru/tokenfactory/v1/params \
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

#### v1ModuleParams

ModuleParams defines the parameters for the tokenfactory module.

### On Denom Creation Costs

We'd like for fees to be paid by the user/signer of a ransaction, but in many
casess, token creation is abstracted away behind a smart contract. Setting a
nonzero `denom_creation_fee` would force each contract to handle collecting
and paying a fees for denom (factory/{contract-addr}/{subdenom}) creation on
behalf of the end user.

For IBC token transfers, it's unclear who should pay the feeΓÇöthe contract,
the relayer, or the original sender?
> "Charging fees will mess up composability, the same way Terra transfer tax
  caused all kinds of headaches for contract devs." - @ethanfrey

### Recommended Solution

Have the end user (signer) pay fees directly in the form of higher gas costs.
This way, contracts won't need to handle collecting or paying fees. And for
IBC, the gas costs are already paid by the original sender and can be
estimated by the relayer. It's easier to tune gas costs to make spam
prohibitively expensive since there are per-transaction and per-block gas
limits.

See https://github.com/CosmWasm/token-factory/issues/11 for the initial
discussion of the issue with @ethanfrey and @valardragon.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| denom_creation_gas_consume | string (uint64) | Adds gas consumption to the execution of `MsgCreateDenom` as a method of spam prevention. Defaults to 10 NIBI. | No |

#### v1QueryDenomInfoResponse

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| admin | string |  | No |
| metadata | [v1beta1Metadata](#v1beta1metadata) | Metadata: Official x/bank metadata for the denom. All token factory denoms are standard, native assets. | No |

#### v1QueryDenomsResponse

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| denoms | [ string ] |  | No |

#### v1QueryParamsResponse

QueryParamsResponse is the response type for the Query/Params RPC method.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| params | [v1ModuleParams](#v1moduleparams) |  | No |

#### v1beta1DenomUnit

DenomUnit represents a struct that describes a given
denomination unit of the basic token.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| denom | string | denom represents the string name of the given denom unit (e.g uatom). | No |
| exponent | long | exponent represents power of 10 exponent that one must raise the base_denom to in order to equal the given DenomUnit's denom 1 denom = 10^exponent base_denom (e.g. with a base_denom of uatom, one can create a DenomUnit of 'atom' with exponent = 6, thus: 1 atom = 10^6 uatom). | No |
| aliases | [ string ] |  | No |

#### v1beta1Metadata

Metadata represents a struct that describes
a basic token.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| description | string |  | No |
| denom_units | [ [v1beta1DenomUnit](#v1beta1denomunit) ] |  | No |
| base | string | base represents the base denom (should be the DenomUnit with exponent = 0). | No |
| display | string | display indicates the suggested denom that should be displayed in clients. | No |
| name | string | Since: cosmos-sdk 0.43 | No |
| symbol | string | symbol is the token symbol usually shown on exchanges (eg: ATOM). This can be the same as the display.  Since: cosmos-sdk 0.43 | No |
| uri | string | URI to a document (on or off-chain) that contains additional information. Optional.  Since: cosmos-sdk 0.46 | No |
| uri_hash | string | URIHash is a sha256 hash of a document pointed by URI. It's used to verify that the document didn't change. Optional.  Since: cosmos-sdk 0.46 | No |

