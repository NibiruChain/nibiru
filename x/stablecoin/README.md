The stablecoin module is responsible for minting and burning USDM.

# Minting Stablecoins

In a new terminal, run the following command:

```sh
// send a transaction to mint stablecoin
$ matrixd tx stablecoin mint 1000validatortoken --from validator --home data/localnet --chain-id localnet

// query the balance
$ matrixd q bank balances cosmos1zaavvzxez0elundtn32qnk9lkm8kmcszzsv80v
