# Changelog - Nibiru/evm-core-ts 

Record of pull requests and differences between versions for
the `@nibiruchain/evm-core` package on `npm`.

## [0.1.x]

- v0.1.0: [#2339](https://gittub.com/NibiruChain/nibiru/pull/2339) - feat:
Publish new version for `@nibiruchain/solidity@0.0.7`.
  - Add ErisEvm.sol contract runner and constants from the contract on Nibiru
  mainnet. Adds `ADDR_ERIS_EVM` constant and an `ERIS_CONST` helper object with
  related addresses and bank denominations.
  - Adds support for `latestAnswer()` in the `ChainlinkAggregatorV3Interface`
  type.
  - Renamed `wnibiCaller` and `erc20Caller` to `wnibiRunner` and `erc20Runner` to
    more consistency with the ethers v6 `ContractRunner` convention.

## [0.0.x]

- v0.0.9: [#2334](https://gittub.com/NibiruChain/nibiru/pull/2334) - feat:
Publish new version for `@nibiruchain/solidity@0.0.6`, which updates
`NibiruOracleChainLinkLike.sol` to have additional methods used by Aave.
- v0.0.8: [#2309](https://gittub.com/NibiruChain/nibiru/pull/2309) - feat:
Publish new version for `@nibiruchain/solidity@0.0.5`, which includes
`ERC20MinterWithMetadataUpdates` and new additions to the `FunToken` precompile.
- v0.0.7: [#2238](https://gittub.com/NibiruChain/nibiru/pull/2238) - fix: Set `wnibiCaller` to use a fixed address for mainnet as its default argument.
- v0.0.6: [#2238](https://github.com/NibiruChain/nibiru/pull/2238) - feat: Add WNIBI caller 
- v0.0.5: [#2231](https://github.com/NibiruChain/nibiru/pull/2231) - chore(evm-core-ts): Add publint linting for npm package compatibility on more environments
- v0.0.4: [#2229](https://github.com/NibiruChain/nibiru/pull/2229) - Improve compatibility with older "moduleResolution" settings. It is recommended to use "bundler".
- v0.0.2: [#2204](https://github.com/NibiruChain/nibiru/pull/2204) - Add ERC20
caller to the "ethers" export.
- v0.0.1: [#2197](https://github.com/NibiruChain/nibiru/pull/2197) - Export
precompile addresses, ABIs, and ethers v6 adapters for strongly typed contracts.
This includes the FunToken, Wasm, and Nibiru Oracle precompiled contracts.
