# nibiru-ownable

`nibiru-ownable` provides two-step CosmWasm ownership and optional,
contract-local delegated permissions. The stored owner always passes an
owner-or-perm gate. Delegated membership never makes an account an owner.

## Ownership

Initialize ownership during instantiation, then dispatch the variant injected
by attribute macro `#[ownable_execute]` to function `update_ownership`.

```rust
nibiru_ownable::initialize_owner(deps.storage, msg.owner.as_deref())?;

match msg {
    ExecuteMsg::UpdateOwnership(action) => {
        nibiru_ownable::update_ownership(deps, &env.block, &info.sender, action)?;
    }
    _ => {}
}
```

Type `Action` supports transfer proposal, pending-owner acceptance, and
renunciation. A contract may reject actions that do not fit its own safety
rules before calling function `update_ownership`.

## Delegated permissions

Permission-aware contracts derive trait `PermPolicy` on the message enum passed
to `assert_msg_auth`. Use `#[ownable_execute(perms)]` to add owner-only
`UpdatePerms(Vec<PermUpdate>)` and ownership actions.

```rust
#[cw_serde]
enum ExecuteMsg {
    Admin(AdminExecuteMsg),
}

#[ownable_execute(perms)]
#[cw_serde]
#[derive(PermPolicy)]
#[perms(namespace = "admin")]
enum AdminExecuteMsg {
    #[perms(owner_or_any("operator"))]
    Reconcile {},
}
```

Call function `assert_msg_auth` before dispatching the covered message. It
checks the generated policy. Function `update_perms` validates the full batch
before it changes storage, then applies grants and revokes in order. Empty and
duplicate updates are valid, and each update produces a `perm_update` event.

```rust
match msg {
    ExecuteMsg::Admin(msg) => {
        nibiru_ownable::assert_msg_auth(deps.storage, &info.sender, &msg)?;

        if let AdminExecuteMsg::UpdatePerms(updates) = msg {
            let events = nibiru_ownable::update_perms::<AdminExecuteMsg>(
                deps.storage,
                deps.api,
                &info.sender,
                updates,
            )?;
            return Ok(Response::new().add_events(events));
        }
    }
}
```

Owner-or-perm policy does not replace application checks. A public policy route
can still restrict its handler to a contract, a route operator, or another
application-specific authority.

## Queries and addresses

Attribute macro `#[ownable_query(perms)]` adds these query variants:

- `Ownership {}` returns the stored ownership state.
- `Perms {}` returns the compiled `Vec<PermRule>` policy catalog. An enum-level
  `#[perms(namespace = "admin")]` prefix produces identifiers such as
  `admin.reconcile`.
- `PermsForMembers { members }` accepts Nibiru Bech32 address strings and
  returns delegated membership for each validated account. The owner receives
  virtual `owner` membership.

Permission updates and member queries use Bech32 strings because CosmWasm
validates them through the chain API. This supports externally owned accounts
and contract addresses, including CW3 multisigs. EVM hex is not a permission
member format.

## Documentation and license

- [API documentation](https://docs.rs/nibiru-ownable)
- [Nibiru source repository](https://github.com/NibiruChain/nibiru)

Contents of this crate at or before version `0.5.0` are licensed under AGPL-3.0
or later. Later contents are Apache-2.0. See the repository license for the
full terms.
