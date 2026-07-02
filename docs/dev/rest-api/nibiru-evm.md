# REST API - nibiru/evm

## Query Service - nibiru/evm

### /nibiru/evm/v1/balances/{address}

```bash
# You can also use wget
curl -X GET 'https://lcd.nibiru.fi/nibiru/evm/v1/balances/{address}?token={token}' \
  -H 'Accept: application/json'
```
##### Summary

Balance queries the native EVM balance for a single account. When `token` is
provided, the response also includes Bank and/or ERC20 balance details for that
token when available.

##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ------ |
| address | path | address is the ethereum hex address to query the balance for. | Yes | string |
| token | query | token is an ERC20 address or bank denom to query alongside the native EVM balance. Leave empty to query only the native EVM balance. | No | string |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [v1QueryBalanceResponse](#v1querybalanceresponse) |
| default | An unexpected error response. | [runtimeError](#runtimeerror) |

### /nibiru/evm/v1/base_fee

```bash
# You can also use wget
curl -X GET https://lcd.nibiru.fi/nibiru/evm/v1/base_fee \
  -H 'Accept: application/json'
```
##### Summary

BaseFee queries the base fee of the parent block of the current block,
Similar to feemarket module's method

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [v1QueryBaseFeeResponse](#v1querybasefeeresponse) |
| default | An unexpected error response. | [runtimeError](#runtimeerror) |

### /nibiru/evm/v1/codes/{address}

```bash
# You can also use wget
curl -X GET https://lcd.nibiru.fi/nibiru/evm/v1/codes/{address} \
  -H 'Accept: application/json'
```
##### Summary

Code queries the balance of all coins for a single account.

##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ------ |
| address | path | address is the ethereum hex address to query the code for. | Yes | string |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [v1QueryCodeResponse](#v1querycoderesponse) |
| default | An unexpected error response. | [runtimeError](#runtimeerror) |

### /nibiru/evm/v1/estimate_gas

```bash
# You can also use wget
curl -X GET https://lcd.nibiru.fi/nibiru/evm/v1/estimate_gas \
  -H 'Accept: application/json'
```
##### Summary

EstimateGas implements the `eth_estimateGas` rpc api

##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ------ |
| args | query | args uses the same json format as the json rpc api. | No | byte |
| gas_cap | query | gas_cap defines the default gas cap to be used. | No | string (uint64) |
| proposer_address | query | proposer_address of the requested block in hex format. | No | byte |
| chain_id | query | chain_id is the eip155 chain id parsed from the requested block header. | No | string (int64) |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [v1EstimateGasResponse](#v1estimategasresponse) |
| default | An unexpected error response. | [runtimeError](#runtimeerror) |

### /nibiru/evm/v1/eth_account/{address}

```bash
# You can also use wget
curl -X GET https://lcd.nibiru.fi/nibiru/evm/v1/eth_account/{address} \
  -H 'Accept: application/json'
```
##### Summary

EthAccount queries a Nibiru account using its EVM address or Bech32 Nibiru
address.

##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ------ |
| address | path | address is the Ethereum hex address or nibi Bech32 address to query the account for. | Yes | string |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [v1QueryEthAccountResponse](#v1queryethaccountresponse) |
| default | An unexpected error response. | [runtimeError](#runtimeerror) |

### /nibiru/evm/v1/eth_call

```bash
# You can also use wget
curl -X GET https://lcd.nibiru.fi/nibiru/evm/v1/eth_call \
  -H 'Accept: application/json'
```
##### Summary

EthCall implements the `eth_call` rpc api

##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ------ |
| args | query | args uses the same json format as the json rpc api. | No | byte |
| gas_cap | query | gas_cap defines the default gas cap to be used. | No | string (uint64) |
| proposer_address | query | proposer_address of the requested block in hex format. | No | byte |
| chain_id | query | chain_id is the eip155 chain id parsed from the requested block header. | No | string (int64) |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [v1MsgEthereumTxResponse](#v1msgethereumtxresponse) |
| default | An unexpected error response. | [runtimeError](#runtimeerror) |

### /nibiru/evm/v1/funtoken/{token}

```bash
# You can also use wget
curl -X GET https://lcd.nibiru.fi/nibiru/evm/v1/funtoken/{token} \
  -H 'Accept: application/json'
```
##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ------ |
| token | path | Either the hexadecimal-encoded ERC20 contract address or denomination of the Bank Coin. | Yes | string |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [v1QueryFunTokenMappingResponse](#v1queryfuntokenmappingresponse) |
| default | An unexpected error response. | [runtimeError](#runtimeerror) |

### /nibiru/evm/v1/params

```bash
# You can also use wget
curl -X GET https://lcd.nibiru.fi/nibiru/evm/v1/params \
  -H 'Accept: application/json'
```
##### Summary

Params queries the parameters of x/evm module.

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [v1QueryParamsResponse](#v1queryparamsresponse) |
| default | An unexpected error response. | [runtimeError](#runtimeerror) |

### /nibiru/evm/v1/storage/{address}/{key}

```bash
# You can also use wget
curl -X GET https://lcd.nibiru.fi/nibiru/evm/v1/storage/{address}/{key} \
  -H 'Accept: application/json'
```
##### Summary

Storage queries the balance of all coins for a single account.

##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ------ |
| address | path | address is the ethereum hex address to query the storage state for. | Yes | string |
| key | path | key defines the key of the storage state | Yes | string |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [v1QueryStorageResponse](#v1querystorageresponse) |
| default | An unexpected error response. | [runtimeError](#runtimeerror) |

### /nibiru/evm/v1/trace_block

```bash
# You can also use wget
curl -X GET https://lcd.nibiru.fi/nibiru/evm/v1/trace_block \
  -H 'Accept: application/json'
```
##### Summary

TraceBlock implements the `debug_traceBlockByNumber` and `debug_traceBlockByHash` rpc api

##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ------ |
| trace_config.tracer | query | tracer is a custom javascript tracer. | No | string |
| trace_config.timeout | query | timeout overrides the default timeout of 5 seconds for JavaScript-based tracing calls. | No | string |
| trace_config.reexec | query | reexec defines the number of blocks the tracer is willing to go back. | No | string (uint64) |
| trace_config.disable_stack | query | disable_stack switches stack capture. | No | boolean |
| trace_config.disable_storage | query | disable_storage switches storage capture. | No | boolean |
| trace_config.debug | query | debug can be used to print output during capture end. | No | boolean |
| trace_config.limit | query | limit defines the maximum length of output, but zero means unlimited. | No | integer |
| trace_config.enable_memory | query | enable_memory switches memory capture. | No | boolean |
| trace_config.enable_return_data | query | enable_return_data switches the capture of return data. | No | boolean |
| trace_config.tracer_config.only_top_call | query |  | No | boolean |
| block_number | query | block_number of the traced block. | No | string (int64) |
| block_hash | query | block_hash (hex) of the traced block. | No | string |
| block_time | query | block_time of the traced block. | No | dateTime |
| proposer_address | query | proposer_address is the address of the requested block. | No | byte |
| chain_id | query | chain_id is the eip155 chain id parsed from the requested block header. | No | string (int64) |
| block_max_gas | query | block_max_gas of the traced block. | No | string (int64) |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [v1QueryTraceBlockResponse](#v1querytraceblockresponse) |
| default | An unexpected error response. | [runtimeError](#runtimeerror) |

### /nibiru/evm/v1/trace_call

```bash
# You can also use wget
curl -X GET https://lcd.nibiru.fi/nibiru/evm/v1/trace_call \
  -H 'Accept: application/json'
```
##### Summary

TraceCall implements the `debug_traceCall` rpc api

##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ------ |
| msg.data.type_url | query | A URL/resource name that uniquely identifies the type of the serialized protocol buffer message. This string must contain at least one "/" character. The last segment of the URL's path must represent the fully qualified name of the type (as in `path/google.protobuf.Duration`). The name should be in a canonical form (e.g., leading "." is not accepted).  In practice, teams usually precompile into the binary all types that they expect it to use in the context of Any. However, for URLs which use the scheme `http`, `https`, or no scheme, one can optionally set up a type server that maps type URLs to message definitions as follows:  * If no scheme is provided, `https` is assumed. * An HTTP GET on the URL must yield a [google.protobuf.Type][]   value in binary format, or produce an error. * Applications are allowed to cache lookup results based on the   URL, or have them precompiled into a binary to avoid any   lookup. Therefore, binary compatibility needs to be preserved   on changes to types. (Use versioned type names to manage   breaking changes.)  Note: this functionality is not currently available in the official protobuf release, and it is not used for type URLs beginning with type.googleapis.com. As of May 2023, there are no widely used type server implementations and no plans to implement one.  Schemes other than `http`, `https` (or the empty scheme) might be used with implementation specific semantics. | No | string |
| msg.data.value | query | Must be a valid serialized protocol buffer of the above specified type. | No | byte |
| msg.size | query | size is the encoded storage size of the transaction (DEPRECATED). | No | double |
| msg.hash | query | hash of the transaction in hex format. | No | string |
| msg.from | query | from is the ethereum signer address in hex format. This address value is checked against the address derived from the signature (V, R, S) using the secp256k1 elliptic curve. | No | string |
| trace_config.tracer | query | tracer is a custom javascript tracer. | No | string |
| trace_config.timeout | query | timeout overrides the default timeout of 5 seconds for JavaScript-based tracing calls. | No | string |
| trace_config.reexec | query | reexec defines the number of blocks the tracer is willing to go back. | No | string (uint64) |
| trace_config.disable_stack | query | disable_stack switches stack capture. | No | boolean |
| trace_config.disable_storage | query | disable_storage switches storage capture. | No | boolean |
| trace_config.debug | query | debug can be used to print output during capture end. | No | boolean |
| trace_config.limit | query | limit defines the maximum length of output, but zero means unlimited. | No | integer |
| trace_config.enable_memory | query | enable_memory switches memory capture. | No | boolean |
| trace_config.enable_return_data | query | enable_return_data switches the capture of return data. | No | boolean |
| trace_config.tracer_config.only_top_call | query |  | No | boolean |
| block_number | query | block_number of requested transaction. | No | string (int64) |
| block_hash | query | block_hash of requested transaction. | No | string |
| block_time | query | block_time of requested transaction. | No | dateTime |
| proposer_address | query | proposer_address is the proposer of the requested block. | No | byte |
| chain_id | query | chain_id is the the eip155 chain id parsed from the requested block header. | No | string (int64) |
| block_max_gas | query | block_max_gas of the block of the requested transaction. | No | string (int64) |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [v1QueryTraceTxResponse](#v1querytracetxresponse) |
| default | An unexpected error response. | [runtimeError](#runtimeerror) |

### /nibiru/evm/v1/trace_tx

```bash
# You can also use wget
curl -X GET https://lcd.nibiru.fi/nibiru/evm/v1/trace_tx \
  -H 'Accept: application/json'
```
##### Summary

TraceTx implements the `debug_traceTransaction` rpc api

##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ------ |
| msg.data.type_url | query | A URL/resource name that uniquely identifies the type of the serialized protocol buffer message. This string must contain at least one "/" character. The last segment of the URL's path must represent the fully qualified name of the type (as in `path/google.protobuf.Duration`). The name should be in a canonical form (e.g., leading "." is not accepted).  In practice, teams usually precompile into the binary all types that they expect it to use in the context of Any. However, for URLs which use the scheme `http`, `https`, or no scheme, one can optionally set up a type server that maps type URLs to message definitions as follows:  * If no scheme is provided, `https` is assumed. * An HTTP GET on the URL must yield a [google.protobuf.Type][]   value in binary format, or produce an error. * Applications are allowed to cache lookup results based on the   URL, or have them precompiled into a binary to avoid any   lookup. Therefore, binary compatibility needs to be preserved   on changes to types. (Use versioned type names to manage   breaking changes.)  Note: this functionality is not currently available in the official protobuf release, and it is not used for type URLs beginning with type.googleapis.com. As of May 2023, there are no widely used type server implementations and no plans to implement one.  Schemes other than `http`, `https` (or the empty scheme) might be used with implementation specific semantics. | No | string |
| msg.data.value | query | Must be a valid serialized protocol buffer of the above specified type. | No | byte |
| msg.size | query | size is the encoded storage size of the transaction (DEPRECATED). | No | double |
| msg.hash | query | hash of the transaction in hex format. | No | string |
| msg.from | query | from is the ethereum signer address in hex format. This address value is checked against the address derived from the signature (V, R, S) using the secp256k1 elliptic curve. | No | string |
| trace_config.tracer | query | tracer is a custom javascript tracer. | No | string |
| trace_config.timeout | query | timeout overrides the default timeout of 5 seconds for JavaScript-based tracing calls. | No | string |
| trace_config.reexec | query | reexec defines the number of blocks the tracer is willing to go back. | No | string (uint64) |
| trace_config.disable_stack | query | disable_stack switches stack capture. | No | boolean |
| trace_config.disable_storage | query | disable_storage switches storage capture. | No | boolean |
| trace_config.debug | query | debug can be used to print output during capture end. | No | boolean |
| trace_config.limit | query | limit defines the maximum length of output, but zero means unlimited. | No | integer |
| trace_config.enable_memory | query | enable_memory switches memory capture. | No | boolean |
| trace_config.enable_return_data | query | enable_return_data switches the capture of return data. | No | boolean |
| trace_config.tracer_config.only_top_call | query |  | No | boolean |
| block_number | query | block_number of requested transaction. | No | string (int64) |
| block_hash | query | block_hash of requested transaction. | No | string |
| block_time | query | block_time of requested transaction. | No | dateTime |
| proposer_address | query | proposer_address is the proposer of the requested block. | No | byte |
| chain_id | query | chain_id is the the eip155 chain id parsed from the requested block header. | No | string (int64) |
| block_max_gas | query | block_max_gas of the block of the requested transaction. | No | string (int64) |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [v1QueryTraceTxResponse](#v1querytracetxresponse) |
| default | An unexpected error response. | [runtimeError](#runtimeerror) |

### /nibiru/evm/v1/validator_account/{cons_address}

```bash
# You can also use wget
curl -X GET https://lcd.nibiru.fi/nibiru/evm/v1/validator_account/{cons_address} \
  -H 'Accept: application/json'
```
##### Summary

ValidatorAccount queries an Ethereum account's from a validator consensus
Address.

##### Parameters

| Name | Located in | Description | Required | Schema |
| ---- | ---------- | ----------- | -------- | ------ |
| cons_address | path | cons_address is the validator cons address to query the account for. | Yes | string |

##### Responses

| Code | Description | Schema |
| ---- | ----------- | ------ |
| 200 | A successful response. | [v1QueryValidatorAccountResponse](#v1queryvalidatoraccountresponse) |
| default | An unexpected error response. | [runtimeError](#runtimeerror) |

---
## Models

#### protobufAny

`Any` contains an arbitrary serialized protocol buffer message along with a
URL that describes the type of the serialized message.

Protobuf library provides support to pack/unpack Any values in the form
of utility functions or additional generated methods of the Any type.

Example 1: Pack and unpack a message in C++.

    Foo foo = ...;
    Any any;
    any.PackFrom(foo);
    ...
    if (any.UnpackTo(&foo)) {
      ...
    }

Example 2: Pack and unpack a message in Java.

    Foo foo = ...;
    Any any = Any.pack(foo);
    ...
    if (any.is(Foo.class)) {
      foo = any.unpack(Foo.class);
    }
    // or ...
    if (any.isSameTypeAs(Foo.getDefaultInstance())) {
      foo = any.unpack(Foo.getDefaultInstance());
    }

 Example 3: Pack and unpack a message in Python.

    foo = Foo(...)
    any = Any()
    any.Pack(foo)
    ...
    if any.Is(Foo.DESCRIPTOR):
      any.Unpack(foo)
      ...

 Example 4: Pack and unpack a message in Go

     foo := &pb.Foo{...}
     any, err := anypb.New(foo)
     if err != nil {
       ...
     }
     ...
     foo := &pb.Foo{}
     if err := any.UnmarshalTo(foo); err != nil {
       ...
     }

The pack methods provided by protobuf library will by default use
'type.googleapis.com/full.type.name' as the type URL and the unpack
methods only use the fully qualified type name after the last '/'
in the type URL, for example "foo.bar.com/x/y.z" will yield type
name "y.z".

## JSON
The JSON representation of an `Any` value uses the regular
representation of the deserialized, embedded message, with an
additional field `@type` which contains the type URL. Example:

    package google.profile;
    message Person {
      string first_name = 1;
      string last_name = 2;
    }

    {
      "@type": "type.googleapis.com/google.profile.Person",
      "firstName": <string>,
      "lastName": <string>
    }

If the embedded message type is well-known and has a custom JSON
representation, that representation will be embedded adding a field
`value` which holds the custom JSON in addition to the `@type`
field. Example (for message [google.protobuf.Duration][]):

    {
      "@type": "type.googleapis.com/google.protobuf.Duration",
      "value": "1.212s"
    }

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| type_url | string | A URL/resource name that uniquely identifies the type of the serialized protocol buffer message. This string must contain at least one "/" character. The last segment of the URL's path must represent the fully qualified name of the type (as in `path/google.protobuf.Duration`). The name should be in a canonical form (e.g., leading "." is not accepted).  In practice, teams usually precompile into the binary all types that they expect it to use in the context of Any. However, for URLs which use the scheme `http`, `https`, or no scheme, one can optionally set up a type server that maps type URLs to message definitions as follows:  * If no scheme is provided, `https` is assumed. * An HTTP GET on the URL must yield a [google.protobuf.Type][]   value in binary format, or produce an error. * Applications are allowed to cache lookup results based on the   URL, or have them precompiled into a binary to avoid any   lookup. Therefore, binary compatibility needs to be preserved   on changes to types. (Use versioned type names to manage   breaking changes.)  Note: this functionality is not currently available in the official protobuf release, and it is not used for type URLs beginning with type.googleapis.com. As of May 2023, there are no widely used type server implementations and no plans to implement one.  Schemes other than `http`, `https` (or the empty scheme) might be used with implementation specific semantics. | No |
| value | byte | Must be a valid serialized protocol buffer of the above specified type. | No |

#### runtimeError

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| error | string |  | No |
| code | integer |  | No |
| message | string |  | No |
| details | [ [protobufAny](#protobufany) ] |  | No |

#### v1BalanceBank

BalanceBank is the Bank module balance view for a token.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| symbol | string |  | No |
| balance_human | string |  | No |
| decimals | integer |  | No |
| coin_denom | string |  | No |
| balance_base | string |  | No |

#### v1BalanceERC20

BalanceERC20 is the ERC20 balance view for a token.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| address | string |  | No |
| symbol | string |  | No |
| balance_human | string |  | No |
| decimals | integer |  | No |
| name | string |  | No |
| balance_base | string |  | No |

#### v1EstimateGasResponse

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| gas | string (uint64) |  | No |

#### v1FunToken

FunToken is a fungible token mapping between a Bank Coin and a corresponding
ERC-20 smart contract. Bank Coins here refer to tokens like NIBI, IBC
coins (ICS-20), and token factory coins, which are each represented by the
"Coin" type in Golang.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| erc20_addr | string |  | No |
| bank_denom | string | bank_denom: Coin denomination in the Bank Module. | No |
| is_made_from_coin | boolean | True if the `FunToken` mapping was created from an existing Bank Coin and the ERC-20 contract gets deployed by the module account. False if the mapping was created from an externally owned ERC-20 contract. | No |

#### v1Log

Log represents an protobuf compatible Ethereum Log that defines a contract
log event. These events are generated by the LOG opcode and stored/indexed by
the node.

NOTE: address, topics and data are consensus fields. The rest of the fields
are derived, i.e. filled in by the nodes, but not secured by consensus.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| address | string |  | No |
| topics | [ string ] | topics is a list of topics provided by the contract. | No |
| data | byte |  | No |
| block_number | string (uint64) |  | No |
| tx_hash | string |  | No |
| tx_index | string (uint64) |  | No |
| block_hash | string |  | No |
| index | string (uint64) |  | No |
| removed | boolean | removed is true if this log was reverted due to a chain reorganisation. You must pay attention to this field if you receive logs through a filter query. | No |

#### v1MsgEthereumTx

MsgEthereumTx encapsulates an Ethereum transaction as an SDK message.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| data | [protobufAny](#protobufany) |  | No |
| size | double |  | No |
| hash | string |  | No |
| from | string |  | No |

#### v1MsgEthereumTxResponse

MsgEthereumTxResponse defines the Msg/EthereumTx response type.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| hash | string |  | No |
| logs | [ [v1Log](#v1log) ] | logs contains the transaction hash and the proto-compatible ethereum logs. | No |
| ret | byte |  | No |
| vm_error | string |  | No |
| gas_used | string (uint64) |  | No |

#### v1Params

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| extra_eips | [ string (int64) ] |  | No |
| evm_channels | [ string ] |  | No |
| create_funtoken_fee | string | Fee deducted and burned when calling "CreateFunToken" in units of "evm_denom". | No |
| canonical_wnibi | string |  | No |

#### v1QueryBalanceResponse

QueryBalanceResponse is the response type for the Query/Balance RPC method.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| balance_wei | string | balance is the balance of the EVM denomination in units of wei. | No |
| bank | [v1BalanceBank](#v1balancebank) | bank is the Bank module token balance details when a bank representation is available for the requested token. | No |
| erc20 | [v1BalanceERC20](#v1balanceerc20) | erc20 is the ERC20 token balance details when an ERC20 representation is available for the requested token. | No |

#### v1QueryBaseFeeResponse

QueryBaseFeeResponse returns the EIP1559 base fee.
See https://github.com/ethereum/EIPs/blob/ba6c342c23164072adb500c3136e3ae6eabff306/EIPS/eip-1559.md.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| base_fee | string | base_fee is the EIP1559 base fee in units of wei. | No |
| base_fee_unibi | string | base_fee is the EIP1559 base fee in units of micronibi ("unibi"). | No |

#### v1QueryCodeResponse

QueryCodeResponse is the response type for the Query/Code RPC
method.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| code | byte | code represents the code bytes from an ethereum address. | No |

#### v1QueryEthAccountResponse

QueryEthAccountResponse is the response type for the Query/EthAccount RPC method.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| balance | string | balance is the balance of unibi (micronibi). | No |
| balance_wei | string | balance_wei is the balance of wei (attoether, where NIBI is ether). | No |
| code_hash | string | code_hash is the hex-formatted code bytes from the EOA. | No |
| nonce | string (uint64) | nonce is the account's sequence number. | No |
| eth_address | string | eth_address: The hexadecimal-encoded string representing the 20 byte address of a Nibiru EVM account. | No |
| bech32_address | string | bech32_address is the nibi-prefixed address of the account that can receive bank transfers ("cosmos.bank.v1beta1.MsgSend"). | No |

#### v1QueryFunTokenMappingResponse

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| fun_token | [v1FunToken](#v1funtoken) |  | No |

#### v1QueryParamsResponse

QueryParamsResponse defines the response type for querying x/evm parameters.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| params | [v1Params](#v1params) | params define the evm module parameters. | No |

#### v1QueryStorageResponse

QueryStorageResponse is the response type for the Query/Storage RPC
method.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| value | string | value defines the storage state value hash associated with the given key. | No |

#### v1QueryTraceBlockResponse

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| data | byte |  | No |

#### v1QueryTraceTxResponse

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| data | byte |  | No |

#### v1QueryValidatorAccountResponse

QueryValidatorAccountResponse is the response type for the
Query/ValidatorAccount RPC method.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| account_address | string | account_address is the Nibiru address of the account in bech32 format. | No |
| sequence | string (uint64) | sequence is the account's sequence number. | No |
| account_number | string (uint64) |  | No |

#### v1TraceConfig

TraceConfig holds extra parameters to trace functions.

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| tracer | string |  | No |
| timeout | string |  | No |
| reexec | string (uint64) |  | No |
| disable_stack | boolean |  | No |
| disable_storage | boolean |  | No |
| debug | boolean |  | No |
| limit | integer |  | No |
| enable_memory | boolean |  | No |
| enable_return_data | boolean |  | No |
| tracer_config | [v1TracerConfig](#v1tracerconfig) |  | No |

#### v1TracerConfig

| Name | Type | Description | Required |
| ---- | ---- | ----------- | -------- |
| only_top_call | boolean |  | No |
