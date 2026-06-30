[NibiJS Documentation - v4.5.0](../README.md) / [Exports](../README.md) / StableSwap

# Class: StableSwap

StableSwap contains the logic for exchanging tokens

Based on: <https://github.com/NibiruChain/nibiru/blob/master/contrib/scripts/testing/stableswap_model.py>

Constructor:

| Parameter          | Type          | Description                                          |
|--------------------|---------------|------------------------------------------------------|
| `Amplification`    | `BigNumber`   | The amplification coefficient of the pool.           |
| `totalTokenSupply` | `BigNumber[]` | The total supply of each token in the pool.          |
| `tokenPrices`      | `BigNumber[]` | The prices of each token in the pool.                |
| `fee`              | `BigNumber`   | The fee applied to transactions in the pool.         |

## Table of contents

### Constructors

- [constructor](StableSwap.md#constructor)

### Properties

- [Amplification](StableSwap.md#amplification)
- [fee](StableSwap.md#fee)
- [totalTokenSupply](StableSwap.md#totaltokensupply)
- [totalTokensInPool](StableSwap.md#totaltokensinpool)

### Methods

- [D](StableSwap.md#d)
- [exchange](StableSwap.md#exchange)
- [xp](StableSwap.md#xp)
- [y](StableSwap.md#y)

## Constructors

### constructor

• **new StableSwap**(`Amplification`, `totalTokenSupply`, `fee`)

#### Parameters

| Name               | Type          |
| :----------------- | :------------ |
| `Amplification`    | `BigNumber`   |
| `totalTokenSupply` | `BigNumber`[] |
| `fee`              | `BigNumber`   |

#### Defined in

[stableswap/stableswap.ts:25](https://github.com/NibiruChain/ts-sdk/blob/23db897/packages/nibijs/src/stableswap/stableswap.ts#L25)

## Properties

### Amplification

• **Amplification**: `BigNumber`

#### Defined in

[stableswap/stableswap.ts:20](https://github.com/NibiruChain/ts-sdk/blob/23db897/packages/nibijs/src/stableswap/stableswap.ts#L20)

---

### fee

• **fee**: `BigNumber`

#### Defined in

[stableswap/stableswap.ts:23](https://github.com/NibiruChain/ts-sdk/blob/23db897/packages/nibijs/src/stableswap/stableswap.ts#L23)

---

### totalTokenSupply

• **totalTokenSupply**: `BigNumber`[]

#### Defined in

[stableswap/stableswap.ts:21](https://github.com/NibiruChain/ts-sdk/blob/23db897/packages/nibijs/src/stableswap/stableswap.ts#L21)

---

### totalTokensInPool

• **totalTokensInPool**: `BigNumber`

#### Defined in

[stableswap/stableswap.ts:22](https://github.com/NibiruChain/ts-sdk/blob/23db897/packages/nibijs/src/stableswap/stableswap.ts#L22)

## Methods

### D

▸ **D**(): `BigNumber`

D()

D invariant calculation in non-overflowing integer operations iteratively
A _sum(x_i) _n**n + D = A _ D _ n**n + D**(n+1) / (n**n \* prod(x_i))

#### Returns

`BigNumber`

**`Memberof`**

StableSwap

#### Defined in

[stableswap/stableswap.ts:54](https://github.com/NibiruChain/ts-sdk/blob/23db897/packages/nibijs/src/stableswap/stableswap.ts#L54)

---

### exchange

▸ **exchange**(`fromIndex`, `toIndex`, `dx`): `BigNumber`

exchange() runs a theorhetical Curve StableSwap model to determine impact on token price

#### Parameters

| Name        | Type        |
| :---------- | :---------- |
| `fromIndex` | `number`    |
| `toIndex`   | `number`    |
| `dx`        | `BigNumber` |

#### Returns

`BigNumber`

**`Memberof`**

StableSwap

#### Defined in

[stableswap/stableswap.ts:143](https://github.com/NibiruChain/ts-sdk/blob/23db897/packages/nibijs/src/stableswap/stableswap.ts#L143)

---

### xp

▸ **xp**(): `BigNumber`[]

xp() gives an array of total tokens

#### Returns

`BigNumber`[]

**`Memberof`**

StableSwap

#### Defined in

[stableswap/stableswap.ts:41](https://github.com/NibiruChain/ts-sdk/blob/23db897/packages/nibijs/src/stableswap/stableswap.ts#L41)

---

### y

▸ **y**(`fromIndex`, `toIndex`, `x`): `BigNumber`

y()

Calculate x[j] if one makes x[i] = x

Done by solving quadratic equation iteratively.
x_1**2 + x1 * (sum' - (A*n**n - 1)_ D / (A _n**n)) = D ** (n+1)/(n ** (2 _ n) _ prod' \* A)
x_1**2 + b\*x_1 = c

x_1 = (x_1\**2 + c) / (2*x_1 + b)

#### Parameters

| Name        | Type        |
| :---------- | :---------- |
| `fromIndex` | `number`    |
| `toIndex`   | `number`    |
| `x`         | `BigNumber` |

#### Returns

`BigNumber`

**`Memberof`**

StableSwap

#### Defined in

[stableswap/stableswap.ts:104](https://github.com/NibiruChain/ts-sdk/blob/23db897/packages/nibijs/src/stableswap/stableswap.ts#L104)
