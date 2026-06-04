[NibiJS Documentation - v4.5.0](../README.md) / [Exports](../README.md) / Result

# Class: Result<T\>

Poor-man's Result type from Rust.

The Result type forces you to explicitly handle errors in contrast to allowing
errors to propagate up the call stack implicitly. Handling potential errors
explicitly leads to more robust and reliable code.

Ref: <a href="https://doc.rust-lang.org/book/ch09-02-recoverable-errors-with-result.html#propagating-errors">Propagating Errors - Rust Book</a>.

**`Example`**

```ts
// ---------------------------------------
// Most common use-case: Result.ofSafeExec
// ---------------------------------------
res = Result.ofSafeExec(somethingDangerous) // without args

// with args
res = Result.ofSafeExec(() => somethingDangerous(arg0, arg1))
```

**`Example`**

```ts
// ---------------------------------------
// Direct constructor
// ---------------------------------------
let res = new Result({ ok: "Operation successful!" })
if (res.isOk()) {
  happyPath(res.ok)
} else {
  handleGracefully(res.err!) // throws impossible based on constructor args
}
```

## Type parameters

| Name |
| :--- |
| `T`  |

## Table of contents

### Constructors

- [constructor](Result.md#constructor)

### Properties

- [err](Result.md#err)
- [ok](Result.md#ok)

### Methods

- [isErr](Result.md#iserr)
- [isOk](Result.md#isok)
- [ofSafeExec](Result.md#ofsafeexec)
- [ofSafeExecAsync](Result.md#ofsafeexecasync)

## Constructors

### constructor

• **new Result**<`T`\>(`«destructured»`)

#### Type parameters

| Name |
| :--- |
| `T`  |

#### Parameters

| Name             | Type      |
| :--------------- | :-------- |
| `«destructured»` | `Object`  |
| › `err?`         | `unknown` |
| › `ok?`          | `T`       |

#### Defined in

[result.ts:33](https://github.com/NibiruChain/ts-sdk/blob/23db897/packages/nibijs/src/result.ts#L33)

## Properties

### err

• **err**: `undefined` \| `Error`

#### Defined in

[result.ts:32](https://github.com/NibiruChain/ts-sdk/blob/23db897/packages/nibijs/src/result.ts#L32)

---

### ok

• **ok**: `undefined` \| `T`

#### Defined in

[result.ts:31](https://github.com/NibiruChain/ts-sdk/blob/23db897/packages/nibijs/src/result.ts#L31)

## Methods

### isErr

▸ **isErr**(): `boolean`

#### Returns

`boolean`

#### Defined in

[result.ts:44](https://github.com/NibiruChain/ts-sdk/blob/23db897/packages/nibijs/src/result.ts#L44)

---

### isOk

▸ **isOk**(): `boolean`

#### Returns

`boolean`

#### Defined in

[result.ts:45](https://github.com/NibiruChain/ts-sdk/blob/23db897/packages/nibijs/src/result.ts#L45)

---

### ofSafeExec

▸ `Static` **ofSafeExec**<`Y`\>(`fn`): [`Result`](Result.md)<`Y`\>

Constructor for "Result" using the return value of the input function.

#### Type parameters

| Name |
| :--- |
| `Y`  |

#### Parameters

| Name | Type                        |
| :--- | :-------------------------- |
| `fn` | (...`args`: `any`[]) => `Y` |

#### Returns

[`Result`](Result.md)<`Y`\>

#### Defined in

[result.ts:48](https://github.com/NibiruChain/ts-sdk/blob/23db897/packages/nibijs/src/result.ts#L48)

---

### ofSafeExecAsync

▸ `Static` **ofSafeExecAsync**<`Y`\>(`fn`): `Promise`<[`Result`](Result.md)<`Y`\>\>

Constructor for "Result" using the return value of the input async function.

#### Type parameters

| Name |
| :--- |
| `Y`  |

#### Parameters

| Name | Type                  |
| :--- | :-------------------- |
| `fn` | () => `Promise`<`Y`\> |

#### Returns

`Promise`<[`Result`](Result.md)<`Y`\>\>

**`Example`**

```ts
const result = Result.ofSafeExecAsync(async () => someAsyncFunc(args))
```

#### Defined in

[result.ts:60](https://github.com/NibiruChain/ts-sdk/blob/23db897/packages/nibijs/src/result.ts#L60)
