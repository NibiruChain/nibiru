# CW Address Like

This crate provides an trait `AddressLike`, which marks types that can be used as addresses in CosmWasm, namely `String` and `cosmwasm_std::Addr`.

## Background

In CosmWasm, there are two types that are typically used to represent addresses:

- `String` - Represents an _unverified_ address, which are used in contract APIs, i.e. messages and query responses.
- `cosmwasm_std::Addr` - Represents an _verified_ address, used in contract internal logics.

When a contract receives an address (as a `String`) from a message, it must not simply assume it is valid. Instead, it should use the `deps.api.addr_validate` method to verify it, which returns an `Addr`. The contract can then use the `Addr` in its business logics or save it in storage.

Similarly, the contract should also converts `Addr`s back to `String`s when returning them in query responses.

### The problem

A problem arises when _we want to define a struct or enum that is to be used in both the API and internal logics._ For example, consider a contract that saves a "config" in its storage, which uses an `Addr` inside to represent the address of the contract's owner, while also providing a query method for the config, which uses a `String`.

In such cases, developers may either define two types, one for each case:

```rust
struct Config {
    pub owner: Addr,
}

struct ConfigResponse {
    pub owner: String,
}
```

This approach works, but is somewhat cumbersome, especially when the config contains more fields.

Another approach is to define a single type that contains a generic:

```rust
struct Config<T> {
    pub owner: T,
}
```

Then use `Config<String>` in the API and `Config<Addr>` in internal logics.

The main drawback of this approach is there's no restriction on what `T` can be. It is theoretically possible to plug any type in as `T` here, not limited to `String` and `Addr`.

## How to use

In this crate we provide an `AddressLike` trait, which is defined simply as:

```rust
pub trait AddressLike {}

impl AddressLike for Addr {}
impl AddressLike for String {}
```

The developer can then define their type as:

```rust
struct Config<T: AddressLike> {
    pub owner: T,
}
```

This restricts that only `String` and `Addr` can be used as `T`.

## License

Contents of this crate at or prior to version `1.0.3` are published under [GNU Affero General Public License v3](https://github.com/steak-enjoyers/cw-plus-plus/blob/9c8fcf1c95b74dd415caf5602068c558e9d16ecc/LICENSE) or later; contents after the said version are published under [Apache-2.0](../../LICENSE) license.
