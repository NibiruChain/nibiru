# `x/stablecoin`        <!-- omit in toc -->


The stablecoin module is responsible for minting and burning USDM, maintenance of USDM's price stability, and orchestration of Matrix Protocol's collateral ratio.

#### Table of Contents
- Messages and Events - [[00_msgs_and_events.md]](00_msgs_and_events.md): [description]
- Keepers and Parameters - [[01_msgs_and_events.md]](01_msgs_and_events.md): [description]
- Module Accounts of `x/stablecoin` - [[02_stablecoin_accounts.md]](02_stablecoin_accounts.md): [description]
- Recollateralize - [[03_recollateralize.md]](03_recollateralize.md): **Recollateralize** is a function that incentivizes the caller to add up to the amount of collateral needed to reach some **target collateral ratio** (`collRatioTarget`). Recollateralize checks if the USD value of collateral in the protocol is below the required amount defined by the current collateral ratio.
- Buybacks - [[04_buybacks.md]](04_buybacks.md): [description]
- [CLI Usage Guide](#cli-usage-guide)
  - [Minting Stablecoins](#minting-stablecoins)

---

## Module Accounts of `x/stablecoin`

Treasury: TODO docs

StableEF: TODO docs

---

# CLI Usage Guide

## Minting Stablecoins

In a new terminal, run the following command:

```sh
// send a transaction to mint stablecoin
$ matrixd tx stablecoin mint 1000validatortoken --from validator --home data/localnet --chain-id localnet

// query the balance
$ matrixd q bank balances cosmos1zaavvzxez0elundtn32qnk9lkm8kmcszzsv80v
