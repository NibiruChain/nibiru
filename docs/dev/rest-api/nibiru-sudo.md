# REST API - nibiru/sudo

## Query Service - nibiru/sudo

### /nibiru/sudo/sudoers

```bash
# You can also use wget
curl -X GET https://lcd.nibiru.fi/nibiru/sudo/sudoers \
  -H 'Accept: application/json'
```
##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [v1QuerySudoersResponse](#v1querysudoersresponse) |
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

#### v1QuerySudoersResponse

QuerySudoersResponse indicates the successful execution of MsgEditSudeors.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| sudoers | [v1Sudoers](#v1sudoers) |  | No |

#### v1Sudoers

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| root | string | Root: The "root" user. | No |
| contracts | [ string ] | Contracts: The set of contracts with elevated permissions. | No |

