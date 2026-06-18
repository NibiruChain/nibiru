---
order: 7
---

[NibiJS Documentation - v4.5.0](README.md) / Exports

# NibiJS - Exports

## Table of contents

### Enumerations

- [BECH32_PREFIX](enums/BECH32_PREFIX.md)
- [Signer](enums/Signer.md)

### Classes

- [CustomChain](classes/CustomChain.md)
- [MsgFactory](classes/MsgFactory.md)
- [NibiruQuerier](classes/NibiruQuerier.md)
- [NibiruTxClient](classes/NibiruTxClient.md)
- [PerpMsgFactory](classes/PerpMsgFactory.md)
- [Result](classes/Result.md)
- [SpotMsgFactory](classes/SpotMsgFactory.md)
- [StableSwap](classes/StableSwap.md)

### Interfaces

- [ABCIEvent](interfaces/ABCIEvent.md)
- [Chain](interfaces/Chain.md)
- [ChainIdParts](interfaces/ChainIdParts.md)
- [CoinMap](interfaces/CoinMap.md)
- [EpochsExtension](interfaces/EpochsExtension.md)
- [EventAttribute](interfaces/EventAttribute.md)
- [EventMap](interfaces/EventMap.md)
- [EventMapAttribute](interfaces/EventMapAttribute.md)
- [InflationExtension](interfaces/InflationExtension.md)
- [MsgAddMarginEncodeObject](interfaces/MsgAddMarginEncodeObject.md)
- [MsgClosePositionEncodeObject](interfaces/MsgClosePositionEncodeObject.md)
- [MsgCreatePoolEncodeObject](interfaces/MsgCreatePoolEncodeObject.md)
- [MsgDonateToEcosystemFundEncodeObject](interfaces/MsgDonateToEcosystemFundEncodeObject.md)
- [MsgExitPoolEncodeObject](interfaces/MsgExitPoolEncodeObject.md)
- [MsgJoinPoolEncodeObject](interfaces/MsgJoinPoolEncodeObject.md)
- [MsgMultiLiquidateEncodeObject](interfaces/MsgMultiLiquidateEncodeObject.md)
- [MsgOpenPositionEncodeObject](interfaces/MsgOpenPositionEncodeObject.md)
- [MsgPartialCloseEncodeObject](interfaces/MsgPartialCloseEncodeObject.md)
- [MsgRemoveMarginEncodeObject](interfaces/MsgRemoveMarginEncodeObject.md)
- [MsgSwapAssetsEncodeObject](interfaces/MsgSwapAssetsEncodeObject.md)
- [MsgTypeUrls](interfaces/MsgTypeUrls.md)
- [OracleExtension](interfaces/OracleExtension.md)
- [PageRequest](interfaces/PageRequest.md)
- [PerpExtension](interfaces/PerpExtension.md)
- [SpotExtension](interfaces/SpotExtension.md)
- [SudoExtension](interfaces/SudoExtension.md)
- [TxLog](interfaces/TxLog.md)
- [TxMessage](interfaces/TxMessage.md)

### Type Aliases

- [NibiruExtensions](exports.md#nibiruextensions)

### Variables

- [ERR](exports.md#err)
- [INT_MULT](exports.md#int_mult)
- [Localnet](exports.md#localnet)
- [Msg](exports.md#msg)
- [PERP_MSG_TYPE_URLS](exports.md#perp_msg_type_urls)
- [SPOT_MSG_TYPE_URLS](exports.md#spot_msg_type_urls)
- [TEST_ADDRESS](exports.md#test_address)
- [TEST_CHAIN](exports.md#test_chain)
- [TEST_MNEMONIC](exports.md#test_mnemonic)
- [nibiruRegistryTypes](exports.md#nibiruregistrytypes)
- [perpTypes](exports.md#perptypes)
- [spotTypes](exports.md#spottypes)


### NibiruExtensions

Ƭ **NibiruExtensions**: `StargateQueryClient` & [`SpotExtension`](interfaces/SpotExtension.md) & [`PerpExtension`](interfaces/PerpExtension.md) & [`SudoExtension`](interfaces/SudoExtension.md) & [`InflationExtension`](interfaces/InflationExtension.md) & [`OracleExtension`](interfaces/OracleExtension.md) & [`EpochsExtension`](interfaces/EpochsExtension.md) & `DistributionExtension` & `GovExtension` & `StakingExtension` & `IbcExtension` & `WasmExtension` & `AuthExtension`

#### Defined in

[query/query.ts:32](https://github.com/NibiruChain/ts-sdk/blob/23db897/packages/nibijs/src/query/query.ts#L32)

## Variables

### ERR

• `Const` **ERR**: `Object`

#### Type declaration

| Name          | Type     |
| :------------ | :------- |
| `collections` | `string` |
| `noPrices`    | `string` |
| `sequence`    | `string` |

#### Defined in

[testutil.ts:19](https://github.com/NibiruChain/ts-sdk/blob/23db897/packages/nibijs/src/testutil.ts#L19)

---

### INT_MULT

• `Const` **INT_MULT**: `1000000`

#### Defined in

[chain/parse.ts:2](https://github.com/NibiruChain/ts-sdk/blob/23db897/packages/nibijs/src/chain/parse.ts#L2)

---

### Localnet

• `Const` **Localnet**: [`Chain`](interfaces/Chain.md)

Localnet: "Chain" configuration for a local Nibiru network. A local
environment is no different from a real one, except that it has a single
validator running on your host machine. Localnet is primarily used as a
controllable, isolated development environment for testing purposes.

#### Defined in

[chain/chain.ts:93](https://github.com/NibiruChain/ts-sdk/blob/23db897/packages/nibijs/src/chain/chain.ts#L93)

---

### Msg

• `Const` **Msg**: [`MsgFactory`](classes/MsgFactory.md)

#### Defined in

[msg/index.ts:9](https://github.com/NibiruChain/ts-sdk/blob/23db897/packages/nibijs/src/msg/index.ts#L9)

---

### PERP_MSG_TYPE_URLS

• `Const` **PERP_MSG_TYPE_URLS**: `Object`

#### Type declaration

| Name                       | Type     |
| :------------------------- | :------- |
| `MsgAddMargin`             | `string` |
| `MsgClosePosition`         | `string` |
| `MsgDonateToEcosystemFund` | `string` |
| `MsgMarketOrder`           | `string` |
| `MsgMultiLiquidate`        | `string` |
| `MsgPartialClose`          | `string` |
| `MsgRemoveMargin`          | `string` |

#### Defined in

[msg/perp.ts:16](https://github.com/NibiruChain/ts-sdk/blob/23db897/packages/nibijs/src/msg/perp.ts#L16)

---

### SPOT_MSG_TYPE_URLS

• `Const` **SPOT_MSG_TYPE_URLS**: `Object`

#### Type declaration

| Name            | Type     |
| :-------------- | :------- |
| `MsgCreatePool` | `string` |
| `MsgExitPool`   | `string` |
| `MsgJoinPool`   | `string` |
| `MsgSwapAssets` | `string` |

#### Defined in

[msg/spot.ts:12](https://github.com/NibiruChain/ts-sdk/blob/23db897/packages/nibijs/src/msg/spot.ts#L12)

---

### TEST_ADDRESS

• `Const` **TEST_ADDRESS**: `string`

Address for the wallet of the default validator on localnet"

#### Defined in

[testutil.ts:16](https://github.com/NibiruChain/ts-sdk/blob/23db897/packages/nibijs/src/testutil.ts#L16)

---

### TEST_CHAIN

• `Const` **TEST_CHAIN**: [`Chain`](interfaces/Chain.md) = `Localnet`

TEST_CHAIN: Alias for Localnet.

**`See`**

Localnet

#### Defined in

[testutil.ts:8](https://github.com/NibiruChain/ts-sdk/blob/23db897/packages/nibijs/src/testutil.ts#L8)

---

### TEST_MNEMONIC

• `Const` **TEST_MNEMONIC**: `string`

Mnemonic for the wallet of the default validator on localnet"

#### Defined in

[testutil.ts:11](https://github.com/NibiruChain/ts-sdk/blob/23db897/packages/nibijs/src/testutil.ts#L11)

---

### nibiruRegistryTypes

• `Const` **nibiruRegistryTypes**: `ReadonlyArray`<[`string`, `GeneratedType`]\>

#### Defined in

[tx/txClient.ts:31](https://github.com/NibiruChain/ts-sdk/blob/23db897/packages/nibijs/src/tx/txClient.ts#L31)

---

### perpTypes

• `Const` **perpTypes**: `ReadonlyArray`<[`string`, `GeneratedType`]\>

#### Defined in

[msg/perp.ts:26](https://github.com/NibiruChain/ts-sdk/blob/23db897/packages/nibijs/src/msg/perp.ts#L26)

---

### spotTypes

• `Const` **spotTypes**: `ReadonlyArray`<[`string`, `GeneratedType`]\>

#### Defined in

[msg/spot.ts:19](https://github.com/NibiruChain/ts-sdk/blob/23db897/packages/nibijs/src/msg/spot.ts#L19)

