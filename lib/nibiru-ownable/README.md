# nibiru-ownable

> Utility for single-party ownership of CosmWasm smart contracts.

`nibiru-ownable` provides single-owner management and optional delegated perms
for CosmWasm contracts. Ownership transfer uses a two-step propose-and-accept
flow with an optional expiration.

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
    nibiru_ownable::initialize_owner(deps.storage, msg.owner.as_deref())?;
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
        _ => unimplemented!(),
    }
    Ok(Response::new())
}
```

## Delegated perms

Perm-aware contracts derive `PermPolicy` on their execute enums. Every variant
must declare exactly one gate:

- `#[perms(public)]` leaves authorization to the handler.
- `#[perms(owner_or_any())]` allows only the owner.
- `#[perms(owner_or_any("sai_oper", "intent_executor"))]` allows the owner or
  a member of either listed perm.
- `#[perms(nested = "msg")]` delegates to a named nested message.
- `#[perms(nested)]` delegates through a one-field tuple variant.

Use `#[ownable_execute(perms)]` to inject owner-only `UpdatePerms` and nested
`UpdateOwnership` variants. This mode requires `#[derive(PermPolicy)]`:

```rust
use cosmwasm_schema::cw_serde;
use cosmwasm_std::Uint128;
use nibiru_ownable::{ownable_execute, PermPolicy};

#[cw_serde]
#[derive(PermPolicy)]
enum AdminExecuteMsg {
    #[perms(owner_or_any("sai_oper"))]
    SetMinimumPositionSize { value: Uint128 },
}

#[ownable_execute(perms)]
#[cw_serde]
#[derive(PermPolicy)]
enum ExecuteMsg {
    #[perms(public)]
    Deposit {},

    #[perms(nested = "msg")]
    Admin { msg: AdminExecuteMsg },
}
```

Call `assert_message_authorized` before dispatch. Apply `UpdatePerms` with
`update_perms::<ExecuteMsg>`. The whole batch is validated before storage is
changed, then grants and revokes are applied in listed order. Duplicate and
empty updates are valid.

```rust
assert_message_authorized(deps.storage, &info.sender, &msg)?;

if let ExecuteMsg::UpdatePerms(updates) = msg {
    let events = update_perms::<ExecuteMsg>(deps.storage, &info.sender, updates)?;
    return Ok(Response::new().add_events(events));
}
```

Use `#[ownable_query(perms)]` to inject these query variants:

- `Ownership {}` returns the ownership state.
- `Perms {}` returns the compiled `Vec<PermRule>` catalog. Only gated routes
  appear. Nested routes use serialized names, for example
  `admin.msg.set_minimum_position_size`.
- `PermsForMembers { members: Vec<UserAddr> }` returns each requested member's
  delegated perms. The current owner's result also contains the virtual
  `owner` perm.

`UserAddr` is serialized as one JSON string. Input accepts either a Nibiru
bech32 externally owned account or a `0x`-prefixed 20-byte EVM address. Output
uses EIP-55 hex, and `to_bech32_addr()` returns the equivalent Nibiru address.

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
- `nibiru_ownable::PermPolicy` - Generated execute-message authorization policy
- `nibiru_ownable::PermRule` - One discoverable gated execute route
- `nibiru_ownable::PermUpdate` - One delegated membership grant or revoke
- `nibiru_ownable::UserAddr` - Validated dual-format externally owned account

## Documentation

For detailed API documentation, visit [docs.rs/nibiru-ownable](https://docs.rs/nibiru-ownable).

## License

Contents of this crate at or prior to version `0.5.0` are published under [GNU Affero General Public License v3](https://github.com/steak-enjoyers/cw-plus-plus/blob/9c8fcf1c95b74dd415caf5602068c558e9d16ecc/LICENSE) or later; contents after the said version are published under [Apache-2.0](../../LICENSE) license.
