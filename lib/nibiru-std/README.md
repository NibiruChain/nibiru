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

`UserAddr` serializes as canonical EIP-55 hex. It does not accept 32-byte
CosmWasm contract addresses.

## Stargate support

The crate supplies typed constructors for Nibiru and Cosmos SDK messages that
contracts send through `CosmosMsg::Stargate`, plus matching
`QueryRequest::Stargate` request and response types. See the generated API docs
for the available modules and message types.

## Documentation and license

- [API documentation](https://docs.rs/nibiru-std)
- [Nibiru source repository](https://github.com/NibiruChain/nibiru)

`nibiru-std` is licensed under the terms in [LICENSE](LICENSE).
