# x/sudo permissions

Root owns the module. It can rotate the root address, edit named roles, configure
zero-gas actors, and configure the Wasm block-hook registry. A role can contain
EOA or contract addresses. Role membership only grants access where a caller
explicitly checks that role.

`CheckPermissions(actor, ctx, role)` accepts root or a member of `role`.
An empty role requires root. Unknown roles grant no permission. Root rotation
and role revocation take effect on the next check.

The `wasm_deployer` role allows guarded mainnet code uploads and migrations.
Normal Wasm upload permissions, code instantiation access, and contract admin
checks still apply. A deployer cannot migrate a Treasury-administered contract
unless the Treasury authorizes the migration through the existing Wasm paths.

## Edit membership

Only root can submit `/nibiru.sudo.v1.MsgUpdateRoleMembers`. A CW3 root uses a
Stargate message with itself as `sender`. The same message has a CLI command:

```sh
nibid tx sudo update-role-members edit.json --from ROOT
nibid query sudo state
```

```json
{
  "role": "wasm_deployer",
  "add": ["nibi1..."],
  "remove": []
}
```

Role member inputs accept Bech32 and EVM hexadecimal addresses through
`eth.NibiruAddrFromStr`. State stores canonical lowercase Bech32 addresses.
Different spellings of the same account identify one member for grants,
revocations, genesis normalization, and add/remove overlap checks.

Members cannot appear in both `add` and `remove`. Repeated additions or removals
are idempotent. Empty roles are rejected. Members and roles are stored in sorted
order with unique members; removing every member removes the role entry.
Inflation, tokenfactory, EVM configuration, and zero-gas configuration remain
root-only, subject to each caller's existing governance authorization.

## v2.21.0 migration

The module migrates consensus version 1 to 2. The `Sudoers.root` wire field
remains tag 1. The retired `contracts` field's tag 2 and name are reserved;
`roles` uses tag 3. Legacy members receive no implicit role grants.

The v2.21.0 upgrade preserves root, zero-gas configuration, and the Wasm
block-hook registry. It seeds `wasm_deployer` on every chain with the reviewed
ud-prod, ud-prod-2, Matthias, Oleg, and sai-perp-admin addresses. Root can edit
these grants afterward. Exported genesis and sudoers queries use `roles`;
clients must update for the removal of `Sudoers.contracts`.

## Wasm block hooks

The block-hook registry has its own storage item and remains root-controlled.
Keep using `/nibiru.sudo.v1.MsgEditSudoers` with action
`edit_wasm_block_hooks_contract` and `contracts: ["nibi1..."]`. Use
`contracts: [""]` to clear it. The payload field remains part of this message,
although the broad membership field is removed from `Sudoers` state.

BeginBlock and EndBlock read the configured registry, query its dispatch plan,
and call target contracts through `WasmKeeper.Sudo`. These calls do not require
a deployment role. The retired `add_contracts` and `remove_contracts` actions
are rejected; use `MsgUpdateRoleMembers` to manage scoped privileges.

Instantiation is reopened under normal Wasm permissions. Release this change
with the runtime fix that permits removing the temporary instantiation guard.
