[NibiJS Documentation - v4.5.0](../README.md) / [Exports](../README.md) / PageRequest

# Interface: PageRequest

An offset pagination request.

Pagination is the process of dividing a document into discrete pages.
Pagination in the context of API requests resembles this process.

**`Export`**

PageRequest

## Table of contents

### Properties

- [countTotal](PageRequest.md#counttotal)
- [key](PageRequest.md#key)
- [limit](PageRequest.md#limit)
- [offset](PageRequest.md#offset)
- [reverse](PageRequest.md#reverse)

## Properties

### countTotal

• **countTotal**: `boolean`

count_total is set to true to indicate that the result set should include
a count of the total number of items available for pagination in UIs.
count_total is only respected when offset is used. It is ignored when key
is set.

#### Defined in

[query/spot.ts:252](https://github.com/NibiruChain/ts-sdk/blob/23db897/packages/nibijs/src/query/spot.ts#L252)

---

### key

• **key**: `Uint8Array`

key is a value returned in PageResponse.next_key to begin
querying the next page most efficiently. Only one of offset or key
should be set.

#### Defined in

[query/spot.ts:234](https://github.com/NibiruChain/ts-sdk/blob/23db897/packages/nibijs/src/query/spot.ts#L234)

---

### limit

• **limit**: `number`

limit is the total number of results to be returned in the result page.
If left empty it will default to a value to be set by each app.

#### Defined in

[query/spot.ts:245](https://github.com/NibiruChain/ts-sdk/blob/23db897/packages/nibijs/src/query/spot.ts#L245)

---

### offset

• **offset**: `number`

offset is a numeric offset that can be used when key is unavailable.
It is less efficient than using key. Only one of offset or key should
be set.

#### Defined in

[query/spot.ts:240](https://github.com/NibiruChain/ts-sdk/blob/23db897/packages/nibijs/src/query/spot.ts#L240)

---

### reverse

• **reverse**: `boolean`

reverse is set to true if results are to be returned in the descending order.

Since: cosmos-sdk 0.43

#### Defined in

[query/spot.ts:258](https://github.com/NibiruChain/ts-sdk/blob/23db897/packages/nibijs/src/query/spot.ts#L258)
