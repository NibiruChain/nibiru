[NibiJS Documentation - v4.5.0](../README.md) / [Exports](../README.md) / ABCIEvent

# Interface: ABCIEvent

An event as defined by the CometBFT consensus algorithm's
ABCI (application blockchain interface) specification.
Events are non-merklized JSON payloads emitted during transaction
execution on the network. Each event has a type and a list of
key-value strings of arbitrary data.

## Table of contents

### Properties

- [attributes](ABCIEvent.md#attributes)
- [type](ABCIEvent.md#type)

## Properties

### attributes

• **attributes**: [`EventAttribute`](EventAttribute.md)[]

#### Defined in

[tx/event.ts:23](https://github.com/NibiruChain/ts-sdk/blob/23db897/packages/nibijs/src/tx/event.ts#L23)

---

### type

• **type**: `string`

#### Defined in

[tx/event.ts:22](https://github.com/NibiruChain/ts-sdk/blob/23db897/packages/nibijs/src/tx/event.ts#L22)
