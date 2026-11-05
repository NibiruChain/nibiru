## Token Vesting

This contract implements vesting accounts for the native tokens.

Admin and managers are defined at the instantiation of the contracts. Both can
reward users and de-register vesting accounts, but only the admin can withdraw
the unallocated amount from the contract.

- [Token Vesting](#token-vesting)
  - [Master Operations](#master-operations)
    - [By admin and managers](#by-admin-and-managers)
    - [By admin only](#by-admin-only)
  - [Vesting Account Operations](#vesting-account-operations)
  - [Deployed Contract Info](#deployed-contract-info)
  - [Testing Against a Live Chain](#testing-against-a-live-chain)

### Master Operations

#### By admin and managers

```rust
  RewardUsers {
    rewards: Vec<RewardUserRequest>,
    vesting_schedule: VestingSchedule,
  },
```

This creates a set of vesting accounts for the given users.

```rust
  DeregisterVestingAccount {
    addresses: Vec<String>,
},
```

- DeregisterVestingAccount - deregister vesting account
  - It will compute `claimable_amount` and `left_vesting_amount` and send back to the contract admin.

#### By admin only

```rust
  Withdraw {
    amount: Uint128,
    recipient: String,
  },
```

This allows to get part or all of the unallocated amount from the contract and sends it to the `recipient`. Unallocated is equal to the
amount sent on instantiation minus the already rewarded to users.

### Vesting Account Operations

```rust
#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
#[serde(rename_all = "snake_case")]
pub enum ExecuteMsg {
    ////////////////////////
    /// VestingAccount Operations ///
    ////////////////////////
    Claim {
        recipient: Option<String>,
    },
}
```

- Sends newly vested token to the (`recipient` or `vesting_account`). The `claim_amount` is computed
  as (`vested_amount` - `claimed_amount`) and `claimed_amount` is updated to `vested_amount`.

  If everything is claimed, the vesting account is removed from the contract.

### Deployed Contract Info

TODO for mainnet/testnet

| Field         | Value |
| ------------- | ----- |
| code_id       | ...   |
| contract_addr | ...   |
| rpc_url       | ...   |
| chain_id      | ...   |

### Testing Against a Live Chain

You can test this smart contract on a live chain with the following script. It
requires `nibid` version 1 or 2.

```bash
WALLET=devnet_wallet
MANAGER_WALLET=validator
REWARDEE_WALLET=rewardee

CONTRACT_PATH=airdrop_token_vesting.wasm

TX_FLAG=(--keyring-backend test --output json -y --gas 100000000)

ADMIN=nibi1ds5zr8pv3dqnj4glmr73yef5j4wxq4p3wfxuhv
MANAGER=nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl
REWARDEE=nibi1qad9nvdzha9ugl5y28fw3h3ujjg5mrpydrsmeh

# Send funds to managers
nibid tx bank send $WALLET $MANAGER 100000unibi -y $TX_FLAG

# Deploy the contract
nibid tx wasm store "$CONTRACT_PATH" --from $WALLET $TX_FLAG

# you have the code_id from the output of the store tx
CODE_ID=2

# Instantiate the contract
cat << EOF | jq '.' | tee instantiate.json
{
    "admin": "$ADMIN",
    "managers": ["$MANAGER"]
}
EOF
JSON_DATA="$(<instantiate.json)"

nibid tx wasm instantiate $CODE_ID "$JSON_DATA" \
    --amount 100unibi \
    --label "airdrop_vesting" --admin $ADMIN "${TX_FLAG[@]}" \
    --from $WALLET

# You can get the contract address from the output of the instantiate tx
CONTRACT_ADDRESS=nibi1nc5tatafv6eyq7llkr2gv50ff9e22mnf70qgjlv737ktmt4eswrqugq26k

# reward a user
cat << EOF | jq '.' | tee reward.json
{
  "reward_users": {
    "master_address": "$ADMIN",
    "rewards": [
      {
        "user_address": "$REWARDEE",
        "vesting_amount": "50",
        "cliff_amount": "10"
      }
    ],
    "vesting_schedule": {
      "linear_vesting_with_cliff": {
        "start_time": "1708642800",
        "end_time": "1708729200",
        "cliff_time": "1708642800"
      }
    }
  }
}
EOF
JSON_DATA="$(<reward.json)"
nibid tx wasm execute $CONTRACT_ADDRESS "$JSON_DATA" --from $MANAGER_WALLET "${TX_FLAG[@]}"

# query the vesting account
nibid query wasm contract-state smart $CONTRACT_ADDRESS '{"vesting_accounts": {"address": "'$REWARDEE'"}}'


nibid query wasm contract-state smart $CONTRACT_ADDRESS '{"vesting_account": {"address": "'$REWARDEE'"}}' | jq .
{
  "data": {
    "address": "nibi1qad9nvdzha9ugl5y28fw3h3ujjg5mrpydrsmeh",
    "vestings": [
      {
        "master_address": "nibi1ds5zr8pv3dqnj4glmr73yef5j4wxq4p3wfxuhv",
        "vesting_denom": {
          "native": "unibi"
        },
        "vesting_amount": "50",
        "vested_amount": "0",
        "vesting_schedule": {
          "linear_vesting": {
            "start_time": "1708642800",
            "end_time": "1708729200",
            "vesting_amount": "50"
          }
        },
        "claimable_amount": "0"
      }
    ]
  }
}

# Withdraw the unallocated amount
cat << EOF | jq '.' | tee withdraw.json
{
  "withdraw": {
    "recipient": "$MANAGER",
    "amount": "50"
  }
}
EOF
JSON_DATA="$(<withdraw.json)"
nibid tx wasm execute $CONTRACT_ADDRESS "$JSON_DATA" --from $MANAGER_WALLET "${TX_FLAG[@]}"

# Deregister vesting accounts
cat << EOF | jq '.' | tee deregister.json
{
  "deregister_vesting_accounts": {
    "addresses": [
      "nibi1zrz9q4xr2u6tk0gtrzu7c7vyu53uyzp0cr9wgf",
      "nibi1dczse3mp5cg5jcjwxu5qreh277su4x7fku389c"
    ]
  }
}
EOF
JSON_DATA="$(<deregister.json)"
nibid tx wasm execute $CONTRACT_ADDRESS "$JSON_DATA" --from $MANAGER_WALLET "${TX_FLAG[@]}"
```
