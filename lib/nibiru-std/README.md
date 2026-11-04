# nibiru-std

`nibiru-std` contains Nibiru-specific types for CosmWasm contracts. It provides
Stargate message and query types, numeric helpers, and the shared address type
used by Nibiru contract APIs.

## User addresses

Type `UserAddr` represents a 20-byte Nibiru externally owned account. It
accepts either canonical Nibiru bech32 or a `0x`-prefixed EVM address when
parsing JSON. Both forms identify the same account.

```rust
use nibiru_std::address::UserAddr;

let user: UserAddr =
    "nibi1gc24lt74ses9swkq6g7cug4e5y72p7e34jqgul".parse()?;

assert_eq!(user.to_hex(), "0x46155fAfd58660583ac0d23d8E22B9A13Ca0fb31");
assert_eq!(
    user.to_bech32_addr().as_str(),
    "nibi1gc24lt74ses9swkq6g7cug4e5y72p7e34jqgul",
);
# Ok::<(), nibiru_std::errors::NibiruError>(())
```

`UserAddr` serializes as canonical EIP-55 hex. It models the 20-byte identity
used by EVM-compatible user flows. It is not a generic CosmWasm account type.
Use a validated `cosmwasm_std::Addr` for contract callers, CW3 multisigs, and
permission members.

## Stargate support

The crate supplies typed constructors for Nibiru and Cosmos SDK messages that
contracts send through `CosmosMsg::Stargate`, plus matching
`QueryRequest::Stargate` request and response types. See the generated API docs
for the available modules and message types.

## Documentation and license

- [API documentation](https://docs.rs/nibiru-std)
- [Nibiru source repository](https://github.com/NibiruChain/nibiru)

`nibiru-std` is licensed under the terms in [LICENSE](LICENSE).

## Storage transactions

Module `storage_txn` provides `StorageTransaction`, a write overlay over
read-only CosmWasm storage. Reads see pending writes; range iteration merges
pending changes over the base in key order, so `cw-storage-plus` maps and
indexes work unchanged. Nothing reaches the base storage unless the caller
commits with `txn.prepare().commit(&mut base)`.

A smart query can wrap its read-only `deps.storage` in a transaction to run
execute code and return the result, then drop the transaction so nothing
persists.

The module is ported from the private `transactions` module of
[cw-multi-test](https://github.com/CosmWasm/cw-multi-test) v1.2.0, Copyright
Confio GmbH, licensed under the
[Apache License, Version 2.0](http://www.apache.org/licenses/LICENSE-2.0).
The module documentation lists the changes from the original.

## Sudo Types and Root Queries

The sudo types are generated from this checkout's `proto/nibiru/sudo/v1`
definitions. State field `roles` uses protobuf tag 3; tag 2 and the former
`contracts` field are reserved. The generated serde implementations accept
omitted default values and additional JSON fields from the native Wasm query
allowlist. Function `sudo::query_sudo_root` uses these types and the native
service path. Empty roots and query failures grant no authority.

Regenerate these types and run the Rust import cleanup with:

```sh
just rs proto-gen --path proto/nibiru/sudo/v1
cargo test --locked -p nibiru-std
```

Command `just rs proto-gen` accepts Buf input filters and uses the pinned
plugin in `proto/buf.gen.rs.yaml`. It uses in-tree protos, copies generated
packages only after successful generation, then runs command `proto_clean`.
It does not fetch the former external Cosmos SDK or modify Go dependencies.
