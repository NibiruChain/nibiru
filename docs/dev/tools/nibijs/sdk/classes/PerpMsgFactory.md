[NibiJS Documentation - v4.5.0](../README.md) / [Exports](../README.md) / PerpMsgFactory

# Class: PerpMsgFactory

PerpMsgFactory: Convenience methods for broadcasting transaction messages
(TxMessage) from Nibiru's x/perp module.

**`See`**

https://nibiru.fi/docs/ecosystem/nibi-perps/

## Table of contents

### Constructors

- [constructor](PerpMsgFactory.md#constructor)

### Methods

- [addMargin](PerpMsgFactory.md#addmargin)
- [closePosition](PerpMsgFactory.md#closeposition)
- [donateToPerpEF](PerpMsgFactory.md#donatetoperpef)
- [liquidate](PerpMsgFactory.md#liquidate)
- [openPosition](PerpMsgFactory.md#openposition)
- [partialClosePosition](PerpMsgFactory.md#partialcloseposition)
- [removeMargin](PerpMsgFactory.md#removemargin)

## Constructors

### constructor

• **new PerpMsgFactory**()

## Methods

### addMargin

▸ `Static` **addMargin**(`msg`): [`TxMessage`](../interfaces/TxMessage.md)

Returns a 'TxMessage' for adding margin to a position

#### Parameters

| Name  | Type           | Description           |
| :---- | :------------- | :-------------------- |
| `msg` | `MsgAddMargin` | Message to add margin |

#### Returns

[`TxMessage`](../interfaces/TxMessage.md)

- formatted version of MsgAddMargin

**`Static`**

#### Defined in

[msg/perp.ts:124](https://github.com/NibiruChain/ts-sdk/blob/23db897/packages/nibijs/src/msg/perp.ts#L124)

---

### closePosition

▸ `Static` **closePosition**(`msg`): [`TxMessage`](../interfaces/TxMessage.md)

#### Parameters

| Name  | Type               |
| :---- | :----------------- |
| `msg` | `MsgClosePosition` |

#### Returns

[`TxMessage`](../interfaces/TxMessage.md)

#### Defined in

[msg/perp.ts:161](https://github.com/NibiruChain/ts-sdk/blob/23db897/packages/nibijs/src/msg/perp.ts#L161)

---

### donateToPerpEF

▸ `Static` **donateToPerpEF**(`msg`): [`TxMessage`](../interfaces/TxMessage.md)

#### Parameters

| Name  | Type                       |
| :---- | :------------------------- |
| `msg` | `MsgDonateToEcosystemFund` |

#### Returns

[`TxMessage`](../interfaces/TxMessage.md)

#### Defined in

[msg/perp.ts:175](https://github.com/NibiruChain/ts-sdk/blob/23db897/packages/nibijs/src/msg/perp.ts#L175)

---

### liquidate

▸ `Static` **liquidate**(`msg`): [`TxMessage`](../interfaces/TxMessage.md)

#### Parameters

| Name  | Type                |
| :---- | :------------------ |
| `msg` | `MsgMultiLiquidate` |

#### Returns

[`TxMessage`](../interfaces/TxMessage.md)

#### Defined in

[msg/perp.ts:131](https://github.com/NibiruChain/ts-sdk/blob/23db897/packages/nibijs/src/msg/perp.ts#L131)

---

### openPosition

▸ `Static` **openPosition**(`msg`): [`TxMessage`](../interfaces/TxMessage.md)

#### Parameters

| Name                        | Type      |
| :-------------------------- | :-------- |
| `msg`                       | `Object`  |
| `msg.baseAssetAmountLimit?` | `number`  |
| `msg.goLong`                | `boolean` |
| `msg.leverage`              | `number`  |
| `msg.pair`                  | `string`  |
| `msg.quoteAssetAmount`      | `number`  |
| `msg.sender`                | `string`  |

#### Returns

[`TxMessage`](../interfaces/TxMessage.md)

#### Defined in

[msg/perp.ts:138](https://github.com/NibiruChain/ts-sdk/blob/23db897/packages/nibijs/src/msg/perp.ts#L138)

---

### partialClosePosition

▸ `Static` **partialClosePosition**(`msg`): [`TxMessage`](../interfaces/TxMessage.md)

#### Parameters

| Name  | Type              |
| :---- | :---------------- |
| `msg` | `MsgPartialClose` |

#### Returns

[`TxMessage`](../interfaces/TxMessage.md)

#### Defined in

[msg/perp.ts:168](https://github.com/NibiruChain/ts-sdk/blob/23db897/packages/nibijs/src/msg/perp.ts#L168)

---

### removeMargin

▸ `Static` **removeMargin**(`msg`): [`TxMessage`](../interfaces/TxMessage.md)

#### Parameters

| Name  | Type              |
| :---- | :---------------- |
| `msg` | `MsgRemoveMargin` |

#### Returns

[`TxMessage`](../interfaces/TxMessage.md)

#### Defined in

[msg/perp.ts:110](https://github.com/NibiruChain/ts-sdk/blob/23db897/packages/nibijs/src/msg/perp.ts#L110)
