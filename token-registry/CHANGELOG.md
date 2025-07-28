# CHANGELOG - Nibiru/token-registry

- hotfix(token-registry): add USDC, USDT, and WETH (commit: 7b276baa046e6355b0282ae18fa41b6b11d0dab5)
- hotfix(token-registry): reset WNIBI address from
"0x0CaCF669f8446BeCA826913a3c6B96aCD4b02a97" to
"0x1429B38e58b97de646ACd65fdb8a4502c2131484" (commit: 7f523044a7f7fa887b55f297a4cf3c7debfc3fea)
- [#2208](https://github.com/NibiruChain/nibiru/pull/2208) - chore(token-registry): Add the xNIBI CW20 token and early wrapped ERC20 tokens for NIBI and AXV
- [#2317](https://github.com/NibiruChain/nibiru/pull/2317) - chore(evm): add MIM & USDC.arb to official erc20 token registry
- [#2318](https://github.com/NibiruChain/nibiru/pull/2318) - fix(token-registry): prevent overwriting embedded JSON files during tests.
  - Refactored `ERC20S` and `BANK_COINS` into `LoadERC20s()` and
  `LoadBankCoins()` functions that read from disk instead of relying on hardcoded
  variables. Updated `main.go` to load tokens from `official_erc20s.json` and
  `official_bank_coins.json` prior to parsing and writing.
  - This prevents test runs from overwriting source-controlled files when
  executing the script, resolving issues caused by `os.WriteFile` truncating the
  embedded JSON files on error.
- [#2341](https://github.com/NibiruChain/nibiru/pull/2341) - chore(token-registry): Add the USDa and sUSDa ERC20 from Avalon Finance
- [#2352](https://github.com/NibiruChain/nibiru/pull/2352) - chore(token-registry): Add bank coin versions of USDC and USDT from Stargate and LayerZero, and update ErisEvm.sol to fix redeem.
