---
order: 2
---

# Become a Validator (Mainnet)

Instructions for running a validator node for Nibiru Mainnet{synopsis}

First, please follow the [instructions to join the Mainnet as a full node](../full-nodes/full-node-mainnet.md).

::: tip
We recommend saving the `chain-id` into your `client.toml`.
This prevents you from having to pass the `chain-id` flag with every CLI command.

```bash
NETWORK=cataclysm-1

nibid config chain-id $NETWORK
```

:::

## Send a `create-validator` transaction

Create a new key pair if you haven't done this yet:

```bash
nibid keys add <key-name>
```

or add an existing account with a known mnemonic:

```bash
nibid keys add <key-name> --recover
```

Any participant in the network can become a validator by sending a `create-validator` transaction. This involved specifying the following parameters.

- **`commission-max-change-rate`**: The maximum daily increase of the validator commission. This parameter is fixed cannot be changed after the `create-validator` transaction is processed.
- **`commission-max-rate`**: The maximum commission rate that this validator can charge. This parameter is fixed and cannot be changed after the `create-validator` transaction is processed.
- **`commission-rate`**: The commission rate on block rewards and fees charged to delegators. **Note**: The `commission-rate` value must adhere to the following invariants:
  - Must be between 0 and the validator's `commission-max-rate`
  - Must not exceed the validator's `commission-max-change-rate`, which is the maximum percentage point change rate **per day**. In other words, a validator can only change its commission once per day and within `commission-max-change-rate` bounds.
- **`min-self-delegation`**: Minimum amount of NIBI the validator requires to have bonded at all time. If the validator's self-delegated stake falls below this limit, their validator gets jailed and kicked out of the active validator set.
- **`details`**: The validator description. More information is given on this in the next section.
- **`pubkey`**: A validator's Tendermint pubkey is associated with a private key used to sign "prevotes" and "precommits". It is prefixed with `nibivalconspub` and found by executing `nibid tendermint show-validator`.
- **`moniker`**: A (not necessarily unique) name for the validator.

After a validator is created, NIBI holders can delegate NIBI to the validator, effectively adding stake to the validator's pool. The total stake of an address is the combination of NIBI bonded by delegators and NIBI self-bonded by the validator.

Of all of the validators that send a `staking create-validator` transaction, those with the highest total stake are designated members of the validator set. If a validator's total stake falls too low, that validator loses its validator privileges and becomes unable to participate in consensus or generate rewards. Over time, the maximum number of validators may be increased via on-chain governance proposals.

```bash
NETWORK=cataclysm-1

nibid tx staking create-validator \
--amount 10000000unibi \
--commission-max-change-rate "0.1" \
--commission-max-rate "0.20" \
--commission-rate "0.1" \
--min-self-delegation "1" \
--details "put your validator description there" \
--pubkey=$(nibid tendermint show-validator) \
--moniker <your_moniker> \
--chain-id $NETWORK \
--gas-prices 0.025unibi \
--from <key-name>
```

You can verify your node is in the validator set status by viewing the [block explorer](https://explorer.nibiru.fi/cataclysm-1/staking)

### Editing the public description

You can edit your validator's public description. This info is to identify your validator, and will be relied on by delegators to decide which validators to stake to. Make sure to provide input for every flag below. If a flag is not included in the command the field will default to empty (`--moniker` defaults to the machine name) if the field has never been set or remain the same if it has been set in the past.

The `<key_name>` passed as the value for the `--from` flag specifies which validator you are editing. If you choose to not include certain flags, remember that the `--from` in particular must be included to identify which validator to update.

The `--identity` can be used as to verify identity with systems like Keybase or UPort. When using with Keybase `--identity` should be populated with a 16-digit string that is generated with a [keybase.io](https://keybase.io) account. It's a cryptographically secure method of verifying your identity across multiple online networks. The Keybase API allows us to retrieve your Keybase avatar. This is how you can add a logo to your validator profile.

## States for validators  

After a validator is created with a `create-validator` transaction, the validator is in one of three states:

- `in validator set`: Validator is in the active set and participates in consensus. The validator is earning rewards and can be slashed for misbehavior.
- `jailed`: Validator misbehaved and is in jail, i.e. outside of the validator set.

  - If the jailing is due to being offline for too long (i.e. having missed more than `95%` out of the last `10,000` blocks), the validator can send an `unjail` transaction in order to re-enter the validator set.
  - If the jailing is due to double signing, the validator cannot unjail.

- `unbonded`: Validator is not in the active set, and therefore not signing blocks. The validator cannot be slashed and does not earn any reward. It is still possible to delegate NIBI to an unbonded validator. Undelegating from an `unbonded` validator is immediate, meaning that the tokens are not subject to the unbonding period.

## Unjailing a validator

When a validator is "jailed" for downtime, you must submit a `slashing unjail` transaction from the operator account in order to be able to get block proposer rewards again (depends on the zone fee distribution).

```bash
nibid tx slashing unjail \
  --from=<key_name> \
  --chain-id=<chain_id>
```

## Confirming your validator is running

Your validator is active if the following command returns anything:

```bash
nibid query tendermint-validator-set | grep "$(nibid tendermint show-address)"
```

You should now see your validator in one of Nibiru explorers. You are looking for the `bech32` encoded `address` in the `~/.nibid/config/priv_validator.json` file.
