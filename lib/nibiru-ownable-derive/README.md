# nibiru-ownable-derive

> Macros for generating code used by the `nibiru-ownable` crate.

`nibiru-ownable-derive` provides procedural macros that automatically inject ownership-related message variants into your CosmWasm contract's ExecuteMsg and QueryMsg enums, eliminating boilerplate code.

## Macros

### `#[ownable_execute]`

Adds an `UpdateOwnership(nibiru_ownable::Action)` variant to your ExecuteMsg enum:

```rust
use cosmwasm_schema::cw_serde;
use nibiru_ownable::ownable_execute;

#[ownable_execute]  // Must be applied before #[cw_serde]
#[cw_serde]
enum ExecuteMsg {
    Foo {},
    Bar {},
}
```

Expands to:

```rust
#[cw_serde]
enum ExecuteMsg {
    UpdateOwnership(::nibiru_ownable::Action),
    Foo {},
    Bar {},
}
```

### `#[ownable_query]`

Adds an `Ownership {}` variant to your QueryMsg enum:

```rust
use cosmwasm_schema::{cw_serde, QueryResponses};
use nibiru_ownable::ownable_query;

#[ownable_query]  // Must be applied before #[cw_serde]
#[cw_serde]
#[derive(QueryResponses)]
enum QueryMsg {
    #[returns(FooResponse)]
    Foo {},
}
```

Expands to:

```rust
#[cw_serde]
#[derive(QueryResponses)]
enum QueryMsg {
    #[returns(::nibiru_ownable::Ownership<String>)]
    Ownership {},
    #[returns(FooResponse)]
    Foo {},
}
```

## Usage

Add both crates to your `Cargo.toml`:

```toml
[dependencies]
nibiru-ownable = "0.7.0"
nibiru-ownable-derive = "0.7.0"
```

Import the macros from `nibiru-ownable` (they are re-exported):

```rust
use nibiru_ownable::{ownable_execute, ownable_query};
```

## Documentation

For detailed API documentation, visit [docs.rs/nibiru-ownable-derive](https://docs.rs/nibiru-ownable-derive).

## License

Contents of this crate at or prior to version `0.5.0` are published under [GNU Affero General Public License v3](https://github.com/steak-enjoyers/cw-plus-plus/blob/9c8fcf1c95b74dd415caf5602068c558e9d16ecc/LICENSE) or later; contents after the said version are published under [Apache-2.0](../../../LICENSE) license.
