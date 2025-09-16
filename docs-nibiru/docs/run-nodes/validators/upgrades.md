---
order: 5
---

# Binary Upgrades

Every now and then, the Nibiru blockchain will undergo a chain upgrade, which means validators will have to swap the currently running binary for the new binary version at a given block height. {synopsis}

When an upgrade governance proposal is passed, it will have a version identifier and an upgrade block height. At the upgrade block height, the chain will halt. At this point, validators are responsible for swapping their binary with the new version.

1. Download new binary (e.g. vX.Y.Z)

    ```bash
    curl -s https://get.nibiru.fi/@vX.Y.Z! | bash
    ```

2. Restart the binary

    If you're using systemctl,

    ```bash
    sudo systemctl restart nibiru
    ```

## Cosmovisor

[Cosmovisor](https://docs.cosmos.network/main/tooling/cosmovisor) is an open source tool that automatically handles the upgrade process for you. It listens for on-chain governance proposals, downloads the new binary for you, and swaps the binaries at the correct height.

Please see our [Cosmovisor setup page](../full-nodes/cosmovisor.md) on how to configure it.
