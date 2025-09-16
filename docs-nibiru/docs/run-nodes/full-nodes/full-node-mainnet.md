---
order: 2
---

# Run a Full Node (Mainnet)

Guide to running a Nibiru mainnet full node: hardware requirements, installation
options, sync and upgrade workflows, chain initialization, and memory
optimization. {synopsis}

| In this Section | Description |
| --- | --- |
| [Running a node in a network that underwent upgrade(s)](#running-a-node-in-a-network-that-underwent-upgrade-s) | Four upgrade workflows—manual binary swaps, Cosmovisor automation, state sync, and snapshot downloads—with step‑by‑step instructions. |
| [Nibiru Mainnet Upgrade Heights](#nibiru-mainnet-upgrade-heights) | Chain ID, genesis version, and block heights paired with their corresponding release tags. |
| [Full Node: Pre‑requisites](#full-node-pre-requisites) | Minimum hardware specs, system updates, nibid installation options, and version verification. |
| [Init the Chain](#init-the-chain) | Initialization steps: genesis file retrieval, peer configuration, gas‑price settings, and optional fast‑sync methods. |
| [Memory Concerns](#memory-concerns) | Guidance on mitigating `rocksdb` memory growth by switching to `goleveldb`, with performance trade‑offs. |
| [Archive Nodes](#archive-nodes) | Guide to set up an archive node that retains the full historical blockchain state. |

## Running a node in a network that underwent upgrade(s)

When a network undergoes an update at a specific block height, the process of upgrading your node requires precise steps to ensure continuity and compatibility with the network's new state. The upgrade workflow can take several forms depending on the approach you choose. 

1. Sync with Manual Binary Swap:
This method involves a hands-on approach where you oversee the progression of
your node through network updates.
   
   Steps:

   - Initialize your node with Genesis Binary: Initialize your node using the binary version corresponding to the genesis block of the blockchain.
   
   - Monitor Upgrade Heights: Pay close attention to the block height as your node syncs with the network.
   
   - Stop and Swap Binaries Manually: Once your node reaches the designated upgrade height, stop your node and manually swap out the old binary with the new version tailored to that upgrade.
   
   - Resume Syncing: Restart your node with the updated binary, and continue this process each time an upgrade is reached.

2. Sync with [Cosmovisor](./cosmovisor.md):
Cosmovisor is a process manager that automates binary swapping during network
upgrades, simplifying the update process significantly.
   
   Steps:

   - Initialize your node with Genesis Binary: Initialize your node using the
   binary version corresponding to the genesis block of the blockchain.
   
   - Configure Cosmovisor: Set up Cosmovisor to monitor the block height and
   handle the automatic swapping of binaries when an upgrade point is hit.
   
   - Automated Upgrades: Let Cosmovisor manage the transition, providing a
   smoother and less error-prone upgrade experience as it will automatically
   change the binary when necessary.

3. State Sync:
State syncing allows a node to catch up quicker by getting a snapshot of the
state at a certain height, instead of syncing from the genesis block.
   
   Steps:

   - Current Binary Version: Start with the binary version that corresponds to
   the current network state rather than the genesis version.
   
   - Configure State Sync: Enable and configure state sync in your node's
   settings, allowing it to synchronize by jumping directly to a near-recent
   state.

4. Downloading a Snapshot:
This method involves downloading a complete data snapshot which can accelerate
the upgrade and syncing process.
   
   Steps:

   - Download Data Snapshot: Obtain a complete data snapshot from a trusted
   server. This typically includes all the data up to a recent block height.
   
   - Start with Current Binary: Use the current binary version compatible with
   the snapshot's block height, and start your node.
   
   - Resume Syncing: Your node will begin syncing from the snapshot's height,
   bypassing the earlier history for a faster setup.

<!-- ::UPGRADES -->

## Nibiru Mainnet Upgrade Heights

Chain ID: `cataclysm-1`

Genesis version:   [v1.0.0](https://github.com/NibiruChain/nibiru/releases/tag/v1.0.0)

Block `3225239`:   [v1.0.1](https://github.com/NibiruChain/nibiru/releases/tag/v1.0.1)

Block `3539699`:   [v1.0.2](https://github.com/NibiruChain/nibiru/releases/tag/v1.0.2)

Block `4088799`:   [v1.0.3](https://github.com/NibiruChain/nibiru/releases/tag/v1.0.3)

Block `4447094`:   [v1.1.0](https://github.com/NibiruChain/nibiru/releases/tag/v1.1.0)

Block `4804662`:   [v1.2.0](https://github.com/NibiruChain/nibiru/releases/tag/v1.2.0)

Block `6281429`:   [v1.3.0](https://github.com/NibiruChain/nibiru/releases/tag/v1.3.0)

Block `7457147`:   [v1.4.0](https://github.com/NibiruChain/nibiru/releases/tag/v1.4.0)

Block `8375044`:   [v1.5.0](https://github.com/NibiruChain/nibiru/releases/tag/v1.5.0)

Block `18538950`:   [v2.0.0-p1](https://github.com/NibiruChain/nibiru/releases/tag/v2.0.0-p1)

Block `19562174`:   [v2.1.0](https://github.com/NibiruChain/nibiru/releases/tag/v2.1.0)

Block `20937412`:   [v2.2.0-p1](https://github.com/NibiruChain/nibiru/releases/tag/v2.2.0-p1)

Block `22301853`:   [v2.3.0](https://github.com/NibiruChain/nibiru/releases/tag/v2.3.0)

Block `24130375`:   [v2.4.0](https://github.com/NibiruChain/nibiru/releases/tag/v2.4.0)

Block `24477075`:   [v2.5.0](https://github.com/NibiruChain/nibiru/releases/tag/v2.5.0)


<!-- ::/UPGRADES -->

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

Option 1: Use this version if you plan to use state-sync or data snapshot
```bash
curl -s https://get.nibiru.fi/@v2.0.0-p1! | bash
```

Option 2: Use this version if you plan to sync from genesis block; you will need to swap it to the current one at the upgrade height (either manually or with Cosmovisor)
```bash
curl -s https://get.nibiru.fi/@v1.0.0! | bash
```

### Verify nibid version

```bash
nibid version
# Should output v1.0.0 or 2.0.0-p1 depending on chosen approach
```

---

## Init the Chain

1. Init the chain

    ```bash
    NETWORK=cataclysm-1
    nibid init <moniker-name> --chain-id=$NETWORK --home $HOME/.nibid
    ```

2. Copy the genesis file to the `$HOME/.nibid/config` folder.

    You can get genesis from our networks endpoint with:

    ```bash
    NETWORK=cataclysm-1
    curl -s https://networks.nibiru.fi/$NETWORK/genesis > $HOME/.nibid/config/genesis.json
    ```

    Or you can download it from the Tendermint RPC endpoint.

    ```bash
    curl -s https://rpc.nibiru.fi/genesis | jq -r .result.genesis > $HOME/.nibid/config/genesis.json
    ```

    **(Optional) Verify Genesis File Checksum**

    ```bash
    shasum -a 256 $HOME/.nibid/config/genesis.json

    # 18de90cb67cd14464b65211f9b6bcdfe4fb2d059c2cfcffbf72bce365fc536c5 $HOME/.nibid/config/genesis.json
    ```
3. Update persistent peers list in the configuration file `$HOME/.nibid/config/config.toml`.

    ```bash
    NETWORK=cataclysm-1
    sed -i 's|\<persistent_peers\> =.*|persistent_peers = "'$(curl -s https://networks.nibiru.fi/$NETWORK/peers)'"|g' $HOME/.nibid/config/config.toml
    ```    

4. Set minimum gas prices

    ```bash
    sed -i 's/minimum-gas-prices =.*/minimum-gas-prices = "0.025unibi"/g' $HOME/.nibid/config/app.toml
    ```

5. (Optional) Configure one of the following options to catch up faster with the network

    Option 1: Configure state-sync

    ```bash
    NETWORK=cataclysm-1
    config_file="$HOME/.nibid/config/config.toml"

    sed -i "s|enable =.*|enable = true|g" "$config_file"
    sed -i "s|rpc_servers =.*|rpc_servers = \"$(curl -s https://networks.nibiru.fi/$NETWORK/rpc_servers)\"|g" "$config_file"
    sed -i "s|trust_height =.*|trust_height = \"$(curl -s https://networks.nibiru.fi/$NETWORK/trust_height)\"|g" "$config_file"
    sed -i "s|trust_hash =.*|trust_hash = \"$(curl -s https://networks.nibiru.fi/$NETWORK/trust_hash)\"|g" "$config_file"
    ```
    
   Option 2: Download and extract data snapshot

   You can check [available snapshots list for Nibiru Mainnet](https://networks.nibiru.fi/cataclysm-1/snapshots) to locate the snapshot with the date and type that you need

   ```bash
   curl -o cataclysm-1-<timestamp>-<type>-<db-backend>.tar.gz https://storage.googleapis.com/cataclysm-1-snapshots/cataclysm-1-<timestamp>-<type>-<db-backend>.tar.gz
   tar -zxvf cataclysm-1-<timestamp>-<type>-<db-backend>.tar.gz -C $HOME/.nibid/
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

---

## Memory Concerns

If you are using `rocksdb` as the database backend, you may notice the memory keeps increasing. This is a known issue with `rocksdb`. To mitigate this, you can switch to using `goleveldb` as the database backend (for either cosmos-sdk or cometbft, or both).

```bash
sed -i "s|db_backend =.*|db_backend=\"goleveldb\"|g" "$HOME/.nibid/config/config.toml"

sed -i "s|app-db-backend =.*|app-db-backend=\"goleveldb\"|g" "$HOME/.nibid/config/app.toml"
```

Note that this will decrease your RPC query performance, but resolve memory issues.

## Archive Nodes

The process for setting up archive node is almost identical to how you set up a
default full node, which has pruning enabled. To make your full node an archive
node, you'll need to download one of the archive snapshots and set `pruning =
nothing` in the `$HOME/.nibid/config/app.toml` file.

1. Init the chain

    ```bash
    NETWORK=cataclysm-1
    nibid init <moniker-name> --chain-id=$NETWORK --home $HOME/.nibid
    ```

2. Copy the genesis file to the `$HOME/.nibid/config` folder.

    You can get genesis from our networks endpoint with:

    ```bash
    NETWORK=cataclysm-1
    curl -s https://networks.nibiru.fi/$NETWORK/genesis > $HOME/.nibid/config/genesis.json
    ```

    Or you can download it from the Tendermint RPC endpoint.

    ```bash
    curl -s https://rpc.nibiru.fi/genesis | jq -r .result.genesis > $HOME/.nibid/config/genesis.json
    ```

    **(Optional) Verify Genesis File Checksum**

    ```bash
    shasum -a 256 $HOME/.nibid/config/genesis.json

    # 18de90cb67cd14464b65211f9b6bcdfe4fb2d059c2cfcffbf72bce365fc536c5 $HOME/.nibid/config/genesis.json
    ```
3. Update persistent peers list in the configuration file `$HOME/.nibid/config/config.toml`.

    ```bash
    NETWORK=cataclysm-1
    sed -i 's|\<persistent_peers\> =.*|persistent_peers = "'$(curl -s https://networks.nibiru.fi/$NETWORK/peers)'"|g' $HOME/.nibid/config/config.toml
    ```    

4. Set minimum gas prices

    ```bash
    sed -i 's/minimum-gas-prices =.*/minimum-gas-prices = "0.025unibi"/g' $HOME/.nibid/config/app.toml
    ```

5. Download and extract data snapshot

   You can check [available snapshots list for Nibiru Mainnet](https://networks.nibiru.fi/cataclysm-1/snapshots) to locate the snapshot with the date and type that you need

   ```bash
   curl -o cataclysm-1-<timestamp>-<type>-<db-backend>.tar.gz https://storage.googleapis.com/cataclysm-1-snapshots/cataclysm-1-<timestamp>-<type>-<db-backend>.tar.gz
   tar -zxvf cataclysm-1-<timestamp>-<type>-<db-backend>.tar.gz -C $HOME/.nibid/
   ```

6. Disable pruning

   ```bash
   sed -i 's/pruning =.*/pruning = "nothing"/g' $HOME/.nibid/config/app.toml
   ```

7. Start your node (choose one of the options)

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
