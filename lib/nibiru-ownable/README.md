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

Permission-aware contracts derive trait `PermPolicy` and annotate every execute
variant. Use `#[ownable_execute(perms)]` to add owner-only
`UpdatePerms(Vec<PermUpdate>)` and nested ownership handling.

```rust
#[ownable_execute(perms)]
#[cw_serde]
#[derive(PermPolicy)]
enum ExecuteMsg {
    #[perms(nested = "msg")]
    Admin { msg: AdminExecuteMsg },
}

#[cw_serde]
#[derive(PermPolicy)]
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
nibiru_ownable::assert_msg_auth(deps.storage, &info.sender, &msg)?;

if let ExecuteMsg::UpdatePerms(updates) = msg {
    let events = nibiru_ownable::update_perms::<ExecuteMsg>(
        deps.storage,
        &info.sender,
        updates,
    )?;
    return Ok(Response::new().add_events(events));
}
```

Owner-or-perm policy does not replace application checks. A public policy route
can still restrict its handler to a contract, a route operator, or another
application-specific authority.

## Queries and addresses

Attribute macro `#[ownable_query(perms)]` adds these query variants:

- `Ownership {}` returns the stored ownership state.
- `Perms {}` returns the compiled `Vec<PermRule>` policy catalog.
- `PermsForMembers { members }` returns delegated membership for supplied
  `UserAddr` values. The owner receives virtual `owner` membership.

Type `UserAddr` accepts Nibiru bech32 and `0x`-prefixed EVM addresses in JSON.
It serializes as canonical EIP-55 hex and represents only 20-byte externally
owned accounts.

## Documentation and license

- [API documentation](https://docs.rs/nibiru-ownable)
- [Nibiru source repository](https://github.com/NibiruChain/nibiru)

Contents of this crate at or before version `0.5.0` are licensed under AGPL-3.0
or later. Later contents are Apache-2.0. See the repository license for the
full terms.
