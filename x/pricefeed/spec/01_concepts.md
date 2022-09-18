# Concepts

Nibiru’s price feeds are updated the end of every block from messages posted by `oracle` accounts.

## Oracle account

 An `oracle` is simply a whitelisted address that is able to send a special `post_price` message on-chain that includes the asset pair, price, and an expiry time.

## TWAP

This message gets wrapped into a transaction to be included in the current block. At the end of each block, the median of all unexpired, oracle-posted prices is computed for each asset pair and stored by the Nibiru blockchain in a snapshot. Finally, the time-weighted average price (TWAP) is taken to be the “official” price for the pair. This is currently implemented in the `x/pricefeed` module.
