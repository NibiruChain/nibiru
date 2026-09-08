# Contracts Cookbook

This file describes the different messages that can be sent as queries or transactions
to the contracts of this repository with a description of the expected behavior.

- [Contracts Cookbook](#contracts-cookbook)
  - [Core token vesting](#core-token-vesting)
    - [Instantiate](#instantiate)
    - [Execute](#execute)
    - [Query](#query)
  - [4. Nibi Stargate](#4-nibi-stargate)
    - [4.1 Instantiate](#41-instantiate)
    - [4.2 Execute](#42-execute)
  - [5. Airdrop token vesting](#5-airdrop-token-vesting)
    - [5.1 Instantiate](#51-instantiate)
    - [5.2 Execute](#52-execute)
    - [7.3 Query](#73-query)
  - [8. Auto compounder](#8-auto-compounder)
    - [8.1 Instantiate](#81-instantiate)
    - [8.2 Execute](#82-execute)
      - [Admin functions](#admin-functions)
      - [Manager functions](#manager-functions)
    - [8.3 Query](#83-query)

## Core token vesting

This contract implements vesting accounts for the CW20 and native tokens.

### Instantiate

There's no instantiation message.

```js
{
}
```

### Execute

- **Receive**

```js
{
  "receive": {
    "sender": "cosmos1...",
    "amount": "1000000",
    "msg": "eyJ2ZXN0X2lkIjoxLCJ2ZXN0X3R5cGUiOiJ2ZXN0In0=",
  }
}
```

- **RegisterVestingAccount** registers a vesting account

```js
{
  "register_vesting_account": {
    "address": "cosmos1...",
    "master_address": "cosmos1...",
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

- **DeregisterVestingAccount** deregisters a vesting account

```js
{
  "deregister_vesting_account": {
    "address": "cosmos1...",
    "denom": "uusd",
    "vested_token_recipient": "cosmos1...", // address that will receive the vested tokens after deregistration. If None, tokens are received by the owner address.
    "left_vested_token_recipient": "cosmos1...", // address that will receive the left vesting tokens after deregistration.
  }
}
```

- **Claim** allows to claim vested tokens

```js
{
  "claim": {
    "denom": "uusd",
    "recipient": "cosmos1...",
  }
}
```

### Query

- **VestingAccount** returns the vesting account details for a given address.

```js
{
  "vesting_account": {
    "address": "cosmos1...",
  }
}
```

## 4. Nibi Stargate

This smart contract showcases usage examples for certain Nibiru-specific and Cosmos-SDK-specific.

### 4.1 Instantiate

There's no instantiation message.

```js
{
}
```

### 4.2 Execute

- **CreateDenom** creates a new denom

```js
{
  "create_denom": { "subdenom": "zzz" }
}
```

- **Mint** mints tokens

```js
{
  "mint": {
    "coin": { "amount": "[amount]", "denom": "tf/[contract-addr]/[subdenom]" },
    "mint_to": "[mint-to-addr]"
  }
}
```

- **Burn** burns tokens

```js
{
  "burn": {
    "coin": { "amount": "[amount]", "denom": "tf/[contract-addr]/[subdenom]" },
    "burn_from": "[burn-from-addr]"
  }
}
```

- **ChangeAdmin** changes the admin of a denom

```js
{
  "change_admin": {
    "denom": "tf/[contract-addr]/[subdenom]",
    "new_admin": "[ADDR]"
  }
}
```

## 7. Airdrop token vesting

This contract implements vesting accounts for the native tokens.

### 7.1 Instantiate

We need to specify admin and managers

```javascript
{
  "admin": "cosmos1...",
  "managers": ["cosmos1...", "cosmos1..."]
}
```

### 7.2 Execute

- **RewardUsers** registers several vesting contracts

```javascript
{
  "reward_users": {
    "rewards": [
      {
        "user_address": "cosmos1...",
        "vesting_amount": "1000000",
        "cliff_amount": "100000", // Only needed if vesting schedule is linear with cliff
      }
    ],
    "vesting_schedule": {
      "linear_vesting": {
        "start_time": "1703772805",
        "end_time": "1703872805",
        "vesting_amount": "0" // This amount does not matter
      }
    }
  }
}
```

- **DeregisterVestingAccount** deregisters a vesting account

```javascript
{
  "deregister_vesting_account": {
    "address": "cosmos1...",
    "vested_token_recipient": "cosmos1...", // address that will receive the vested tokens after deregistration. If None, tokens are received by the owner address.
    "left_vested_token_recipient": "cosmos1...", // address that will receive the left vesting tokens after deregistration.
  }
}
```

- **Claim** allows to claim vested tokens

```javascript
{
  "claim": {
    "recipient": "cosmos1...",
  }
}
```

### 7.3 Query

- **VestingAccount** returns the vesting account details for a given address.

```javascript
{
  "vesting_account": {
    "address": "cosmos1...",
  }
}
```

## 8. Auto compounder

This contract manages staking re-delegation processes securely, allowing for auto-compounding of staked funds.

### 8.1 Instantiate

We need to specify admin and managers

```javascript
{
  "admin": "cosmos1...",
  "managers": ["cosmos1...", "cosmos1..."]
}
```

### 8.2 Execute

#### Admin functions

- **SetAutoCompounderMode** sets the auto compounder mode

```javascript
{
  "set_auto_compounder_mode": {
    "mode": "true" // true or false
  }
}
```

- **Withdraw** allows to withdraw the funds from the contract

  ```javascript
  {
    "withdraw": {
      "amount": "1000000"
      "recipient": "cosmos1..."
    }
  }
  ```

- **unstakes** allows to unstake the funds from the contract

  ```javascript
  {
    "unstake": {
      "unstake_msgs": [
        {
          "validator": "cosmosvaloper1...",
          "amount": "1000000"
        },
        {
          "validator": "cosmosvaloper1...",
          "amount": "1000000"
        }
      ]
    }
  }
  ```

- **update managers** allows to update the managers of the contract

```javascript
{
  "update_managers": {
    "managers": ["cosmos1...", "cosmos1..."]
  }
}
```

#### Manager functions

- **stake** allows to stake the funds from the contract. The shares are normalized

```javascript
{
  "stake": {
    "stake_msgs": [
      {
        "validator": "cosmosvaloper1...",
        "share": "1000000"
      },
      {
        "validator": "cosmosvaloper1...",
        "share": "1000000"
      }
    ]
  },
  "amount": "1000000"
}
```

### 8.3 Query

- **auto compounder mode** returns wether the auto compounder mode is enabled or not

```javascript
{
  "auto_compounder_mode": {}
}
```

- **AdminAndManagers** returns the admin and managers of the contract

```javascript
{
  "admin_and_managers": {}
}
```