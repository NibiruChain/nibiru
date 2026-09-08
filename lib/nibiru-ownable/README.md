# nibiru-ownable

> Utility for single-party ownership of CosmWasm smart contracts.

`nibiru-ownable` provides a comprehensive ownership management system for CosmWasm smart contracts, inspired by OpenZeppelin's Ownable pattern for Ethereum. It implements a two-phase ownership transfer mechanism with optional expiration deadlines, designed for the CosmWasm execution model.

## How to use

Initialize the owner during instantiation using the `initialize_owner` method provided by this crate:

```rust
use cosmwasm_std::{entry_point, DepsMut, Env, MessageInfo, Response};
use nibiru_ownable::OwnershipError;

#[entry_point]
pub fn instantiate(
    deps: DepsMut,
    env: Env,
    _info: MessageInfo,
    msg: InstantiateMsg,
) -> Result<Response<Empty>, OwnershipError> {
    nibiru_ownable::initialize_owner(deps.storage, deps.api, msg.owner.as_deref())?;
    Ok(Response::new())
}
```

Use the `#[ownable_execute]` macro to extend your execute message:

```rust
use cosmwasm_schema::cw_serde;
use nibiru_ownable::ownable_execute;

#[ownable_execute]
#[cw_serde]
enum ExecuteMsg {
    Foo {},
    Bar {},
}
```

The macro inserts a new variant, `UpdateOwnership` to the enum:

```rust
#[cw_serde]
enum ExecuteMsg {
    UpdateOwnership(nibiru_ownable::Action),
    Foo {},
    Bar {},
}
```

Where `nibiru_ownable::Action` is an enum with three variants:

- `Action::TransferOwnership { new_owner: String, expiry: Option<Expiration> }` - Propose to transfer ownership with optional deadline
- `Action::AcceptOwnership` - Accept the proposed ownership transfer
- `Action::RenounceOwnership` - Renounce ownership permanently, setting the contract's owner to None

Handle the messages using the `update_ownership` function provided by this crate:

```rust
use cosmwasm_std::{entry_point, DepsMut, Env, MessageInfo, Response};
use nibiru_ownable::{cw_serde, update_ownership, OwnershipError};

#[entry_point]
pub fn execute(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    msg: ExecuteMsg,
) -> Result<Response, OwnershipError> {
    match msg {
        ExecuteMsg::UpdateOwnership(action) => {
            update_ownership(deps, &env.block, &info.sender, action)?;
        }
        _ => unimplemneted!(),
    }
    Ok(Response::new())
}
```

Use the `#[ownable_query]` macro to extend your query message:

```rust
use cosmwasm_schema::{cw_serde, QueryResponses};
use nibiru_ownable::ownable_query;

#[ownable_query]
#[cw_serde]
#[derive(QueryResponses)]
pub enum QueryMsg {
    #[returns(FooResponse)]
    Foo {},
    #[returns(BarResponse)]
    Bar {},
}
```

The macro inserts a new variant, `Ownership`:

```rust
#[cw_serde]
#[derive(QueryResponses)]
enum QueryMsg {
    #[returns(Ownership<String>)]
    Ownership {},
    #[returns(FooResponse)]
    Foo {},
    #[returns(BarResponse)]
    Bar {},
}
```

Handle the message using the `get_ownership` function provided by this crate:

```rust
use cosmwasm_std::{entry_point, Deps, Env, Binary};
use nibiru_ownable::get_ownership;

#[entry_point]
pub fn query(deps: Deps, env: Env, msg: QueryMsg) -> StdResult<Binary> {
    match msg {
        QueryMsg::Ownership {} => to_binary(&get_ownership(deps.storage)?),
        _ => unimplemented!(),
    }
}
```

## Core Types

- `nibiru_ownable::Ownership<T>` - Struct containing current owner, pending owner, and expiry
- `nibiru_ownable::Action` - Enum for ownership management actions
- `nibiru_ownable::OwnershipError` - Error types for ownership operations

## Documentation

For detailed API documentation, visit [docs.rs/nibiru-ownable](https://docs.rs/nibiru-ownable).

## License

Contents of this crate at or prior to version `0.5.0` are published under [GNU Affero General Public License v3](https://github.com/steak-enjoyers/cw-plus-plus/blob/9c8fcf1c95b74dd415caf5602068c558e9d16ecc/LICENSE) or later; contents after the said version are published under [Apache-2.0](../../LICENSE) license.