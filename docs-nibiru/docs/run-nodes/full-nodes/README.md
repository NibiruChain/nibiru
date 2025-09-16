---
order: 1
---

# Run a Full Node (Testnet)

Testnets are testing instances of the Nibiru blockchain. Testnet tokens are
separate and distinct from real assets. In order to join a network, you'll need
to use its corresponding version of the binary to [run a full
node](./node-daemon.md).{synopsis}

| In this Section | Description |
| --- | --- |
| [Available Networks](#available-networks) | Overview of Nibiru testnets: chain IDs, descriptions, genesis versions, upgrade history, and status. |
| [Running a node in a network that underwent upgrade(s)](#running-a-node-in-a-network-that-underwent-upgrades) | Four upgrade workflows: manual binary swaps, Cosmovisor automation, state sync, and snapshot downloads. |
| [Block Explorers](#block-explorers) | Links to block explorers for viewing transactions, blocks, addresses, and other on‑chain data. |
| [Full Node: Pre‑requisites](#full-node-pre-requisites) | Hardware requirements, system updates, nibid installation options, and version verification. |
| [Init the Chain](#init-the-chain) | Steps to initialize: download genesis file, configure peers, set gas prices, and enable optional fast‐sync methods. |
| [Next Steps](#next-steps) | Tips on validator participation and pointers to related docs. |
| [Example `nibid` commands](#example-nibid-commands) | Sample CLI commands for querying balances and exploring module references. |

## Available Networks

On Nibiru Chain, the term "network" encompasses a collective system of
[interconnected nodes](../../learn/faq/nodes-faq.md). These nodes, which are
essentially individual computers or servers, collaboratively maintain and
validate the blockchain's ongoing operations.

You can find a table of each Nibiru testnet and its current status below.

| Chain ID | Description | Genesis Version | Upgrade History | Status |
| -------- | ----------- | --------------- | --------------- | ------ |
| nibiru-testnet-1 | Nibiru Chain First Permanent Testnet | [v1.0.0](https://github.com/NibiruChain/nibiru/releases/tag/v1.0.0) | block 48759 - [v1.1.0](https://github.com/NibiruChain/nibiru/releases/tag/v1.1.0)<br> block 3067771 - [v1.2.0](https://github.com/NibiruChain/nibiru/releases/tag/v1.2.0)<br> block 3095130 - [v1.3.0-rc1](https://github.com/NibiruChain/nibiru/releases/tag/v1.3.0-rc1)<br> block 3537069 - [v1.3.0](https://github.com/NibiruChain/nibiru/releases/tag/v1.3.0)<br> block 4566080 - [v1.4.0](https://github.com/NibiruChain/nibiru/releases/tag/v1.4.0)<br> block 5117602 - [v1.5.0](https://github.com/NibiruChain/nibiru/releases/tag/v1.5.0)<br> block 7280214 - [v2.0.0-rc.1](https://github.com/NibiruChain/nibiru/releases/tag/v2.0.0-rc.1)<br> block 8825300 - [v2.0.0-rc.9](https://github.com/NibiruChain/nibiru/releases/tag/v2.0.0-rc.9)<br> block 10354077 - [v2.0.0-rc.14](https://github.com/NibiruChain/nibiru/releases/tag/v2.0.0-rc.14)<br> block 11253562 - [v2.0.0-rc.18](https://github.com/NibiruChain/nibiru/releases/tag/v2.0.0-rc.18)<br> block 11353204 - [v2.0.0-rc.19](https://github.com/NibiruChain/nibiru/releases/tag/v2.0.0-rc.19) | 🚫 Deprecated |
| nibiru-testnet-2 | Nibiru Chain Second Permanent Testnet. Hard Fork of nibiru-testnet-1. Has IBC channels and [web faucet](https://app.nibiru.fi/faucet) | [v2.0.0](https://github.com/NibiruChain/nibiru/releases/tag/v2.0.0) | block 305570 - [v2.1.0](https://github.com/NibiruChain/nibiru/releases/tag/v2.1.0)<br> block 1213333 - [v2.3.0](https://github.com/NibiruChain/nibiru/releases/tag/v2.3.0)<br> block 1324767 - [v2.4.0-rc1](https://github.com/NibiruChain/nibiru/releases/tag/v2.4.0-rc1)<br> block 1927435 - [v2.5.0-rc1](https://github.com/NibiruChain/nibiru/releases/tag/v2.5.0-rc1) | ⚡ Active |

::: tip
See [Nibiru Networks](../../dev/networks/README.md) for RPC information.
:::

## Running a node in a network that underwent upgrade(s)

When a network undergoes an update at a specific block height, the process of
upgrading your node requires precise steps to ensure continuity and compatibility
with the network's new state. The upgrade workflow can take several forms
depending on the approach you choose.

1. Sync with Manual Binary Swap This method involves a hands-on approach where
   you oversee the progression of your node through network updates.

   Steps: Initialize your node with Genesis Binary: Initialize your node using
   the binary version corresponding to the genesis block of the blockchain.

   Monitor Upgrade Heights: Pay close attention to the block height as your node
   syncs with the network.

   Stop and Swap Binaries Manually: Once your node reaches the designated upgrade
   height, stop your node and manually swap out the old binary with the new
   version tailored to that upgrade.

   Resume Syncing: Restart your node with the updated binary, and continue this
   process each time an upgrade is reached.

2. Sync with [Cosmovisor](./cosmovisor.md) Cosmovisor is a process manager that
   automates binary swapping during network upgrades, simplifying the update
process significantly.

   Steps: Initialize your node with Genesis Binary: Initialize your node using
   the binary version corresponding to the genesis block of the blockchain.

   Configure Cosmovisor: Set up Cosmovisor to monitor the block height and handle
   the automatic swapping of binaries when an upgrade point is hit.

   Automated Upgrades: Let Cosmovisor manage the transition, providing a smoother
   and less error-prone upgrade experience as it will automatically change the
   binary when necessary.

3. State Sync State syncing allows a node to catch up quicker by getting a
   snapshot of the state at a certain height, instead of syncing from the genesis
block.

   Steps: Current Binary Version: Start with the binary version that corresponds
   to the current network state rather than the genesis version.

   Configure State Sync: Enable and configure state sync in your node's settings,
   allowing it to synchronize by jumping directly to a near-recent state.

4. Downloading a Snapshot This method involves downloading a complete data
   snapshot which can accelerate the upgrade and syncing process.

   Steps: Download Data Snapshot: Obtain a complete data snapshot from a trusted
   server. This typically includes all the data up to a recent block height.

   Start with Current Binary: Use the current binary version compatible with the
   snapshot's block height, and start your node.

   Resume Syncing: Your node will begin syncing from the snapshot's height,
   bypassing the earlier history for a faster setup.

## Block Explorers

You can see current status of the blockchain on a block explorer. Explorers allow
you to search through transactions, blocks, wallet addresses, and other on-chain
data.

- [Nibiru Explorer | NodesGuru](https://nibiru.explorers.guru/)
- [Nibiru Explorer | NodeStake](https://explorer.nodestake.top/nibiru)
- [Nibiru Explorer | Kjnodes](https://explorer.kjnodes.com/nibiru)
- [Nibiru Explorer | Nibiru Chain Team](https://explorer.nibiru.fi/).

<!-- ## Blockchain Parameters -->
<!---->
<!-- | Chain ID     | Block Time  | Staking Unbonding Time | Governance Voting Period | -->
<!-- | ------------ | ----------- | ---------------------- | ------------------------ | -->
<!-- | nibiru-testnet-1 | 1.5 seconds | 21 days                | 2 hours                  | -->
<!-- | nibiru-itn-2 | 1.5 seconds | 21 days                | 4 days                   | -->
<!-- | nibiru-itn-1 | 1.5 seconds | 24 hours               | 2 hours                  | -->

---

## Full Node: Pre-requisites

### Minimum hardware requirements

- 4CPU
- 16GB RAM
- 1TB of disk space (SSD)

### Update the system

```bash
sudo apt update && sudo apt upgrade --yes
```

### Install nibid

Option 1: Use this version if you plan to sync from genesis block; you will need to swap it to the current one at the upgrade height (either manually or with Cosmovisor)

```bash
curl -s https://get.nibiru.fi/@v2.0.0! | bash
```

Option 2: Use this version if you plan to use state-sync or data snapshot

```bash
curl -s https://get.nibiru.fi/@v2.5.0-rc.1! | bash
```

### Verify nibid version

```bash
nibid version
# Should output v2.0.0 or v2.5.0-rc.1 depending on chosen approach
```

---

## Init the Chain

1. Init the chain

    ```bash
    nibid init <moniker-name> --chain-id=nibiru-testnet-2 --home $HOME/.nibid
    ```

2. Copy the genesis file to the `$HOME/.nibid/config` folder.

    You can get genesis from our cloud storage:

    ```bash
    NETWORK=nibiru-testnet-2
    curl -s https://storage.googleapis.com/$NETWORK-snapshots/$NETWORK.json > $HOME/.nibid/config/genesis.json
    ```

    **Note:** `nibiru-testnet-2` originates from a hard fork of `nibiru-testnet-1`, which means its genesis file includes the full state snapshot and is therefore quite large (~1.3 GB).

    **(Optional) Verify Genesis File Checksum**

    ```bash
    shasum -a 256 $HOME/.nibid/config/genesis.json

    # 459c57b57779d34ca8cc0cb0205e19d2d0940ff67cdef545b40c861d2e3b9ce1 $HOME/.nibid/config/genesis.json
    ```

3. Update persistent peers list in the configuration file `$HOME/.nibid/config/config.toml`.

    ```bash
    NETWORK=nibiru-testnet-2
    sed -i 's|\<persistent_peers\> =.*|persistent_peers = "'$(curl -s https://networks.testnet.nibiru.fi/$NETWORK/peers)'"|g' $HOME/.nibid/config/config.toml
    ```

4. Set minimum gas prices

    ```bash
    sed -i 's/minimum-gas-prices =.*/minimum-gas-prices = "0.025unibi"/g' $HOME/.nibid/config/app.toml
    ```

5. (Optional) Configure one of the following options to catch up faster with the network

    Option 1: Setup state-sync

    ```bash
    NETWORK=nibiru-testnet-2
    config_file="$HOME/.nibid/config/config.toml"

    sed -i "s|enable =.*|enable = true|g" "$config_file"
    sed -i "s|rpc_servers =.*|rpc_servers = \"$(curl -s https://networks.testnet.nibiru.fi/$NETWORK/rpc_servers)\"|g" "$config_file"
    sed -i "s|trust_height =.*|trust_height = \"$(curl -s https://networks.testnet.nibiru.fi/$NETWORK/trust_height)\"|g" "$config_file"
    sed -i "s|trust_hash =.*|trust_hash = \"$(curl -s https://networks.testnet.nibiru.fi/$NETWORK/trust_hash)\"|g" "$config_file"
    ```

   Option 2: Download and extract data snapshot

   You can check [available snapshots list for nibiru-testnet-2](https://networks.testnet.nibiru.fi/nibiru-testnet-2/snapshots) to locate the snapshot with the date and type that you need

   ```bash
   curl -o nibiru-testnet-2-<timestamp>-<type>.tar.gz https://storage.googleapis.com/nibiru-testnet-2-snapshots/nibiru-testnet-2-<timestamp>-<type>.tar.gz
   tar -zxvf nibiru-testnet-2-<timestamp>-<type>.tar.gz -C $HOME/.nibid/
   ```

6. Start your node (choose one of the options)

    Option 1: **Systemd + Systemctl**

    After defining a [service file for use with `systemctl`](./systemctl.md), you can execute:

    ```bash
    sudo systemctl start nibiru
    ```

    Option 2: **Cosmovisor**

    After defining a [service file for use with `cosmovisor`](./cosmovisor.md), you can execute:

    ```bash
    sudo systemctl start cosmovisor-nibiru
    ```

    Option 3: Without a daemon

    ```bash
    nibid start
    ```

7. Request tokens from the [Web Faucet for nibiru-testnet-2](https://app.nibiru.fi/faucet) if required.

    To create a local key pair, you may use the following command:

    ```bash
    nibid keys add <key-name>
    ```
    <!-- ```bash
    FAUCET_URL="https://faucet.testnet-1.nibiru.fi/"
    ADDR="..." # your address
    curl -X POST -d '{"address": "'"$ADDR"'", "coins": ["11000000unibi","100000000unusd","100000000uusdt"]}' $FAUCET_URL
    ``` -->

    <!-- Please note, that current daily limit for the Web Faucet is 11NIBI (`11000000unibi`) and 100 NUSD (`100000000unusd`). -->

---

## Next Steps

::: tip
See the [validator docs](../validators/README.md) on how to participate as a validator.
:::

## Example `nibid` commands

Ex: query an account's balance

```bash
nibid query bank balances nibi1gc6vpl9j0ty8tkt53787zps9ezc70kj88hluw4
```

For the full list of `nibid` commands, see:

- The [`nibid` CLI introduction](../../dev/cli/README.md)
- Nibiru [Module Reference](../../arch/README.md)
