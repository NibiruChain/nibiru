# nibiru-ownable-derive

`nibiru-ownable-derive` implements the procedural macros re-exported by
`nibiru-ownable`. Contract crates should depend on `nibiru-ownable`, not this
crate directly.

## Ownership macros

Attribute macro `#[ownable_execute]` adds an
`UpdateOwnership(nibiru_ownable::Action)` execute variant. Attribute macro
`#[ownable_query]` adds an `Ownership {}` query variant. Apply either macro
before `#[cw_serde]`.

```rust
#[ownable_execute]
#[cw_serde]
enum ExecuteMsg {
    Ping {},
}
```

## Permission macros

Derive macro `PermPolicy` builds authorization requirements and a queryable
catalog from a `#[perms(...)]` declaration on every enum variant:

```rust
#[derive(PermPolicy)]
enum AdminExecuteMsg {
    #[perms(owner_or_any())]
    SetOwnerOnlyValue {},

    #[perms(owner_or_any("operator"))]
    SetOperatorValue {},

    #[perms(public)]
    HandlerAuthorizesThis {},
}
```

- `public` makes no shared authorization decision. The handler remains
  responsible for any caller checks.
- `owner_or_any()` requires the stored owner.
- `owner_or_any("operator")` accepts the stored owner or a member of that
  delegated perm.
- `nested = "field"` delegates to the policy of a named enum field.
- `nested` delegates through a one-field tuple variant.

`#[ownable_execute(perms)]` also adds owner-only
`UpdatePerms(Vec<PermUpdate>)` and nested `UpdateOwnership` variants.
`#[ownable_query(perms)]` adds `Ownership`, `Perms`, and `PermsForMembers`
query variants. Permission mode requires `#[derive(PermPolicy)]` on the same
enum. The forms without `(perms)` retain the ownership-only API.

The generated policy does not enforce authorization by itself. The contract
entry point calls function `nibiru_ownable::assert_msg_auth` before dispatch.

## Documentation and license

- [Public crate documentation](https://docs.rs/nibiru-ownable)
- [Macro crate documentation](https://docs.rs/nibiru-ownable-derive)
- [Nibiru source repository](https://github.com/NibiruChain/nibiru)

Contents of this crate at or before version `0.5.0` are licensed under AGPL-3.0
or later. Later contents are Apache-2.0. See the repository license for the
full terms.
