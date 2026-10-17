# Token vesting

`token-vesting` holds native-token and CW20 vesting accounts. The contract
tracks each account by address and denomination. Its Rust message types live in
[`src/msg.rs`](./src/msg.rs).

## Messages

### Instantiate

The instantiate message is empty.

```json
{}
```

### Register a native-token vesting account

The sender must attach exactly one native coin. The attached amount must match
`vesting_amount` in the schedule. Set `master_address` when another account
must be allowed to deregister the vesting account later.

```json
{
  "register_vesting_account": {
    "address": "nibi1beneficiary...",
    "master_address": "nibi1manager...",
    "vesting_schedule": {
      "linear_vesting": {
        "start_time": "1703772805",
        "end_time": "1703872805",
        "vesting_amount": "1000000"
      }
    }
  }
}
```

The contract also accepts `linear_vesting_with_cliff` schedules. That variant
adds `cliff_amount` and `cliff_time` to the fields above.

For CW20 tokens, send tokens to the contract with a CW20 `send` message. Its
base64-encoded hook message has the same shape as this payload, under
`register_vesting_account`.

### Deregister a vesting account

Only the account recorded as `master_address` may send this message. A native
denomination uses the `{"native":"..."}` form. A CW20 denomination uses
`{"cw20":"<contract address>"}`.

```json
{
  "deregister_vesting_account": {
    "address": "nibi1beneficiary...",
    "denom": { "native": "unibi" },
    "vested_token_recipient": "nibi1beneficiary...",
    "left_vesting_token_recipient": "nibi1manager..."
  }
}
```

Either recipient field may be `null`. The contract then sends vested funds to
the vesting-account owner and unvested funds to the master address.

### Claim vested tokens

The vesting-account owner claims all currently vested funds for the listed
denominations. Set `recipient` to `null` to send funds to the owner.

```json
{
  "claim": {
    "denoms": [{ "native": "unibi" }],
    "recipient": "nibi1recipient..."
  }
}
```

### Query vesting accounts

`vesting_account` returns the caller's stored vesting data for an address. Use
`start_after` and `limit` to paginate results.

```json
{
  "vesting_account": {
    "address": "nibi1beneficiary...",
    "start_after": null,
    "limit": 30
  }
}
```

## Schema

From this directory, command `cargo run --example token-vesting-schema`
generates JSON schemas in `schema/`. Treat the Rust message types and generated
schemas as authoritative when changing these examples.
