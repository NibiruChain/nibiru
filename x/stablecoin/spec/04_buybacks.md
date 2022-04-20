# Buybacks

**TLDR**: A user can call `Buyback` when there's too much collateral in the protocol according to the target collateral ratio. The user swaps NIBI for UST at a 0% transaction fee and the protocol burns the NIBI it buys from the user.

**`collRatio`**: The collateral ratio, or `collRatio` (sdk.Dec), is a value beteween 0 and 1 that determines what proportion of collateral and governance token is used during stablecoin mints and burns.

**`liqRatio`**: The liquidity ratio, or `liqRatio` (sdk.Dec), is a the proportion of the circulating NIBI liquidity relvative to the NUSD (stable) value.

### When is a "buyback" possible?

The protocol has too much collateral. Here, "protocol" refers to the module account of the `x/stablecoin` module, and "too much" refers to the difference between the `collRatio` and `liqRatio`. 

For example, if there's 10M NUSD in circulation, the price of UST collateral is 0.99 NUSD per UST, and the protocol has 5M UST, the `liqRatio` would be (5M * 0.99) / 10M = 0.495.   
Thus, if the collateral ratio, or `collRatio`, is less than 0.495, the an address with sufficient funds can call `Buyback`. 

### How does a buyback work?

The protocol has an excess of collateral. Buybacks allow users to sell NIBI to the protocol in exchange for NUSD, meaning that Nibiru Chain is effectively buying back its shares. After this transfer, the NIBI purchased by protocol is burned. This raises the value of the NIBI token for all of its hodlers. 

- Unlike `Recollateralize`, there is no bonus rate for this transaction.


#### Related Issues: 
- https://github.com/NibiruChain/nibiru/issues/117
