<!--
Guiding Principles:

Changelogs are for humans, not machines.
There should be an entry for every single version.
The same types of changes should be grouped.
Versions and sections should be linkable.
The latest version comes first.
The release date of each version is displayed.
Mention whether you follow Semantic Versioning.

Usage:

Change log entries are to be added to the Unreleased section under the
appropriate stanza (see below). Each entry should ideally include a tag and
the Github issue reference in the following format:

* (<tag>) \#<issue-number> message

The issue numbers will later be link-ified during the release process so you do
not have to worry about including a link manually, but you can if you wish.

Types of changes (Stanzas):

"Features" for new features.
"Improvements" for changes in existing functionality.
"Deprecated" for soon-to-be removed features.
"Bug Fixes" for any bug fixes.
"API Breaking" for breaking CLI commands and REST routes used by end-users.
"State Machine Breaking" for any changes that result in a different AppState given same genesisState and txList.
Ref: https://keepachangelog.com/en/1.0.0/
-->

# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### State Machine Breaking

#### For next mainnet version

- [#1766](https://github.com/NibiruChain/nibiru/pull/1766) - refactor(app-wasmext)!: remove wasmbinding `CosmosMsg::Custom` bindings.
- [#1776](https://github.com/NibiruChain/nibiru/pull/1776) - feat(inflation): make inflation params a collection and add commands to update them
- [#1872](https://github.com/NibiruChain/nibiru/pull/1872) - chore(math): use cosmossdk.io/math to replace sdk types
- [#1874](https://github.com/NibiruChain/nibiru/pull/1874) - chore(proto): remove the proto stringer as per Cosmos SDK migration guidelines
- [#1932](https://github.com/NibiruChain/nibiru/pull/1932) - fix(gosdk): fix keyring import functions

#### Nibiru EVM

- [#1837](https://github.com/NibiruChain/nibiru/pull/1837) - feat(eth): protos, eth types, and evm module types
- [#1838](https://github.com/NibiruChain/nibiru/pull/1838) - feat(eth): Go-ethereum, crypto, encoding, and unit tests for evm/types
- [#1841](https://github.com/NibiruChain/nibiru/pull/1841) - feat(eth): Collections encoders for bytes, Ethereum addresses, and Ethereum hashes
- [#1855](https://github.com/NibiruChain/nibiru/pull/1855) - feat(eth-pubsub): Implement in-memory EventBus for real-time topic management and event distribution
- [#1856](https://github.com/NibiruChain/nibiru/pull/1856) - feat(eth-rpc): Conversion types and functions between Ethereum txs and blocks and Tendermint ones.
- [#1861](https://github.com/NibiruChain/nibiru/pull/1861) - feat(eth-rpc): RPC backend, Ethereum tracer, KV indexer, and RPC APIs
- [#1869](https://github.com/NibiruChain/nibiru/pull/1869) - feat(eth): Module and start of keeper tests
- [#1871](https://github.com/NibiruChain/nibiru/pull/1871) - feat(evm): app config and json-rpc
- [#1873](https://github.com/NibiruChain/nibiru/pull/1873) - feat(evm): keeper collections and grpc query impls for EthAccount, NibiruAccount
- [#1883](https://github.com/NibiruChain/nibiru/pull/1883) - feat(evm): keeper logic, Ante handlers, EthCall, and EVM transactions.
- [#1887](https://github.com/NibiruChain/nibiru/pull/1887) - test(evm): eth api integration test suite
- [#1889](https://github.com/NibiruChain/nibiru/pull/1889) - feat: implemented basic evm tx methods
- [#1895](https://github.com/NibiruChain/nibiru/pull/1895) - refactor(geth): Reference go-ethereum as a submodule for easier change tracking with upstream
- [#1901](https://github.com/NibiruChain/nibiru/pull/1901) - test(evm): more e2e test contracts for edge cases
- [#1907](https://github.com/NibiruChain/nibiru/pull/1907) - test(evm): grpc_query full coverage
- [#1909](https://github.com/NibiruChain/nibiru/pull/1909) - chore(evm): set is_london true by default and removed from config
- [#1911](https://github.com/NibiruChain/nibiru/pull/1911) - chore(evm): simplified config by removing old eth forks
- [#1912](https://github.com/NibiruChain/nibiru/pull/1912) - test(evm): unit tests for evm_ante
- [#1914](https://github.com/NibiruChain/nibiru/pull/1914) - refactor(evm): Remove dead code and document non-EVM ante handler- [#1917](https://github.com/NibiruChain/nibiru/pull/1917) - test(e2e-evm): TypeScript support. Type generation from compiled contracts. Formatter for TS code.
- [#1917](https://github.com/NibiruChain/nibiru/pull/1917) - test(e2e-evm): TypeScript support. Type generation from compiled contracts. Formatter for TS code.
- [#1922](https://github.com/NibiruChain/nibiru/pull/1922) - feat(evm): tracer option is read from the config.
- [#1936](https://github.com/NibiruChain/nibiru/pull/1936) - feat(evm): EVM fungible token protobufs and encoding tests
- [#1947](https://github.com/NibiruChain/nibiru/pull/1947) - fix(evm): fix FunToken state marshalling
- [#1949](https://github.com/NibiruChain/nibiru/pull/1949) - feat(evm): add fungible token mapping queries
- [#1950](https://github.com/NibiruChain/nibiru/pull/1950) - feat(evm): Tx to create FunToken mapping from ERC20, contract embeds, and ERC20 queries.
- [#1956](https://github.com/NibiruChain/nibiru/pull/1956) - feat(evm): msg to send bank coin to erc20
- [#1958](https://github.com/NibiruChain/nibiru/pull/1958) - chore(evm): wiped deprecated evm apis: miner, personal
- [#1959](https://github.com/NibiruChain/nibiru/pull/1959) - feat(evm): Add precompile to the EVM that enables transfers of ERC20 tokens to "nibi" accounts as regular Ethereum transactions
- [#1960](https://github.com/NibiruChain/nibiru/pull/1960) - test(network): graceful cleanup for more consistent CI runs
- [#1961](https://github.com/NibiruChain/nibiru/pull/1961) - chore(test): reverted funtoken precompile test back to the isolated state
- [#1962](https://github.com/NibiruChain/nibiru/pull/1962) - chore(evm): code cleanup, unused code, typos, styles, warnings
- [#1963](https://github.com/NibiruChain/nibiru/pull/1963) - feat(evm): Deduct a fee during the creation of a FunToken mapping. Implemented by `deductCreateFunTokenFee` inside of the `eth.evm.v1.MsgCreateFunToken` transaction.
- [#1965](https://github.com/NibiruChain/nibiru/pull/1965) - refactor(evm): remove evm post-processing hooks
- [#1966](https://github.com/NibiruChain/nibiru/pull/1966) - refactor(evm): clean up AnteHandler setup
- [#1967](https://github.com/NibiruChain/nibiru/pull/1967) - feat(evm): export genesis
- [#1968](https://github.com/NibiruChain/nibiru/pull/1968) - refactor(evm): funtoken events, cli commands and queries
- [#1970](https://github.com/NibiruChain/nibiru/pull/1970) - refactor(evm): move evm antehandlers to separate package. Remove "gosdk/sequence_test.go", which causes a race condition in CI.
- [#1971](https://github.com/NibiruChain/nibiru/pull/1971) - feat(evm): typed events for contract creation, contract execution and transfer
- [#1973](https://github.com/NibiruChain/nibiru/pull/1973) - chore(appconst): Add chain IDs ending in "3" to the "knownEthChainIDMap". This makes it possible to use devnet 3 and testnet 3.
- [#1976](https://github.com/NibiruChain/nibiru/pull/1976) - refactor(evm): unique chain ids for all networks
- [#1977](https://github.com/NibiruChain/nibiru/pull/1977) - fix(localnet): rolled back change of evm validator address with cosmos derivation path
- [#1979](https://github.com/NibiruChain/nibiru/pull/1979) - refactor(db): use pebbledb as the default db in integration tests
- [#1981](https://github.com/NibiruChain/nibiru/pull/1981) - fix(evm): remove isCheckTx() short circuit on `AnteDecVerifyEthAcc`
- [#1982](https://github.com/NibiruChain/nibiru/pull/1982) - feat(evm): add GlobalMinGasPrices
- [#1983](https://github.com/NibiruChain/nibiru/pull/1983) - chore(evm): remove ExtensionOptionsWeb3Tx and ExtensionOptionDynamicFeeTx
- [#1984](https://github.com/NibiruChain/nibiru/pull/1984) - refactor(evm): embeds
- [#1985](https://github.com/NibiruChain/nibiru/pull/1985) - feat(evm)!: Use atto denomination for the wei units in the EVM so that NIBI is "ether" to clients. Only micronibi (unibi) amounts can be transferred. All clients follow the constraint equation, 1 ether == 1 NIBI == 10^6 unibi == 10^18 wei.
- [#1986](https://github.com/NibiruChain/nibiru/pull/1986) - feat(evm): Combine both account queries into "/eth.evm.v1.Query/EthAccount", accepting both nibi-prefixed Bech32 addresses and Ethereum-type hexadecimal addresses as input.
- [#1989](https://github.com/NibiruChain/nibiru/pull/1989) - refactor(evm): simplify evm module address
- [#1996](https://github.com/NibiruChain/nibiru/pull/1996) - perf(evm-keeper-precompile): implement sorted map for `k.precompiles` to remove dead code
- [#1997](https://github.com/NibiruChain/nibiru/pull/1997) - refactor(evm): Remove unnecessary params: "enable_call", "enable_create".
- [#2000](https://github.com/NibiruChain/nibiru/pull/2000) - refactor(evm): simplify ERC-20 keeper methods
- [#2001](https://github.com/NibiruChain/nibiru/pull/2001) - refactor(evm): simplify FunToken methods and tests
- [#2002](https://github.com/NibiruChain/nibiru/pull/2002) - feat(evm): Add the account query to the EVM command. Cover the CLI with tests.
- [#2003](https://github.com/NibiruChain/nibiru/pull/2003) - fix(evm): fix FunToken conversions between Cosmos and EVM
- [#2004](https://github.com/NibiruChain/nibiru/pull/2004) - refactor(evm)!: replace `HexAddr` with `EIP55Addr`
- [#2006](https://github.com/NibiruChain/nibiru/pull/2006) - test(evm): e2e tests for eth_* endpoints
- [#2008](https://github.com/NibiruChain/nibiru/pull/2008) - refactor(evm): clean up precompile setups 
- [#2013](https://github.com/NibiruChain/nibiru/pull/2013) - chore(evm): Set appropriate gas value for the required gas of the "IFunToken.sol" precompile.
- [#2014](https://github.com/NibiruChain/nibiru/pull/2014) - feat(evm): Emit block bloom event in EndBlock hook.
- [#2017](https://github.com/NibiruChain/nibiru/pull/2017) - fix(evm): Fix DynamicFeeTx gas cap parameters
- [#2019](https://github.com/NibiruChain/nibiru/pull/2019) - chore(evm): enabled debug rpc api on localnet.
- [#2020](https://github.com/NibiruChain/nibiru/pull/2020) - test(evm): e2e tests for debug namespace
- [#2022](https://github.com/NibiruChain/nibiru/pull/2022) - feat(evm): debug_traceCall method implemented
- [#2023](https://github.com/NibiruChain/nibiru/pull/2023) - fix(evm)!: adjusted generation and parsing of the block bloom events
- [#2030](https://github.com/NibiruChain/nibiru/pull/2030) - fix(eth/rpc): test and fix eth_getTransactionByHash and eth_getLogs. This is meant to address a bug where the EVMIndexer has a discrepenacy during tx search depending on how the "eth_getTransactionByHash" request is sent. Additionally, this change adjusts the `testnetwork` full node setup to mirror production more closely and expose the structs for the Ethereum JSON-RPC API. 
- [#2031](https://github.com/NibiruChain/nibiru/pull/2031) - fix(evm): debug calls with custom tracer and tracer options

#### Dapp modules: perp, spot, oracle, etc

- [#1573](https://github.com/NibiruChain/nibiru/pull/1573) - feat(perp): Close markets and compute settlement price
- [#1632](https://github.com/NibiruChain/nibiru/pull/1632) - feat(perp): Add settle position transaction
- [#1656](https://github.com/NibiruChain/nibiru/pull/1656) - feat(perp): Make the collateral denom a stateful collections.Item
- [#1663](https://github.com/NibiruChain/nibiru/pull/1663) - feat(perp): Add volume based rebates
- [#1669](https://github.com/NibiruChain/nibiru/pull/1669) - feat(perp): add query to get collateral metadata
- [#1677](https://github.com/NibiruChain/nibiru/pull/1677) - fix(perp): make Gen_market set initial perp versions
- [#1680](https://github.com/NibiruChain/nibiru/pull/1680) - feat(perp): MsgShiftPegMultiplier, MsgShiftSwapInvariant.
- [#1683](https://github.com/NibiruChain/nibiru/pull/1683) - feat(perp): Add `StartDnREpoch` to `AfterEpochEnd` hook
- [#1686](https://github.com/NibiruChain/nibiru/pull/1686) - test(perp): add more tests for perp module msg server for DnR
- [#1687](https://github.com/NibiruChain/nibiru/pull/1687) - chore(wasmbinding): delete CustomQuerier since we have QueryRequest::Stargate now
- [#1705](https://github.com/NibiruChain/nibiru/pull/1705) - feat(perp): Add oracle pair to market object
- [#1718](https://github.com/NibiruChain/nibiru/pull/1718) - fix(perp): fees does not require additional funds
- [#1734](https://github.com/NibiruChain/nibiru/pull/1734) - feat(perp): MsgDonateToPerpFund sudo call as part of #1642
- [#1749](https://github.com/NibiruChain/nibiru/pull/1749) - feat(perp): move close market from Wasm Binding to MsgCloseMarket
- [#1752](https://github.com/NibiruChain/nibiru/pull/1752) - feat(oracle): MsgEditOracleParams sudo tx msg as part of #1642
- [#1755](https://github.com/NibiruChain/nibiru/pull/1755) - feat(oracle): Add more events on validator's performance
- [#1764](https://github.com/NibiruChain/nibiru/pull/1764) - fix(perp): make updateswapinvariant aware of total short supply to avoid panics
- [#1710](https://github.com/NibiruChain/nibiru/pull/1710) - refactor(perp): Clean and organize module errors for x/perp

### Non-breaking/Compatible Improvements

- [#1893](https://github.com/NibiruChain/nibiru/pull/1893) - feat(gosdk): migrate Go-sdk into the Nibiru blockchain repo.
- [#1899](https://github.com/NibiruChain/nibiru/pull/1899) - build(deps): cometbft v0.37.5, cosmos-sdk v0.47.11, proto-builder v0.14.0
- [#1913](https://github.com/NibiruChain/nibiru/pull/1913) - fix(tests): race condition from heavy Network tests
- [#1992](https://github.com/NibiruChain/nibiru/pull/1992) - chore: enabled grpc for localnet
- [#1999](https://github.com/NibiruChain/nibiru/pull/1999) - chore: update nibi go package version to v2

### Dependencies

- Bump `github.com/grpc-ecosystem/grpc-gateway/v2` from 2.18.1 to 2.19.1 ([#1767](https://github.com/NibiruChain/nibiru/pull/1767), [#1782](https://github.com/NibiruChain/nibiru/pull/1782))
- Bump `robinraju/release-downloader` from 1.8 to 1.11 ([#1783](https://github.com/NibiruChain/nibiru/pull/1783), [#1839](https://github.com/NibiruChain/nibiru/pull/1839), [#1948](https://github.com/NibiruChain/nibiru/pull/1948))
- Bump `github.com/prometheus/client_golang` from 1.17.0 to 1.18.0 ([#1750](https://github.com/NibiruChain/nibiru/pull/1750))
- Bump `golang.org/x/crypto` from 0.15.0 to 0.17.0 ([#1724](https://github.com/NibiruChain/nibiru/pull/1724), [#1843](https://github.com/NibiruChain/nibiru/pull/1843))
- Bump `github.com/holiman/uint256` from 1.2.3 to 1.2.4 ([#1730](https://github.com/NibiruChain/nibiru/pull/1730))
- Bump `github.com/dvsekhvalnov/jose2go` from 1.5.0 to 1.6.0 ([#1733](https://github.com/NibiruChain/nibiru/pull/1733))
- Bump `github.com/spf13/cast` from 1.5.1 to 1.6.0 ([#1689](https://github.com/NibiruChain/nibiru/pull/1689))
- Bump `cosmossdk.io/math` from 1.1.2 to 1.2.0 ([#1676](https://github.com/NibiruChain/nibiru/pull/1676))
- Bump `github.com/grpc-ecosystem/grpc-gateway/v2` from 2.18.0 to 2.18.1 ([#1675](https://github.com/NibiruChain/nibiru/pull/1675))
- Bump `actions/setup-go` from 4 to 5 ([#1696](https://github.com/NibiruChain/nibiru/pull/1696))
- Bump `golang` from 1.19 to 1.21 ([#1698](https://github.com/NibiruChain/nibiru/pull/1698))
- [#1678](https://github.com/NibiruChain/nibiru/pull/1678) - chore(deps): collections to v0.4.0 for math.Int value encoder
- Bump `golang.org/x/net` from 0.0.0-20220607020251-c690dde0001d to 0.23.0 in /geth ([#1849](https://github.com/NibiruChain/nibiru/pull/1849))
- Bump `golang.org/x/net` from 0.20.0 to 0.23.0 ([#1850](https://github.com/NibiruChain/nibiru/pull/1850))
- Bump `github.com/supranational/blst` from 0.3.8-0.20220526154634-513d2456b344 to 0.3.11 ([#1851](https://github.com/NibiruChain/nibiru/pull/1851))
- Bump `golangci/golangci-lint-action` from 4 to 6 ([#1854](https://github.com/NibiruChain/nibiru/pull/1854), [#1867](https://github.com/NibiruChain/nibiru/pull/1867))
- Bump `github.com/hashicorp/go-getter` from 1.7.1 to 1.7.5 ([#1858](https://github.com/NibiruChain/nibiru/pull/1858), [#1938](https://github.com/NibiruChain/nibiru/pull/1938))
- Bump `github.com/btcsuite/btcd` from 0.23.3 to 0.24.0 ([#1862](https://github.com/NibiruChain/nibiru/pull/1862))
- Bump `pozetroninc/github-action-get-latest-release` from 0.7.0 to 0.8.0 ([#1863](https://github.com/NibiruChain/nibiru/pull/1863))
- Bump `bufbuild/buf-setup-action` from 1.30.1 to 1.36.0 ([#1891](https://github.com/NibiruChain/nibiru/pull/1891), [#1900](https://github.com/NibiruChain/nibiru/pull/1900), [#1923](https://github.com/NibiruChain/nibiru/pull/1923), [#1972](https://github.com/NibiruChain/nibiru/pull/1972), [#1974](https://github.com/NibiruChain/nibiru/pull/1974), [#1988](https://github.com/NibiruChain/nibiru/pull/1988))

## [v1.5.0](https://github.com/NibiruChain/nibiru/releases/tag/v1.5.0) - 2024-06-21

Nibiru v1.5.0 enables IBC CosmWasm smart contracts.

- [[Release Link](https://github.com/NibiruChain/nibiru/releases/tag/v1.5.0)]
- [[Commits](https://github.com/NibiruChain/nibiru/commits/v1.5.0)]

### Features

- [#1931](https://github.com/NibiruChain/nibiru/pull/1931) - feat(ibc): add `wasm` route to IBC router

## [v1.4.0](https://github.com/NibiruChain/nibiru/releases/tag/v1.4.0) - 2024-06-04

Nibiru v1.4.0 adds PebbleDB support and increases the wasm contract size limit to 3MB.

- [[Release Link](https://github.com/NibiruChain/nibiru/releases/tag/v1.4.0)]
- [[Commits](https://github.com/NibiruChain/nibiru/commits/v1.4.0)]

### State Machine Breaking

- [#1906](https://github.com/NibiruChain/nibiru/pull/1906) - feat(wasm): increase contract size limit to 3MB

### Features

- [#1818](https://github.com/NibiruChain/nibiru/pull/1818) - feat: add pebbledb support
- [#1908](https://github.com/NibiruChain/nibiru/pull/1908) - chore: make pebbledb the default db backend
-

## [v1.3.0](https://github.com/NibiruChain/nibiru/releases/tag/v1.3.0) - 2024-05-07

Nibiru v1.3.0 adds interchain accounts.

- [[Release Link](https://github.com/NibiruChain/nibiru/releases/tag/v1.3.0)]
- [[Commits](https://github.com/NibiruChain/nibiru/commits/v1.3.0)]

### Features

- [#1820](https://github.com/NibiruChain/nibiru/pull/1820) - feat: add interchain accounts

### Bug Fixes

- [#1864](https://github.com/NibiruChain/nibiru/pull/1864) - fix(ica): add ICA controller stack

### Improvements

- [#1859](https://github.com/NibiruChain/nibiru/pull/1859) - refactor(oracle): add oracle slashing events

## [v1.2.0](https://github.com/NibiruChain/nibiru/releases/tag/v1.2.0) - 2024-03-28

Nibiru v1.2.0 adds a burn method to the x/inflation module that allows senders to burn tokens.

- [[Release Link](https://github.com/NibiruChain/nibiru/releases/tag/v1.2.0)]
- [[Commits](https://github.com/NibiruChain/nibiru/commits/v1.2.0)]

### Features

- [#1832](https://github.com/NibiruChain/nibiru/pull/1832) - feat(tokenfactory): add burn method for native tokens

## [v1.1.0](https://github.com/NibiruChain/nibiru/releases/tag/v1.1.0) - 2024-03-19

Nibiru v1.1.0 is the minor release used to add inflation to the network.

### State Machine Breaking

- [#1786](https://github.com/NibiruChain/nibiru/pull/1786) - fix(inflation): fix inflation off-by 2 error
- [#1796](https://github.com/NibiruChain/nibiru/pull/1796) - fix(inflation): fix num skipped epoch when inflation is added to an existing chain
- [#1797](https://github.com/NibiruChain/nibiru/pull/1797) - fix(inflation): fix num skipped epoch updates logic
- [#1712](https://github.com/NibiruChain/nibiru/pull/1712) - refactor(inflation): turn inflation off by default

### Bug Fixes

- [#1706](https://github.com/NibiruChain/nibiru/pull/706) - fix: `v1.1.0` upgrade handler
- [#1804](https://github.com/NibiruChain/nibiru/pull/1804) - fix(inflation): update default parameters
- [#1688](https://github.com/NibiruChain/nibiru/pull/1688) - fix(inflation): make default inflation allocation follow tokenomics

### Features

- [#1670](https://github.com/NibiruChain/nibiru/pull/1670) - feat(inflation): Make inflation polynomial
- [#1682](https://github.com/NibiruChain/nibiru/pull/1682) - feat!: add upgrade handler for v1.1.0
- [#1776](https://github.com/NibiruChain/nibiru/pull/1776) - feat(inflation): make inflation params a collection and add commands to update them
- [#1795](https://github.com/NibiruChain/nibiru/pull/1795) - feat(inflation): add inflation tx cmds

### Improvements

- [#1695](https://github.com/NibiruChain/nibiru/pull/1695) - feat(inflation): add events for inflation distribution
- [#1792](https://github.com/NibiruChain/nibiru/pull/1792) - fix(inflation): uncomment legacy amino register on app module basic
- [#1799](https://github.com/NibiruChain/nibiru/pull/1799) refactor,docs(inflation): Document everything + delete unused code. Make perp and spot optional features in localnet.sh

## [v1.0.3](https://github.com/NibiruChain/nibiru/releases/tag/v1.0.3) - 2024-03-18

### Fix

- [#1816](https://github.com/NibiruChain/nibiru/pull/1816) - fix(ibc): fix ibc transaction from wasm contract

### CLI

- [#1731](https://github.com/NibiruChain/nibiru/pull/1731) - feat(cli): add cli command to decode stargate base64 messages
- [#1754](https://github.com/NibiruChain/nibiru/pull/1754) - refactor(decode-base64): clean code improvements and fn docs

## [v1.0.2](https://github.com/NibiruChain/nibiru/releases/tag/v1.0.2) - 2024-03-03

### Dependencies

- [65c06ba](https://github.com/NibiruChain/nibiru/commit/65c06ba774c260ece942131ad7a93de0e162266e) - Bump `cosmos-sdk` to v0.47.10

## [v1.0.1](https://github.com/NibiruChain/nibiru/releases/tag/v1.0.1) - 2024-02-09

### Dependencies

- [#1778](https://github.com/NibiruChain/nibiru/pull/1778) - chore: bump librocksdb to v8.9.1

## [v1.0.0](https://github.com/NibiruChain/nibiru/releases/tag/v1.0.0)

Nibiru v1.0.0 is the major release used for the genesis of the mainnet network,
`cataclysm-1`. It includes all of the general purpose modules such
as `devgas`, `sudo`, `wasm`, `tokenfactory`, and the defaults from Cosmos SDK
v0.47.5.

- [[Release Link](https://github.com/NibiruChain/nibiru/releases/tag/v1.0.0)]
- [[Commits](https://github.com/NibiruChain/nibiru/commits/v1.0.0)]
- [tag:v1.0.0](https://github.com/NibiruChain/nibiru/commits/v1.0.0) epic(v1.0.0): Remove unneeded Dapp modules for smooth upgrades.
  - chore!: [Date: 2023-10-16] Remove inflation, perp, stablecoin, and spot
    modules and related protobufs. This will make it easier to add the store keys
    layer if we have breaking changes before the Dapps go live without blocking
    the mainnet deployment.  
    Commits:
    [#1667](https://github.com/NibiruChain/nibiru/pull/1667)
    [6a01abe](https://github.com/NibiruChain/nibiru/commit/6a01abe5c99e26a1d17d96f359d06d61bc1e6e70)
    [2a250a3](https://github.com/NibiruChain/nibiru/commit/2a250a3c4c60c58c5526ac7d75ce5b9e13889471)
    [d713f41](https://github.com/NibiruChain/nibiru/commit/d713f41dfe17d6d29451ade4d2f0e6d950ce7c59)
    [011f1ed](https://github.com/NibiruChain/nibiru/commit/011f1ed431d92899d01583e5e6110e663eceaa24)

### Features

- [#1596](https://github.com/NibiruChain/nibiru/pull/1596) - epic(tokenfactory): State transitions, collections, genesis import and export, and app wiring
- [#1607](https://github.com/NibiruChain/nibiru/pull/1607) - Token factory transaction messages for CreateDenom, ChangeAdmin, and UpdateModuleParams
- [#1620](https://github.com/NibiruChain/nibiru/pull/1620) - Token factory transaction messages for Mint and Burn
- [#1573](https://github.com/NibiruChain/nibiru/pull/1573) - feat(perp): Close markets and compute settlement price

### State Machine Breaking

- [#1609](https://github.com/NibiruChain/nibiru/pull/1609) - refactor(app)!: Remove x/stablecoin module.
- [#1613](https://github.com/NibiruChain/nibiru/pull/1613) - feat(app)!: enforce min commission by changing default and genesis validation
- [#1615](https://github.com/NibiruChain/nibiru/pull/1613) - feat(ante)!: Ante handler to add a maximum commission rate of 25% for validators.
- [#1616](https://github.com/NibiruChain/nibiru/pull/1616) - fix(app)!: Add custom wasm snapshotter for proper state exports
- [#1617](https://github.com/NibiruChain/nibiru/pull/1617) - fix(app)!: non-nil snapshot manager is not guaranteed in testapp
- [#1645](https://github.com/NibiruChain/nibiru/pull/1645) - fix(tokenfactory)!: token supply in bank keeper must be correct after MsgBurn.
- [#1646](https://github.com/NibiruChain/nibiru/pull/1646) - feat(wasmbinding)!: whitelisted stargate queries for QueryRequest::Stargate: auth, bank, gov, tokenfactory, epochs, inflation, oracle, sudo, devgas
- [#1667](https://github.com/NibiruChain/nibiru/pull/1667) - chore(inflation)!: unwire x/inflation

### Improvements

- [#1610](https://github.com/NibiruChain/nibiru/pull/1610) - refactor(app): Simplify app.go with less redundant imports using struct embedding.
- [#1614](https://github.com/NibiruChain/nibiru/pull/1614) - refactor(proto): Use explicit namespacing on proto imports for #1608
- [#1630](https://github.com/NibiruChain/nibiru/pull/1630) - refactor(wasm): clean up wasmbinding/ folder structure
- [#1631](https://github.com/NibiruChain/nibiru/pull/1631) - fix(.goreleaser.yml): Load version for wasmvm dynamically.
- [#1638](https://github.com/NibiruChain/nibiru/pull/1638) - test(tokenfactory): integration test core logic with a real smart contract using `nibiru-std`
- [#1659](https://github.com/NibiruChain/nibiru/pull/1659) - refactor(oracle): curate oracle default whitelist

### Dependencies

- Bump `github.com/prometheus/client_golang` from 1.16.0 to 1.17.0 ([#1605](https://github.com/NibiruChain/nibiru/pull/1605))
- Bump `bufbuild/buf-setup-action` from 1.26.1 to 1.27.1 ([#1624](https://github.com/NibiruChain/nibiru/pull/1624), [#1641](https://github.com/NibiruChain/nibiru/pull/1641))
- Bump `stefanzweifel/git-auto-commit-action` from 4 to 5 ([#1625](https://github.com/NibiruChain/nibiru/pull/1625))
- Bump `github.com/CosmWasm/wasmvm` from 1.4.0 to 1.5.0 ([#1629](https://github.com/NibiruChain/nibiru/pull/1629), [#1657](https://github.com/NibiruChain/nibiru/pull/1657))
- Bump `google.golang.org/grpc` from 1.58.2 to 1.59.0 ([#1633](https://github.com/NibiruChain/nibiru/pull/1633), [#1643](https://github.com/NibiruChain/nibiru/pull/1643))
- Bump `golang.org/x/net` from 0.12.0 to 0.17.0 ([#1634](https://github.com/NibiruChain/nibiru/pull/1634))
- Bump `github.com/cosmos/ibc-go/v7` from 7.3.0 to 7.3.1 ([#1647](https://github.com/NibiruChain/nibiru/pull/1647))
- Bump `github.com/CosmWasm/wasmd` from 0.40.2 to 0.43.0 ([#1660](https://github.com/NibiruChain/nibiru/pull/1660))
- Bump `github.com/CosmWasm/wasmd` from 0.43.0 to 0.44.0 ([#1666](https://github.com/NibiruChain/nibiru/pull/1666))

### Bug Fixes

- [#1606](https://github.com/NibiruChain/nibiru/pull/1606) - fix(perp): emit `MarketUpdatedEvent` in the absence of index price
- [#1649](https://github.com/NibiruChain/nibiru/pull/1649) - fix(ledger): fix ledger for newer macos versions
- [#1655](https://github.com/NibiruChain/nibiru/pull/1655) - fix(inflation): inflate NIBI correctly to strategic treasury account

## [v0.21.11] - 2023-10-02

NOTE: It's pragmatic to assume that any change prior to v1.0.0 was state machine breaking.

- Summary: Changes up to pull request #1616
- [[Release Link](https://github.com/NibiruChain/nibiru/releases/tag/v0.21.11)]

## [v0.21.10] - 2023-09-20

- Summary: Changes up to pull request #1595
- [[Release Link](https://github.com/NibiruChain/nibiru/releases/tag/v0.21.10)]

### State Machine Breaking

#### Features (Breaking)

- [#1594](https://github.com/NibiruChain/nibiru/pull/1594) - feat: add user discounts
- [#1585](https://github.com/NibiruChain/nibiru/pull/1585) - feat: include flag versioned in query markets to allow to query disabled markets
- [#1575](https://github.com/NibiruChain/nibiru/pull/1575) - feat(perp): Add trader volume tracking
- [#1559](https://github.com/NibiruChain/nibiru/pull/1559) - feat: add versions to markets to allow to disable them
- [#1543](https://github.com/NibiruChain/nibiru/pull/1543) - epic(devgas): devgas module for incentivizing smart contract
- [#1543](https://github.com/NibiruChain/nibiru/pull/1543) - epic(devgas): devgas module for incentivizing smart contract
- [#1541](https://github.com/NibiruChain/nibiru/pull/1541) - feat(perp): add clamp to premium fractions
- [#1520](https://github.com/NibiruChain/nibiru/pull/1520) - feat(wasm): no op handler + tests with updated contracts
- [#1503](https://github.com/NibiruChain/nibiru/pull/1503) - feat(wasm): add Oracle Exchange Rate query for wasm
- [#1503](https://github.com/NibiruChain/nibiru/pull/1503) - feat(wasm): add Oracle Exchange Rate query for wasm
- [#1502](https://github.com/NibiruChain/nibiru/pull/1502) - feat: add ledger build support
- [#1501](https://github.com/NibiruChain/nibiru/pull/1501) - feat(proto): add Python buf generation logic for py-sdk
- [#1501](https://github.com/NibiruChain/nibiru/pull/1501) - feat(proto): add Python buf generation logic for py-sdk
- [#1501](https://github.com/NibiruChain/nibiru/pull/1501) - feat(localnet.sh): (1) Make it possible to run while offline. (2) Implement --no-build option to use the script with the current `nibid` installed.
- [#1501](https://github.com/NibiruChain/nibiru/pull/1501) - feat(localnet.sh): (1) Make it possible to run while offline. (2) Implement --no-build option to use the script with the current `nibid` installed.
- [#1498](https://github.com/NibiruChain/nibiru/pull/1498) - feat: add cli to change root sudo command
- [#1498](https://github.com/NibiruChain/nibiru/pull/1498) - feat: add cli to change root sudo command
- [#1495](https://github.com/NibiruChain/nibiru/pull/1495) - feat: add genmsg module
- [#1494](https://github.com/NibiruChain/nibiru/pull/1494) - feat: create cli to add sudo account into genesis
- [#1479](https://github.com/NibiruChain/nibiru/pull/1479) - feat(perp): implement `PartialClose`
- [#1479](https://github.com/NibiruChain/nibiru/pull/1479) - feat(perp): implement `PartialClose`
- [#1463](https://github.com/NibiruChain/nibiru/pull/1463) - feat(oracle): add genesis pricefeeder delegation
- [#1463](https://github.com/NibiruChain/nibiru/pull/1463) - feat(oracle): add genesis pricefeeder delegation
- [#1463](https://github.com/NibiruChain/nibiru/pull/1463) - feat(oracle): add genesis pricefeeder delegation
- [#1421](https://github.com/NibiruChain/nibiru/pull/1421) - feat(oracle): add expiry time to oracle prices
- [#1407](https://github.com/NibiruChain/nibiru/pull/1407) - feat!: upgrade to Cosmos SDK v0.47.3
- [#1407](https://github.com/NibiruChain/nibiru/pull/1407) - feat!: upgrade to Cosmos SDK v0.47.3
- [#1387](https://github.com/NibiruChain/nibiru/pull/1387) - feat: upgrade to Cosmos SDK v0.46.10
- [#1387](https://github.com/NibiruChain/nibiru/pull/1387) - feat: upgrade to Cosmos SDK v0.46.10
- [#1380](https://github.com/NibiruChain/nibiru/pull/1380) - feat(wasm): Add CreateMarket admin call for the controller contract
- [#1380](https://github.com/NibiruChain/nibiru/pull/1380) - feat(wasm): Add CreateMarket admin call for the controller contract
- [#1373](https://github.com/NibiruChain/nibiru/pull/1373) - feat(perp): `perpv2` `add-genesis-perp-market` CLI command
- [#1371](https://github.com/NibiruChain/nibiru/pull/1371) - feat: realize bad debt when a user tries to close his position
- [#1370](https://github.com/NibiruChain/nibiru/pull/1370) - feat(perp): `perpv2` `CreatePool` method
- [#1367](https://github.com/NibiruChain/nibiru/pull/1367) - feat: wire enable market to wasm
- [#1367](https://github.com/NibiruChain/nibiru/pull/1367) - feat: wire enable market to wasm
- [#1366](https://github.com/NibiruChain/nibiru/pull/1366) - feat: fix bindings test in cw_test
- [#1363](https://github.com/NibiruChain/nibiru/pull/1363) - feat(perp): wire `PerpV2` module
- [#1362](https://github.com/NibiruChain/nibiru/pull/1362) - feat(perp): add `perpv2` cli
- [#1361](https://github.com/NibiruChain/nibiru/pull/1361) - feat(perp): add `PerpV2` module
- [#1359](https://github.com/NibiruChain/nibiru/pull/1359) - feat(perp): Add InsuranceFundWithdraw admin call with corresponding smart contract
- [#1359](https://github.com/NibiruChain/nibiru/pull/1359) - feat(perp): Add InsuranceFundWithdraw admin call with corresponding smart contract
- [#1352](https://github.com/NibiruChain/nibiru/pull/1352) - feat(perp): add PerpKeeperV2 `MsgServer`
- [#1350](https://github.com/NibiruChain/nibiru/pull/1350) - feat(perp): `EditPriceMultiplier` and `EditSwapInvariant`
- [#1345](https://github.com/NibiruChain/nibiru/pull/1345) - feat(perp): PerpV2 QueryServer
- [#1344](https://github.com/NibiruChain/nibiru/pull/1344) - feat(perp): PerpKeeperV2 `AddMargin` and `RemoveMargin`
- [#1343](https://github.com/NibiruChain/nibiru/pull/1343) - feat(perp): add PerpKeeperV2 `MultiLiquidate`
- [#1342](https://github.com/NibiruChain/nibiru/pull/1342) - feat(perp): market not enabled can only be used to close out existing positions
- [#1342](https://github.com/NibiruChain/nibiru/pull/1342) - feat(perp): market not enabled can only be used to close out existing positions
- [#1341](https://github.com/NibiruChain/nibiru/pull/1341) - feat(bindings/oracle): add bindings for oracle module params
- [#1340](https://github.com/NibiruChain/nibiru/pull/1340) - feat(wasm): Enforce x/sudo contract permission checks on the shifter contract + integration tests
- [#1338](https://github.com/NibiruChain/nibiru/pull/1338) - feat(perp): V2 OpenPosition

#### Bug Fixes (Breaking)

- [#1586](https://github.com/NibiruChain/nibiru/pull/1586) - fix(sudo): make messages compatible with `Amino`
- [#1565](https://github.com/NibiruChain/nibiru/pull/1565) - fix(oracle)!: Count vote omission as abstain for less slashing + more stability
- [#1493](https://github.com/NibiruChain/nibiru/pull/1493) - fix(perp): allow `ClosePosition` when there is bad debt
- [#1476](https://github.com/NibiruChain/nibiru/pull/1476) - fix(wasm)!: call `ValidateBasic` before all `sdk.Msg` calls for the bindings-perp contract + remove sudo permissioning
- [#1467](https://github.com/NibiruChain/nibiru/pull/1467) - fix(oracle): make `calcTwap` safer
- [#1467](https://github.com/NibiruChain/nibiru/pull/1467) - fix(oracle): make `calcTwap` safer
- [#1464](https://github.com/NibiruChain/nibiru/pull/1464) - fix(gov): wire legacy proposal handlers
- [#1464](https://github.com/NibiruChain/nibiru/pull/1464) - fix(gov): wire legacy proposal handlers
- [#1459](https://github.com/NibiruChain/nibiru/pull/1459) - fix(spot): wire `x/spot` msgService into app router
- [#1459](https://github.com/NibiruChain/nibiru/pull/1459) - fix(spot): wire `x/spot` msgService into app router
- [#1452](https://github.com/NibiruChain/nibiru/pull/1452) - fix(oracle): continue with abci hook during error
- [#1451](https://github.com/NibiruChain/nibiru/pull/1451) - fix(perp): decrease position with zero size
- [#1446](https://github.com/NibiruChain/nibiru/pull/1446) - fix(cmd): Add custom InitCmd to set set desired Tendermint consensus params for each node.
- [#1441](https://github.com/NibiruChain/nibiru/pull/1441) - fix(oracle): ignore abstain votes in std dev calculation
- [#1425](https://github.com/NibiruChain/nibiru/pull/1425) - fix: remove positions from state when closed with reverse position
- [#1423](https://github.com/NibiruChain/nibiru/pull/1423) - fix: remove panics from abci hooks
- [#1422](https://github.com/NibiruChain/nibiru/pull/1422) - fix(oracle): handle zero oracle rewards
- [#1420](https://github.com/NibiruChain/nibiru/pull/1420) - refactor(oracle): update default params
- [#1419](https://github.com/NibiruChain/nibiru/pull/1419) - fix(spot): add pools to genesis state
- [#1417](https://github.com/NibiruChain/nibiru/pull/1417) - fix: run end blocker on block end for perp v2
- [#1414](https://github.com/NibiruChain/nibiru/pull/1414) - fix(oracle): Add deterministic map iterations to avoid consensus failure.
- [#1413](https://github.com/NibiruChain/nibiru/pull/1413) - fix(perp): provide descriptive errors when all liquidations fail in MultiLiquidate
- [#1413](https://github.com/NibiruChain/nibiru/pull/1413) - fix(perp): provide descriptive errors when all liquidations fail in MultiLiquidate
- [#1397](https://github.com/NibiruChain/nibiru/pull/1397) - fix: ensure margin is high enough when removing it
- [#1383](https://github.com/NibiruChain/nibiru/pull/1383) - feat: enforce contract to be whitelisted when calling perp bindings
- [#1379](https://github.com/NibiruChain/nibiru/pull/1379) - feat(perp): check for denom in add/remove margin
- [#1210](https://github.com/NibiruChain/nibiru/pull/1210) - fix(ci): fix docker push workflow

#### Else (Breaking)

- [#1477](https://github.com/NibiruChain/nibiru/pull/1477) - refactor(oracle)!: Move away from deprecated events to typed events in x/oracle
- [#1477](https://github.com/NibiruChain/nibiru/pull/1477) - refactor(oracle)!: Move away from deprecated events to typed events in x/oracle
- [#1473](https://github.com/NibiruChain/nibiru/pull/1473) - refactor(perp)!: rename `OpenPosition` to `MarketOrder`
- [#1473](https://github.com/NibiruChain/nibiru/pull/1473) - refactor(perp)!: rename `OpenPosition` to `MarketOrder`
- [#1427](https://github.com/NibiruChain/nibiru/pull/1427) - refactor(perp)!: PositionChangedEvent `MarginToUser`
- [#1427](https://github.com/NibiruChain/nibiru/pull/1427) - refactor(perp)!: PositionChangedEvent `MarginToUser`
- [#1426](https://github.com/NibiruChain/nibiru/pull/1426) - refactor(perp): remove price fluctuation limit check
- [#1388](https://github.com/NibiruChain/nibiru/pull/1388) - refactor(perp)!: idempotent position changed event
- [#1388](https://github.com/NibiruChain/nibiru/pull/1388) - refactor(perp)!: idempotent position changed event
- [#1385](https://github.com/NibiruChain/nibiru/pull/1385) - test(perp): add clearing house negative tests
- [#1385](https://github.com/NibiruChain/nibiru/pull/1385) - test(perp): add clearing house negative tests
- [#1382](https://github.com/NibiruChain/nibiru/pull/1382) - refactor(perp)!: remove `perpv1`
- [#1382](https://github.com/NibiruChain/nibiru/pull/1382) - refactor(perp)!: remove `perpv1`
- [#1356](https://github.com/NibiruChain/nibiru/pull/1356) - build: Regress wasmvm (v1.1.1), tendermint (v0.34.24), and Cosmos-SDK (v0.45.14) dependencies
- [#1356](https://github.com/NibiruChain/nibiru/pull/1356) - build: Regress wasmvm (v1.1.1), tendermint (v0.34.24), and Cosmos-SDK (v0.45.14) dependencies
- [#1346](https://github.com/NibiruChain/nibiru/pull/1346) - build: Upgrade wasmvm (v1.2.1), tendermint (v0.34.26), and Cosmos-SDK (v0.45.14) dependencies
- [#1346](https://github.com/NibiruChain/nibiru/pull/1346) - build: Upgrade wasmvm (v1.2.1), tendermint (v0.34.26), and Cosmos-SDK (v0.45.14) dependencies

### Non-breaking/Compatible Improvements

- [#1579](https://github.com/NibiruChain/nibiru/pull/1579) - chore(proto): Add a buf.gen.rs.yaml and corresponding script to create Rust types for Wasm Stargate messages
- [#1574](https://github.com/NibiruChain/nibiru/pull/1574) - chore(goreleaser): update wasmvm to v1.4.0
- [#1558](https://github.com/NibiruChain/nibiru/pull/1558) - feat(perp): paginated query to read the position store
- [#1555](https://github.com/NibiruChain/nibiru/pull/1555) - feat(devgas): Convert legacy ABCI events to typed proto events
- [#1554](https://github.com/NibiruChain/nibiru/pull/1554) - refactor: runs gofumpt formatter, which has nice conventions: go install mvdan.cc/gofumpt@latest
- [#1536](https://github.com/NibiruChain/nibiru/pull/1536) - test(perp): add more tests to perp module and cli
- [#1533](https://github.com/NibiruChain/nibiru/pull/1533) - feat(perp): add differential fields to PositionChangedEvent
- [#1527](https://github.com/NibiruChain/nibiru/pull/1527) - test(common): add docs for testutil and increase test coverage
- [#1521](https://github.com/NibiruChain/nibiru/pull/1521) - test(sudo): increase unit test coverage
- [#1519](https://github.com/NibiruChain/nibiru/pull/1519) - test: add more tests to x/perp keeper
- [#1518](https://github.com/NibiruChain/nibiru/pull/1518) - test: add more tests to x/perp
- [#1517](https://github.com/NibiruChain/nibiru/pull/1517) - test: add more tests to x/hooks
- [#1506](https://github.com/NibiruChain/nibiru/pull/1506) - refactor(oracle): Implement OrderedMap and use it for iterating through maps in x/oracle
- [#1500](https://github.com/NibiruChain/nibiru/pull/1500) - refactor(perp): clean up reverse market order mechanics
- [#1466](https://github.com/NibiruChain/nibiru/pull/1466) - refactor(perp): `PositionLiquidatedEvent`
- [#1466](https://github.com/NibiruChain/nibiru/pull/1466) - refactor(perp): `PositionLiquidatedEvent`
- [#1462](https://github.com/NibiruChain/nibiru/pull/1462) - fix(perp): Add pair to liquidation failed event.
- [#1424](https://github.com/NibiruChain/nibiru/pull/1424) - feat(perp): Add change type and exchanged margin to position changed events.
- [#1408](https://github.com/NibiruChain/nibiru/pull/1408) - feat(spot): idempotent events
- [#1406](https://github.com/NibiruChain/nibiru/pull/1406) - feat(perp): emit additional event info
- [#1405](https://github.com/NibiruChain/nibiru/pull/1405) - ci: use Buf to build protos
- [#1390](https://github.com/NibiruChain/nibiru/pull/1390) - fix(localnet.sh): Fix genesis market initialization + add force exits on failure
- [#1369](https://github.com/NibiruChain/nibiru/pull/1369) - refactor(oracle): divert rewards from `perpv2` instead of `perpv1`
- [#1365](https://github.com/NibiruChain/nibiru/pull/1365) - refactor(perp): split `perp` module into v1/ and v2/

### Dependencies

- [#1523](https://github.com/NibiruChain/nibiru/pull/1523) - chore: bump cosmos-sdk to v0.47.4
- [#1381](https://github.com/NibiruChain/nibiru/pull/1381) - chore(deps): Bump github.com/cosmos/cosmos-sdk to 0.45.16

- Bump `github.com/docker/distribution` from 2.8.1+incompatible to 2.8.2+incompatible (#1339)

- Bump `github.com/CosmWasm/wasmvm` from 1.2.1 to 1.4.0 (#1354, #1507, [#1564](https://github.com/NibiruChain/nibiru/pull/1564))
- Bump `github.com/spf13/cast` from 1.5.0 to 1.5.1 (#1358)
- Bump `github.com/stretchr/testify` from 1.8.2 to 1.8.4 (#1384, #1435)
- Bump `cosmossdk.io/math` from 1.0.0-beta.6 to 1.1.2 (#1394, [#1547](https://github.com/NibiruChain/nibiru/pull/1547))
- Bump `google.golang.org/grpc` from 1.53.0 to 1.58.2 (#1395, #1437, #1443, #1497, [#1525](https://github.com/NibiruChain/nibiru/pull/1525), [#1568](https://github.com/NibiruChain/nibiru/pull/1568), [#1582](https://github.com/NibiruChain/nibiru/pull/1582), [#1598](https://github.com/NibiruChain/nibiru/pull/1598))
- Bump `github.com/gin-gonic/gin` from 1.8.1 to 1.9.1 (#1409)
- Bump `github.com/spf13/viper` from 1.15.0 to 1.16.0 (#1436)
- Bump `github.com/prometheus/client_golang` from 1.15.1 to 1.16.0 (#1431)
- Bump `github.com/cosmos/ibc-go/v7` from 7.1.0 to 7.3.0 (#1445, [#1562](https://github.com/NibiruChain/nibiru/pull/1562))
- Bump `bufbuild/buf-setup-action` from 1.21.0 to 1.26.1 (#1449, #1469, #1505, #1510, [#1537](https://github.com/NibiruChain/nibiru/pull/1537), [#1540](https://github.com/NibiruChain/nibiru/pull/1540), [#1544](https://github.com/NibiruChain/nibiru/pull/1544))
- Bump `google.golang.org/protobuf` from 1.30.0 to 1.31.0 (#1450)
- Bump `cosmossdk.io/errors` from 1.0.0-beta.7 to 1.0.0 (#1499)
- Bump `github.com/holiman/uint256` from 1.2.2 to 1.2.3 (#1504)
- Bump `docker/build-push-action` from 4 to 5 ([#1572](https://github.com/NibiruChain/nibiru/pull/1572))
- Bump `docker/login-action` from 2 to 3 ([#1571](https://github.com/NibiruChain/nibiru/pull/1571))
- Bump `docker/setup-buildx-action` from 2 to 3 ([#1570](https://github.com/NibiruChain/nibiru/pull/1570))
- Bump `docker/setup-qemu-action` from 2 to 3 ([#1569](https://github.com/NibiruChain/nibiru/pull/1569))
- Bump `github.com/cosmos/cosmos-sdk` from v0.47.4 to v0.47.5 ([#1578](https://github.com/NibiruChain/nibiru/pull/1578))
- Bump `codecov/codecov-action` from 3 to 4 ([#1583](https://github.com/NibiruChain/nibiru/pull/1583))
- Bump `actions/checkout` from 3 to 4 ([#1593](https://github.com/NibiruChain/nibiru/pull/1593))
- Bump `github.com/docker/distribution` from 2.8.1+incompatible to 2.8.2+incompatible (#1339)
- Bump `github.com/CosmWasm/wasmvm` from 1.2.1 to 1.3.0 (#1354, #1507)
- Bump `github.com/spf13/cast` from 1.5.0 to 1.5.1 (#1358)
- Bump `github.com/stretchr/testify` from 1.8.2 to 1.8.4 (#1384, #1435)
- Bump `cosmossdk.io/math` from 1.0.0-beta.6 to 1.1.2 (#1394, [#1547](https://github.com/NibiruChain/nibiru/pull/1547))
- Bump `google.golang.org/grpc` from 1.53.0 to 1.57.0 (#1395, #1437, #1443, #1497, [#1525](https://github.com/NibiruChain/nibiru/pull/1525))
- Bump `github.com/gin-gonic/gin` from 1.8.1 to 1.9.1 (#1409)
- Bump `github.com/spf13/viper` from 1.15.0 to 1.16.0 (#1436)
- Bump `github.com/prometheus/client_golang` from 1.15.1 to 1.16.0 (#1431)
- Bump `github.com/cosmos/ibc-go/v7` from 7.1.0 to 7.3.0 (#1445, [#1562](https://github.com/NibiruChain/nibiru/pull/1562))
- Bump `bufbuild/buf-setup-action` from 1.21.0 to 1.26.1 (#1449, #1469, #1505, #1510, [#1537](https://github.com/NibiruChain/nibiru/pull/1537), [#1540](https://github.com/NibiruChain/nibiru/pull/1540), [#1544](https://github.com/NibiruChain/nibiru/pull/1544))
- Bump `google.golang.org/protobuf` from 1.30.0 to 1.31.0 (#1450)
- Bump `cosmossdk.io/errors` from 1.0.0-beta.7 to 1.0.0 (#1499)
- Bump `github.com/holiman/uint256` from 1.2.2 to 1.2.3 (#1504)
- Bump `actions/checkout` from 3 to 4 ([#1563](https://github.com/NibiruChain/nibiru/pull/1563))

## [v0.19.4] - 2023-05-26

- Summary: Changes up to pull request #1337
- [[Release Link](https://github.com/NibiruChain/nibiru/releases/tag/v0.19.4)]

### State Machine Breaking

- [#1336](https://github.com/NibiruChain/nibiru/pull/1336) - feat: move oracle params out of params subspace and onto the keeper
- [#1336](https://github.com/NibiruChain/nibiru/pull/1336) - feat: move oracle params out of params subspace and onto the keeper
- [#1335](https://github.com/NibiruChain/nibiru/pull/1335) - refactor(perp): move remaining perpv1 files to v1 directory
- [#1334](https://github.com/NibiruChain/nibiru/pull/1334) - feat(perp): add PerpKeeperV2 `ClosePosition`
- [#1333](https://github.com/NibiruChain/nibiru/pull/1333) - feat(perp): add basic clearing house functions
- [#1332](https://github.com/NibiruChain/nibiru/pull/1332) - feat(perp): add hooks to update funding rate
- [#1331](https://github.com/NibiruChain/nibiru/pull/1331) - refactor(perp): create perp v1 type package and module package
- [#1329](https://github.com/NibiruChain/nibiru/pull/1329) - feat(perp): add PerpKeeperV2 withdraw methods
- [#1328](https://github.com/NibiruChain/nibiru/pull/1328) - feat(perp): add PerpKeeperV2 swap methods
- [#1322](https://gitub.com/NibiruChain/nibiru/pull/1322) - build(deps): Bumps github.com/armon/go-metrics from 0.4.0 to 0.4.1.
- [#1319](https://github.com/NibiruChain/nibiru/pull/1319) - test: add integration test actions
- [#1317](https://github.com/NibiruChain/nibiru/pull/1317) - feat(testutil): Use secp256k1 algo for private key generation in common/testutil.
- [#1317](https://github.com/NibiruChain/nibiru/pull/1317) - feat(sudo): Implement and test CLI commands for tx and queries.
- [#1317](https://github.com/NibiruChain/nibiru/pull/1317) - feat(sudo): Implement and test CLI commands for tx and queries.
- [#1315](https://github.com/NibiruChain/nibiru/pull/1315) - feat: oracle rewards distribution every week
- [#1315](https://github.com/NibiruChain/nibiru/pull/1315) - feat: oracle rewards distribution every week
- [#1312](https://github.com/NibiruChain/nibiru/pull/1312) - feat(wasm): wire depth shift handler to the wasm router
- [#1312](https://github.com/NibiruChain/nibiru/pull/1312) - feat(wasm): wire depth shift handler to the wasm router
- [#1311](https://github.com/NibiruChain/nibiru/pull/1311) - feat(perp): add PerpKeeperV2
- [#1311](https://github.com/NibiruChain/nibiru/pull/1311) - feat(perp): add Calc and Twap methods
- [#1309](https://github.com/NibiruChain/nibiru/pull/1309) - feat: minimum swap amount set to $1
- [#1309](https://github.com/NibiruChain/nibiru/pull/1309) - feat: minimum swap amount set to $1
- [#1308](https://github.com/NibiruChain/nibiru/pull/1308) - feat(perp): ensure there's no int overflow in liq depth calculation
- [#1307](https://github.com/NibiruChain/nibiru/pull/1307) - feat(sudo): Create the x/sudo module + integration tests
- [#1307](https://github.com/NibiruChain/nibiru/pull/1307) - feat(sudo): Create the x/sudo module + integration tests
- [#1306](https://github.com/NibiruChain/nibiru/pull/1306) - feat(perp): complete perp v2 types
- [#1306](https://github.com/NibiruChain/nibiru/pull/1306) - feat(perp): complete perp v2 types
- [#1305](https://github.com/NibiruChain/nibiru/pull/1305) - refactor(perp!): Remove unnecessary protos
- [#1305](https://github.com/NibiruChain/nibiru/pull/1305) - refactor(perp!): Remove unnecessary protos
- [#1304](https://github.com/NibiruChain/nibiru/pull/1304) - feat: db backend - rocksdb
- [#1304](https://github.com/NibiruChain/nibiru/pull/1304) - feat: db backend - rocksdb
- [#1302](https://github.com/NibiruChain/nibiru/pull/1302) - refactor(oracle)!: price snapshot start time inclusive
- [#1302](https://github.com/NibiruChain/nibiru/pull/1302) - refactor(oracle)!: price snapshot start time inclusive
- [#1301](https://github.com/NibiruChain/nibiru/pull/1301) - fix(epochs)!: correct epoch start time
- [#1301](https://github.com/NibiruChain/nibiru/pull/1301) - fix(epochs)!: correct epoch start time
- [#1299](https://github.com/NibiruChain/nibiru/pull/1299) - feat(wasm): Add peg shift bindings
- [#1299](https://github.com/NibiruChain/nibiru/pull/1299) - feat(wasm): Add peg shift bindings
- [#1298](https://github.com/NibiruChain/nibiru/pull/1298) - refactor(perp)!: remove `MaxOracleSpreadRatio` from Perpv2
- [#1298](https://github.com/NibiruChain/nibiru/pull/1298) - refactor(perp)!: remove `MaxOracleSpreadRatio` from Perpv2
- [#1296](https://github.com/NibiruChain/nibiru/pull/1296) - refactor(perp)!: update perp v2 state protos
- [#1296](https://github.com/NibiruChain/nibiru/pull/1296) - refactor(perp)!: update perp v2 state protos
- [#1295](https://github.com/NibiruChain/nibiru/pull/1295) - refactor(app): Organize keepers, store keys, and module manager initialization in app.go
- [#1292](https://github.com/NibiruChain/nibiru/pull/1292) - feat(wasm): Add module bindings for execute calls in x/perp: OpenPosition, ClosePosition, AddMargin, RemoveMargin.
- [#1292](https://github.com/NibiruChain/nibiru/pull/1292) - feat(wasm): Add module bindings for execute calls in x/perp: OpenPosition, ClosePosition, AddMargin, RemoveMargin.
- [#1291](https://github.com/NibiruChain/nibiru/pull/1291) - refactor(perp)!: add perp v2 state protos
- [#1291](https://github.com/NibiruChain/nibiru/pull/1291) - refactor(perp)!: add perp v2 state protos
- [#1290](https://github.com/NibiruChain/nibiru/pull/1290) - refactor: fix quote/base reserve naming convention
- [#1289](https://github.com/NibiruChain/nibiru/pull/1289) - feat: SqrtDepth equal to base reserves when pool creation
- [#1287](https://github.com/NibiruChain/nibiru/pull/1287) - feat(wasm): Add module bindings for custom queries in x/perp: Reserves, AllMarkets, BasePrice, PremiumFraction, Metrics, PerpParams, PerpModuleAccounts
- [#1287](https://github.com/NibiruChain/nibiru/pull/1287) - feat(wasm): Add module bindings for custom queries in x/perp: Reserves, AllMarkets, BasePrice, PremiumFraction, Metrics, PerpParams, PerpModuleAccounts
- [#1286](https://github.com/NibiruChain/nibiru/pull/1286) - feat: bias is zero when creating pool
- [#1284](https://github.com/NibiruChain/nibiru/pull/1284) - feat: fails if base and quote reserves are not equal on CreatePool
- [#1282](https://github.com/NibiruChain/nibiru/pull/1282) - feat(inflation)!: add inflation module
- [#1282](https://github.com/NibiruChain/nibiru/pull/1282) - feat(inflation)!: add inflation module
- [#1281](https://github.com/NibiruChain/nibiru/pull/1281) - feat: add peg multiplier to the pricing logic
- [#1281](https://github.com/NibiruChain/nibiru/pull/1281) - feat: add peg multiplier to the pricing logic
- [#1271](https://github.com/NibiruChain/nibiru/pull/1271) - refactor(perp)!: vpool → perp/amm #2 | imports and renames
- [#1271](https://github.com/NibiruChain/nibiru/pull/1271) - refactor(perp)!: vpool → perp/amm #2 | imports and renames
- [#1270](https://github.com/NibiruChain/nibiru/pull/1270) - refactor(proto)!: lint protos and standardize versioning
- [#1270](https://github.com/NibiruChain/nibiru/pull/1270) - refactor(proto)!: lint protos and standardize versioning
- [#1269](https://github.com/NibiruChain/nibiru/pull/1269) - refactor(perp)!: merge x/util with x/perp
- [#1269](https://github.com/NibiruChain/nibiru/pull/1269) - refactor(perp)!: merge x/util with x/perp
- [#1267](https://github.com/NibiruChain/nibiru/pull/1267) - refactor(perp)!: vpool → perp/amm #1 | Moves types, keeper, and cli
- [#1267](https://github.com/NibiruChain/nibiru/pull/1267) - refactor(perp)!: vpool → perp/amm #1 | Moves types, keeper, and cli
- [#1255](https://github.com/NibiruChain/nibiru/pull/1255) - feat: add peg multiplier field into vpool, which for now defaults to 1
- [#1255](https://github.com/NibiruChain/nibiru/pull/1255) - feat: add peg multiplier field into vpool, which for now defaults to 1
- [#1254](https://github.com/NibiruChain/nibiru/pull/1254) - feat: add bias field into vpool
- [#1254](https://github.com/NibiruChain/nibiru/pull/1254) - feat: add bias field into vpool
- [#1248](https://github.com/NibiruChain/nibiru/pull/1248) - refactor(common): Combine x/testutil and x/common/testutil.
- [#1245](https://github.com/NibiruChain/nibiru/pull/1245) - fix(localnet.sh): force localnet.sh to work even if Coingecko is down
- [#1244](https://github.com/NibiruChain/nibiru/pull/1244) - feat: add typed event for oracle post price
- [#1243](https://github.com/NibiruChain/nibiru/pull/1243) - feat(vpool): sqrt of liquidity depth tracked on pool
- [#1243](https://github.com/NibiruChain/nibiru/pull/1243) - feat(vpool): sqrt of liquidity depth tracked on pool
- [#1240](https://github.com/NibiruChain/nibiru/pull/1240) - ci: Test `make proto-gen` when the proto gen scripts or .proto files change
- [#1237](https://github.com/NibiruChain/nibiru/pull/1237) - feat: reduce gas on openposition
- [#1229](https://github.com/NibiruChain/nibiru/pull/1229) - feat: upgrade ibc to v4.2.0 and wasm v0.30.0
- [#1229](https://github.com/NibiruChain/nibiru/pull/1229) - feat: upgrade ibc to v4.2.0 and wasm v0.30.0
- [#1228](https://github.com/NibiruChain/nibiru/pull/1228) - feat: update github.com/CosmWasm/wasmd 0.29.2
- [#1220](https://github.com/NibiruChain/nibiru/pull/1220) - feat: reduce gas fees when posting price
- [#1220](https://github.com/NibiruChain/nibiru/pull/1220) - feat: reduce gas fees when posting price
- [#1219](https://github.com/NibiruChain/nibiru/pull/1219) - fix(ci): use chaosnet image on chaosnet docker compose
- [#1212](https://github.com/NibiruChain/nibiru/pull/1212) - fix(spot): gracefully handle join spot pool with wrong tokens denom

### Non-breaking/Compatible Improvements

- [#1337](https://github.com/NibiruChain/nibiru/pull/1337) - fix(ci): fix dockerfile with rocksdb
- [#1276](https://github.com/NibiruChain/nibiru/pull/1276) - feat: add ewma function
- [#1218](https://github.com/NibiruChain/nibiru/pull/1218) - ci(release): Publish chaosnet image when tagging a release
- [#1210](https://github.com/NibiruChain/nibiru/pull/1210) - fix(ci): fix docker push workflow

### Dependencies

- Bump `technote-space/get-diff-action` from 4 to 6 (#1327)
- Bump `robinraju/release-downloader` from 1.6 to 1.8 (#1326)
- Bump `pozetroninc/github-action-get-latest-release` from 0.6.0 to 0.7.0 (#1325)
- Bump `actions/setup-go` from 3 to 4 (#1324)

- [#1321](https://github.com/NibiruChain/nibiru/pull/1321) - build(deps): bump github.com/prometheus/client_golang from 1.15.0 to 1.15.1
- [#1256](https://github.com/NibiruChain/nibiru/pull/1256) - chore(deps): bump github.com/spf13/cobra from 1.6.1 to 1.7.0
- [#1231](https://github.com/NibiruChain/nibiru/pull/1231) - chore(deps): bump github.com/cosmos/ibc-go/v4 from 4.2.0 to 4.3.0 #1231
- [#1230](https://github.com/NibiruChain/nibiru/pull/1230) - chore(deps): Bump github.com/holiman/uint256 from 1.2.1 to 1.2.2
- [#1223](https://github.com/NibiruChain/nibiru/pull/1223) - chore(deps): bump github.com/golang/protobuf from 1.5.2 to 1.5.3
- [#1222](https://github.com/NibiruChain/nibiru/pull/1222) - chore(deps): bump google.golang.org/protobuf from 1.28.2-0.20220831092852-f930b1dc76e8 to 1.29.0
- [#1211](https://github.com/NibiruChain/nibiru/pull/1211) - chore(deps): Bump github.com/stretchr/testify from 1.8.1 to 1.8.2

- [#1283](https://github.com/NibiruChain/nibiru/pull/1283) - chore(deps): bump github.com/prometheus/client_golang from 1.14.0 to 1.15.0

## [v0.19.2](https://github.com/NibiruChain/nibiru/releases/tag/v0.19.2) - 2023-02-24

Summary: Changes up to pull request #1208

### State Machine Breaking

- [#1196](https://github.com/NibiruChain/nibiru/pull/1196) - refactor(spot)!: default whitelisted asset and query cli
- [#1195](https://github.com/NibiruChain/nibiru/pull/1195) - feat(perp)!: Add `MultiLiquidation` feature for perps
- [#1194](https://github.com/NibiruChain/nibiru/pull/1194) - fix(oracle): local min voters
- [#1187](https://github.com/NibiruChain/nibiru/pull/1187) - feat(oracle): default vote threshold and min voters
- [#1176](https://github.com/NibiruChain/nibiru/pull/1176) - refactor(spot)!: replace `x/dex` module with `x/spot`.
- [#1173](https://github.com/NibiruChain/nibiru/pull/1173) - refactor(spot)!: replace `x/dex` module with `x/spot`.
- [#1171](https://github.com/NibiruChain/nibiru/pull/1171) - refactor(asset)!: Replace `common.AssetPair` with `asset.Pair`.
- [#1164](https://github.com/NibiruChain/nibiru/pull/1164) - refactor: remove client interface for liquidate msg
- [#1158](https://github.com/NibiruChain/nibiru/pull/1158) - feat(asset-registry)!: Add `AssetRegistry`
- [#1156](https://github.com/NibiruChain/nibiru/pull/1156) - refactor: remove lockup & incentivation module
- [#1154](https://github.com/NibiruChain/nibiru/pull/1154) - refactor(asset-pair)!: refactors `common.AssetPair` as an extension of string
- [#1151](https://github.com/NibiruChain/nibiru/pull/1151) - fix(dex): fix swap calculation for stableswap pools
- [#1131](https://github.com/NibiruChain/nibiru/pull/1131) - fix(oracle): use correct distribution module account

### Non-breaking/Compatible Improvements

- [#1205](https://github.com/NibiruChain/nibiru/pull/1205) - test: first testing framework skeleton and example
- [#1203](https://github.com/NibiruChain/nibiru/pull/1203) - ci: make chaosnet pull nibiru image if --build is not specified
- [#1199](https://github.com/NibiruChain/nibiru/pull/1199) - chore(deps): bump golang.org/x/net from 0.4.0 to 0.7.0
- [#1197](https://github.com/NibiruChain/nibiru/pull/1197) - feat: add fees into events in spot module: `EventPoolExited`, `EventPoolCreated`, `EventAssetsSwapped`.
- [#1197](https://github.com/NibiruChain/nibiru/pull/1197) - refactor(testutil): clean up `x/common/testutil` test setup code
- [#1193](https://github.com/NibiruChain/nibiru/pull/1193) - refactor(oracle): clean up `x/oracle/keeper` tests
- [#1192](https://github.com/NibiruChain/nibiru/pull/1192) - feat: chaosnet docker-compose
- [#1191](https://github.com/NibiruChain/nibiru/pull/1191) - fix(oracle): default whitelisted pairs
- [#1190](https://github.com/NibiruChain/nibiru/pull/1190) - ci(release): fix TM_VERSION not being set on releases
- [#1189](https://github.com/NibiruChain/nibiru/pull/1189) - ci(codecov): add Codecov reporting
- [#1188](https://github.com/NibiruChain/nibiru/pull/1188) - fix(spot): remove A precision and clean up borked logic
- [#1184](https://github.com/NibiruChain/nibiru/pull/1184) - docs(oracle): proto type docs, (2) spec clean-up, and (3) remove panic case
- [#1181](https://github.com/NibiruChain/nibiru/pull/1181) - refactor(oracle): keeper method locations
- [#1180](https://github.com/NibiruChain/nibiru/pull/1180) - refactor(oracle): whitelist refactor
- [#1179](https://github.com/NibiruChain/nibiru/pull/1179) - refactor(oracle): types refactor for validator performance map and whitelist map
- [#1165](https://github.com/NibiruChain/nibiru/pull/1165) - chore(deps): bump cosmos-sdk to [v0.45.12](https://github.com/cosmos/cosmos-sdk/blob/release/v0.45.x/CHANGELOG.md#v04512---2023-01-23)
- [#1161](https://github.com/NibiruChain/nibiru/pull/1161) - refactor: migrate simapp tests to use main app
- [#1160](https://github.com/NibiruChain/nibiru/pull/1160) - feat: generic set
- [#1149](https://github.com/NibiruChain/nibiru/pull/1149) - chore(deps): Bump [github.com/btcsuite/btcd](https://github.com/btcsuite/btcd) from 0.22.1 to 0.22.2
- [#1146](https://github.com/NibiruChain/nibiru/pull/1146) - fix: local docker-compose network
- [#1145](https://github.com/NibiruChain/nibiru/pull/1145) - chore: add USD quote asset
- [#1144](https://github.com/NibiruChain/nibiru/pull/1144) - ci: release for linux and darwin (arm64 and amd64)
- [#1141](https://github.com/NibiruChain/nibiru/pull/1141) - refactor(oracle): rename variables for readability
- [#1139](https://github.com/NibiruChain/nibiru/pull/1139) - feat: add default oracle whitelisted pairs
- [#1138](https://github.com/NibiruChain/nibiru/pull/1138) - refactor: put Makefile workflows in separate directory
- [#1135](https://github.com/NibiruChain/nibiru/pull/1135) - fix: add genesis oracle prices to localnet
- [#1134](https://github.com/NibiruChain/nibiru/pull/1134) - refactor: remove panics from vpool and spillovers from the perp module. It's now impossible to call functions in x/perp that would panic in vpool.
- [#1127](https://github.com/NibiruChain/nibiru/pull/1127) - refactor: remove unnecessary panics from x/dex and x/stablecoin
- [#1126](https://github.com/NibiruChain/nibiru/pull/1126) - test(oracle): stop the tyrannical behavior of TestFuzz_PickReferencePair
- [#1126](https://github.com/NibiruChain/nibiru/pull/1126) - test(oracle): stop the tyrannical behavior of TestFuzz_PickReferencePair
- [#1126](https://github.com/NibiruChain/nibiru/pull/1126) - refactor(perp): remove unnecessary panics
- [#1089](https://github.com/NibiruChain/nibiru/pull/1089) - refactor(deps): Bump [github.com/holiman/uint256](https://github.com/holiman/uint256) from 1.1.1 to 1.2.1 (syntax changes)
- [#1032](https://github.com/NibiruChain/nibiru/pull/1107) - ci: Create e2e wasm contract test

## [v0.16.3] - 2022-12-28

- [[Release Link](https://github.com/NibiruChain/nibiru/releases/tag/v0.16.3)]
  [[Commits](https://github.com/NibiruChain/nibiru/commits/v0.16.3)]

### Features

- [#1115](https://github.com/NibiruChain/nibiru/pull/1115) - feat: improve single asset join calculation
- [#1117](https://github.com/NibiruChain/nibiru/pull/1117) - feat: wire multi-liquidate transaction
- [#1120](https://github.com/NibiruChain/nibiru/pull/1120) - feat: replace pricefeed with oracle

### Bug Fixes

- [#1113](https://github.com/NibiruChain/nibiru/pull/1113) - fix: fix quick simulation issue
- [#1114](https://github.com/NibiruChain/nibiru/pull/1114) - fix(dex): fix single asset join
- [#1116](https://github.com/NibiruChain/nibiru/pull/1116) - fix(dex): unfroze pool when LP share supply of 0
- [#1124](https://github.com/NibiruChain/nibiru/pull/1124) - fix(dex): fix unexpected panic in stableswap calcs

## [v0.16.2] - 2022-12-13

- [[Release Link](https://github.com/NibiruChain/nibiru/releases/tag/v0.16.2)]
  [[Commits](https://github.com/NibiruChain/nibiru/commits/v0.16.2)]

### Features

- [#1032](https://github.com/NibiruChain/nibiru/pull/1032) - feeder: add price provide API and bitfinex price source
- [#1038](https://github.com/NibiruChain/nibiru/pull/1038) - feat(dex): add single asset join
- [#1050](https://github.com/NibiruChain/nibiru/pull/1050) - feat(dex): add stableswap pools
- [#1058](https://github.com/NibiruChain/nibiru/pull/1058) - feature: use collections external lib
- [#1082](https://github.com/NibiruChain/nibiru/pull/1082) - feat(vpool): Add gov proposal for editing the sswap invariant of a vpool..
- [#1092](https://github.com/NibiruChain/nibiru/pull/1092) - refactor(dex)!: revive dex module using intermediate test app
- [#1097](https://github.com/NibiruChain/nibiru/pull/1097) - feat(perp): Track and expose the net size of a pair with a query
- [#1105](https://github.com/NibiruChain/nibiru/pull/1105) - feat(perp): Add (notional) volume to metrics state

### API Breaking

- [#1074](https://github.com/NibiruChain/nibiru/pull/1074) - feat(vpool): Add gov proposal for editing the vpool config without changing the reserves.

### State Machine Breaking

- [#1102](https://github.com/NibiruChain/nibiru/pull/1102) - refactor(perp)!: replace CumulativePremiumFractions array with single value

### Breaking Changes

- [#1074](https://github.com/NibiruChain/nibiru/pull/1074) - feat(vpool): Add gov proposal for editing the vpool config without changing the reserves.

### Improvements

- [#1111](https://github.com/NibiruChain/nibiru/pull/1111) - feat(vpool)!: Use flags and certain default values instead of unnamed args for add-genesis-vpool to improve ease of use
- [#1046](https://github.com/NibiruChain/nibiru/pull/1046) - remove: feeder. The price feeder was moved to an external repo.
- [#1015](https://github.com/NibiruChain/nibiru/pull/1015) - feat(dex): throw error when swap output amount is less than 1
- [#1018](https://github.com/NibiruChain/nibiru/pull/1018) - chore(dex): refactor to match best practice
- [#1024](https://github.com/NibiruChain/nibiru/pull/1024) - refactor(oracle): remove Pair and PairList
- [#1034](https://github.com/NibiruChain/nibiru/pull/1034) - refactor(proto): use proto-typed events x/dex
- [#1035](https://github.com/NibiruChain/nibiru/pull/1035) - refactor(proto): use proto-typed events for epochs
- [#1014](https://github.com/NibiruChain/nibiru/pull/1014) - refactor(oracle): full refactor of EndBlock UpdateExchangeRates() long function
- [#1054](https://github.com/NibiruChain/nibiru/pull/1054) - chore(deps): Bump github.com/cosmos/ibc-go/v3 from 3.3.0 to 3.4.0
- [#1043](https://github.com/NibiruChain/nibiru/pull/1043) - chore(deps): Bump github.com/spf13/cobra from 1.6.0 to 1.6.1
- [#1056](https://github.com/NibiruChain/nibiru/pull/1056) - chore(deps): Bump github.com/prometheus/client_golang from 1.13.0 to 1.13.1
- [#1055](https://github.com/NibiruChain/nibiru/pull/1055) - chore(deps): Bump github.com/spf13/viper from 1.13.0 to 1.14.0
- [#1061](https://github.com/NibiruChain/nibiru/pull/1061) - feat(cmd): hard-code block time parameters in the Tendermint config
- [#1068](https://github.com/NibiruChain/nibiru/pull/1068) - refactor(vpool)!: Remove ReserveSnapshot from the vpool genesis state since reserves are taken automatically on vpool initialization.
- [#1064](https://github.com/NibiruChain/nibiru/pull/1064) - test(wasm): add test for Cosmwasm
- [#1075](https://github.com/NibiruChain/nibiru/pull/1075) - feat(dex): remove possibility to create multiple pools with the same assets
- [#1080](https://github.com/NibiruChain/nibiru/pull/1080) - feat(perp): Add exchanged notional to the position changed event #1080
- [#1082](https://github.com/NibiruChain/nibiru/pull/1082) - feat(localnet.sh): Set genesis prices based on real BTC and ETH prices
- [#1086](https://github.com/NibiruChain/nibiru/pull/1086) - refactor(perp)!: Removed unused field, `LiquidationPenalty`, from `PositionChangedEvent`
- [#1093](https://github.com/NibiruChain/nibiru/pull/1093) - simulation(dex): add simulation tests for stableswap pools
- [#1091](https://github.com/NibiruChain/nibiru/pull/1091) - refactor: Use common.Precision instead of 1_000_000 in the codebase
- [#1109](https://github.com/NibiruChain/nibiru/pull/1109) - refactor(vpool)!: Condense swap SwapXForY and SwapYForX events into SwapEvent

### Bug Fixes

- [#1100](https://github.com/NibiruChain/nibiru/pull/1100) - fix(oracle): fix flaky oracle test
- [#1110](https://github.com/NibiruChain/nibiru/pull/1110) - fix(dex): fix dex issue on unsorted join pool

### CI

- [#1088](https://github.com/NibiruChain/nibiru/pull/1088) - ci: build cross binaries

## v0.15.0

### CI

- [#785](https://github.com/NibiruChain/nibiru/pull/785) - ci: create simulations job

### State Machine Breaking

- [#994](https://github.com/NibiruChain/nibiru/pull/994) - x/oracle refactor to use collections
- [#991](https://github.com/NibiruChain/nibiru/pull/991) - collections refactoring of keys and values
- [#978](https://github.com/NibiruChain/nibiru/pull/978) - x/vpool move state logic to collections
- [#977](https://github.com/NibiruChain/nibiru/pull/977) - x/perp add whitelisted liquidators
- [#960](https://github.com/NibiruChain/nibiru/pull/960) - x/common validate asset pair denoms
- [#952](https://github.com/NibiruChain/nibiru/pull/952) - x/perp move state logic to collections
- [#872](https://github.com/NibiruChain/nibiru/pull/872) - x/perp remove module balances from genesis
- [#878](https://github.com/NibiruChain/nibiru/pull/878) - rename `PremiumFraction` to `FundingRate`
- [#900](https://github.com/NibiruChain/nibiru/pull/900) - refactor x/vpool snapshot state management
- [#904](https://github.com/NibiruChain/nibiru/pull/904) - refactor: change Pool name to VPool in vpool module
- [#894](https://github.com/NibiruChain/nibiru/pull/894) - add the collections package!
- [#897](https://github.com/NibiruChain/nibiru/pull/897) - x/pricefeed - use collections.
- [#933](https://github.com/NibiruChain/nibiru/pull/933) - refactor(perp): remove whitelist and simplify state keys
- [#959](https://github.com/NibiruChain/nibiru/pull/959) - feat(vpool): complete genesis import export
  - removed Params from genesis.
  - added pair into ReserveSnapshot type.
  - added validation of snapshots and snapshots in genesis.
- [#975](https://github.com/NibiruChain/nibiru/pull/975) - fix(perp): funding payment calculations
- [#976](https://github.com/NibiruChain/nibiru/pull/976) - refactor(epochs): refactor to increase readability and some tests
  - EpochInfo.CurrentEpoch changed from int64 to uint64.

### API Breaking

- [#880](https://github.com/NibiruChain/nibiru/pull/880) - refactor `PostRawPrice` return values
- [#900](https://github.com/NibiruChain/nibiru/pull/900) - fix x/vpool twap calculation to be bounded in time
- [#919](https://github.com/NibiruChain/nibiru/pull/919) - refactor(proto): vpool module files consistency
  - MarkPriceChanged renamed to MarkPriceChangedEvent
- [#875](https://github.com/NibiruChain/nibiru/pull/875) - x/perp add MsgMultiLiquidate
- [#979](https://github.com/NibiruChain/nibiru/pull/979) - refactor and clean VPool.

### Improvements

- [#1044](https://github.com/NibiruChain/nibiru/pull/1044) - feat(wasm): cosmwasm module integration
- [#858](https://github.com/NibiruChain/nibiru/pull/858) - fix trading limit ratio check; checks in both directions on both quote and base assets
- [#865](https://github.com/NibiruChain/nibiru/pull/865) - refactor(vpool): clean up interface for CmdGetBaseAssetPrice to use add and remove as directions
- [#868](https://github.com/NibiruChain/nibiru/pull/868) - refactor dex integration tests to be independent between them
- [#876](https://github.com/NibiruChain/nibiru/pull/876) - chore(deps): bump github.com/spf13/viper from 1.12.0 to 1.13.0
- [#879](https://github.com/NibiruChain/nibiru/pull/879) - test(perp): liquidate cli test and genesis fix for testutil initGenFiles
- [#889](https://github.com/NibiruChain/nibiru/pull/889) - feat: decouple keeper from servers in pricefeed module
- [#886](https://github.com/NibiruChain/nibiru/pull/886) - feat: decouple keeper from servers in perp module
- [#901](https://github.com/NibiruChain/nibiru/pull/901) - refactor(vpool): remove `GetUnderlyingPrice` method
- [#902](https://github.com/NibiruChain/nibiru/pull/902) - refactor(common): improve usability of `common.AssetPair`
- [#913](https://github.com/NibiruChain/nibiru/pull/913) - chore(epochs): update x/epochs module
- [#911](https://github.com/NibiruChain/nibiru/pull/911) - test(perp): add `MsgOpenPosition` simulation tests
- [#917](https://github.com/NibiruChain/nibiru/pull/917) - refactor(proto): perp module files consistency
- [#920](https://github.com/NibiruChain/nibiru/pull/920) - refactor(proto): pricefeed module files consistency
- [#926](https://github.com/NibiruChain/nibiru/pull/926) - feat: use spot twap for funding rate calculation
- [#932](https://github.com/NibiruChain/nibiru/pull/932) - refactor(perp): rename premium fraction to funding rate
- [#963](https://github.com/NibiruChain/nibiru/pull/963) - test: add collections api tests
- [#971](https://github.com/NibiruChain/nibiru/pull/971) - chore: use upstream 99designs/keyring module
- [#964](https://github.com/NibiruChain/nibiru/pull/964) - test(vpool): refactor flaky vpool cli test
- [#956](https://github.com/NibiruChain/nibiru/pull/956) - test(perp): partial liquidate unit test
- [#981](https://github.com/NibiruChain/nibiru/pull/981) - chore(testutil): clean up x/testutil packages
- [#980](https://github.com/NibiruChain/nibiru/pull/980) - test(perp): add `MsgClosePosition`, `MsgAddMargin`, and `MsgRemoveMargin` simulation tests
- [#987](https://github.com/NibiruChain/nibiru/pull/987) - feat: create a query that directly returns all module accounts without pagination or iteration
- [#982](https://github.com/NibiruChain/nibiru/pull/982) - improvements for pricefeed genesis
- [#989](https://github.com/NibiruChain/nibiru/pull/989) - test(perp): cli test for AddMargin
- [#1001](https://github.com/NibiruChain/nibiru/pull/1001) - chore(deps): bump github.com/spf13/cobra from 1.5.0 to 1.6.0
- [#1013](https://github.com/NibiruChain/nibiru/pull/1013) - test(vpool): more calc twap tests and documentation
- [#1012](https://github.com/NibiruChain/nibiru/pull/1012) - test(vpool): make vpool simulation with random parameters

### Features

- [#1019](https://github.com/NibiruChain/nibiru/pull/1019) - add fields to the snapshot reserve event
- [#1010](https://github.com/NibiruChain/nibiru/pull/1010) - feeder: initialize oracle feeder core logic
- [#966](https://github.com/NibiruChain/nibiru/pull/966) - collections: add indexed map
- [#852](https://github.com/NibiruChain/nibiru/pull/852) - feat(genesis): add cli command to add pairs at genesis
- [#861](https://github.com/NibiruChain/nibiru/pull/861) - feat: query cumulative funding payments
- [#985](https://github.com/NibiruChain/nibiru/pull/985) - feat: query all active positions for a trader
- [#997](https://github.com/NibiruChain/nibiru/pull/997) - feat: emit `ReserveSnapshotSavedEvent` in vpool EndBlocker
- [#1011](https://github.com/NibiruChain/nibiru/pull/1011) - feat(perp): add DonateToEF cli command
- [#1044](https://github.com/NibiruChain/nibiru/pull/1044) - feat(wasm): cosmwasm module integration

### Fixes

- [#1023](https://github.com/NibiruChain/nibiru/pull/1023) - collections: golang compiler bug
- [#1017](https://github.com/NibiruChain/nibiru/pull/1017) - collections: correctly reports value type and key in case of not found errors.
- [#857](https://github.com/NibiruChain/nibiru/pull/857) - x/perp add proper stateless genesis validation checks
- [#874](https://github.com/NibiruChain/nibiru/pull/874) - fix --home issue with unsafe-reset-all command, updating tendermint to v0.34.21
- [#892](https://github.com/NibiruChain/nibiru/pull/892) - chore: fix localnet script
- [#925](https://github.com/NibiruChain/nibiru/pull/925) - fix(vpool): snapshot iteration
- [#930](https://github.com/NibiruChain/nibiru/pull/930) - fix(vpool): snapshot iteration on mark twap
- [#911](https://github.com/NibiruChain/nibiru/pull/911) - fix(perp): handle issue where no vpool snapshots are found
- [#958](https://github.com/NibiruChain/nibiru/pull/930) - fix(pricefeed): add twap to prices query
- [#961](https://github.com/NibiruChain/nibiru/pull/961) - fix(perp): wire the funding rate query
- [#993](https://github.com/NibiruChain/nibiru/pull/993) - fix(vpool): fluctuation limit check
- [#1000](https://github.com/NibiruChain/nibiru/pull/1000) - chore: bump cosmos-sdk to v0.45.9 to fix ibc bug
- [#1002](https://github.com/NibiruChain/nibiru/pull/1002) - fix: update go.mod dependencies to fix the protocgen script

## v0.14.0

### API Breaking

- [#830](https://github.com/NibiruChain/nibiru/pull/830) - test(vpool): Make missing fields for 'query vpool all-pools' display as empty strings.
  - Improve test coverage of functions used in the query server.
  - Added 'pair' field to the `all-pools` to make the prices array easier to digest
- [#878](https://github.com/NibiruChain/nibiru/pull/878) - rename `funding-payments` query to `funding-rate`

### Improvements

- [#837](https://github.com/NibiruChain/nibiru/pull/837) - simplify makefile, removing unused module creation and usage of new command to add vpool at genesis
- [#836](https://github.com/NibiruChain/nibiru/pull/836) - refactor(genesis): DRY improvements and functions added to localnet.sh for readability
- [#842](https://github.com/NibiruChain/nibiru/pull/842) - use self-hosted runner
- [#843](https://github.com/NibiruChain/nibiru/pull/843) - add timeout to github actions integration tests
- [#847](https://github.com/NibiruChain/nibiru/pull/847) - add command in localnet to whitelist oracle
- [#848](https://github.com/NibiruChain/nibiru/pull/848) - add check max leverage on add vpool in genesis command

### Fixes

- [#850](https://github.com/NibiruChain/nibiru/pull/850) - x/vpool - properly validate vpools at genesis
- [#854](https://github.com/NibiruChain/nibiru/pull/854) - add buildx to the docker release workflow

### Features

- [#827](https://github.com/NibiruChain/nibiru/pull/827) - feat(genesis): add cli command to add vpool at genesis
- [#838](https://github.com/NibiruChain/nibiru/pull/838) - feat(genesis): add cli command to whitelist oracles at genesis
- [#846](https://github.com/NibiruChain/nibiru/pull/846) - x/oracle remove reference pair

## [v0.13.0](https://github.com/NibiruChain/nibiru/releases/tag/v0.13.0) - 2022-08-16

## API Breaking

- [#831](https://github.com/NibiruChain/nibiru/pull/831) - remove modules that are not used in testnet

### CI

- [#795](https://github.com/NibiruChain/nibiru/pull/795) - integration tests run when PR is approved
- [#826](https://github.com/NibiruChain/nibiru/pull/826) - create and push docker image on release

### Improvements

- [#798](https://github.com/NibiruChain/nibiru/pull/798) - fix integration tests caused by PR #786
- [#801](https://github.com/NibiruChain/nibiru/pull/801) - remove unused pair constants
- [#788](https://github.com/NibiruChain/nibiru/pull/788) - add --overwrite flag to the nibid init call of localnet.sh
- [#804](https://github.com/NibiruChain/nibiru/pull/804) - bump ibc-go to v3.1.1
- [#817](https://github.com/NibiruChain/nibiru/pull/817) - Make post prices transactions gasless for whitelisted oracles
- [#818](https://github.com/NibiruChain/nibiru/pull/818) - fix(localnet.sh): add max leverage to vpools in genesis to fix open-position
- [#819](https://github.com/NibiruChain/nibiru/pull/819) - add golangci-linter using docker in Makefile
- [#835](https://github.com/NibiruChain/nibiru/pull/835) - x/oracle cleanup code

### Features

- [#839](https://github.com/NibiruChain/nibiru/pull/839) - x/oracle rewarding
- [#791](https://github.com/NibiruChain/nibiru/pull/791) Add the x/oracle module
- [#811](https://github.com/NibiruChain/nibiru/pull/811) Return the index twap in `QueryPrice` cmd
- [#813](https://github.com/NibiruChain/nibiru/pull/813) - (vpool): Expose mark price, mark TWAP, index price, and k (swap invariant) in the all-pools query
- [#816](https://github.com/NibiruChain/nibiru/pull/816) - Remove tobin tax from x/oracle
- [#810](https://github.com/NibiruChain/nibiru/pull/810) - feat(x/perp): expose 'marginRatioIndex' and block number on QueryPosition
- [#832](https://github.com/NibiruChain/nibiru/pull/832) - x/oracle app wiring

### Documentation

- [#814](https://github.com/NibiruChain/nibiru/pull/814) - docs(perp): Added events specification for the perp module.

## [v0.12.1](https://github.com/NibiruChain/nibiru/releases/tag/v0.12.1) - 2022-08-04

- [#796](https://github.com/NibiruChain/nibiru/pull/796) - fix bug that caused that epochKeeper was nil when running epoch hook from Perp module
- [#793](https://github.com/NibiruChain/nibiru/pull/793) - add a vpool parameter to limit leverage in open position

## [v0.12.0](https://github.com/NibiruChain/nibiru/releases/tag/v0.12.0) - 2022-08-03

### Improvements

- [#775](https://github.com/NibiruChain/nibiru/pull/775) - bump google.golang.org/protobuf from 1.28.0 to 1.28.1
- [#768](https://github.com/NibiruChain/nibiru/pull/768) - add simulation tests to make file
- [#767](https://github.com/NibiruChain/nibiru/pull/767) - add fluctuation limit checks on `OpenPosition`.
- [#786](https://github.com/NibiruChain/nibiru/pull/786) - add genesis params in localnet script.
- [#770](https://github.com/NibiruChain/nibiru/pull/770) - Return err in case of zero time elapsed and zero snapshots on `GetCurrentTWAP` func. If zero time has elapsed, and snapshots exists, return the instantaneous average.

### Bug Fixes

- [#766](https://github.com/NibiruChain/nibiru/pull/766) - Fixed margin ratio calculation for trader position.
- [#776](https://github.com/NibiruChain/nibiru/pull/776) - Fix a bug where the user could open infinite leverage positions
- [#779](https://github.com/NibiruChain/nibiru/pull/779) - Fix issue with released tokens being invalid in `ExitPool`

### Testing

- [#782](https://github.com/NibiruChain/nibiru/pull/782) - replace GitHub test workflows to use make commands
- [#784](https://github.com/NibiruChain/nibiru/pull/784) - fix runsim
- [#783](https://github.com/NibiruChain/nibiru/pull/783) - sanitise inputs for msg swap simulations

## [v0.11.0](https://github.com/NibiruChain/nibiru/releases/tag/v0.11.0) - 2022-07-29

### Documentation

- [#701](https://github.com/NibiruChain/nibiru/pull/701) Add release process guide

### Improvements

- [#715](https://github.com/NibiruChain/nibiru/pull/715) - remove redundant perp.Keeper.SetPosition parameters
- [#718](https://github.com/NibiruChain/nibiru/pull/718) - add guard clauses on OpenPosition (leverage and quote amount != 0)
- [#728](https://github.com/NibiruChain/nibiru/pull/728) - add dependabot file into the project.
- [#723](https://github.com/NibiruChain/nibiru/pull/723) - refactor perp keeper's `RemoveMargin` method
- [#730](https://github.com/NibiruChain/nibiru/pull/730) - update localnet script.
- [#736](https://github.com/NibiruChain/nibiru/pull/736) - Bumps [github.com/spf13/cast](https://github.com/spf13/cast) from 1.4.1 to 1.5.0
- [#735](https://github.com/NibiruChain/nibiru/pull/735) - Bump github.com/spf13/cobra from 1.4.0 to 1.5.0
- [#729](https://github.com/NibiruChain/nibiru/pull/729) - move maintenance margin to the vpool module
- [#741](https://github.com/NibiruChain/nibiru/pull/741) - remove unused code and refactored variable names.
- [#742](https://github.com/NibiruChain/nibiru/pull/742) - Vpools are not tradeable if they have invalid oracle prices.
- [#739](https://github.com/NibiruChain/nibiru/pull/739) - Bump github.com/spf13/viper from 1.11.0 to 1.12.0

### API Breaking

- [#721](https://github.com/NibiruChain/nibiru/pull/721) - Updated proto property names to adhere to standard snake_casing and added Unlock REST endpoint
- [#724](https://github.com/NibiruChain/nibiru/pull/724) - Add position fields in `ClosePositionResponse`.
- [#737](https://github.com/NibiruChain/nibiru/pull/737) - Renamed from property to avoid python name clash

### State Machine Breaking

- [#733](https://github.com/NibiruChain/nibiru/pull/733) - Bump github.com/cosmos/ibc-go/v3 from 3.0.0 to 3.1.0
- [#741](https://github.com/NibiruChain/nibiru/pull/741) - Rename `epoch_identifier` param to `funding_rate_interval`.
- [#745](https://github.com/NibiruChain/nibiru/pull/745) - Updated pricefeed twap calc to use bounded time

### Bug Fixes

- [#746](https://github.com/NibiruChain/nibiru/pull/746) - Pin cosmos-sdk version to v0.45 for proto generation.

## [v0.10.0](https://github.com/NibiruChain/nibiru/releases/tag/v0.10.0) - 2022-07-18

### Improvements

- [#705](https://github.com/NibiruChain/nibiru/pull/705) Refactor PerpKeeper's `AddMargin` method to accept individual fields instead of the entire Msg object.

### API Breaking

- [#709](https://github.com/NibiruChain/nibiru/pull/709) Add fields to `OpenPosition` response.
- [#707](https://github.com/NibiruChain/nibiru/pull/707) Add fluctuation limit checks in vpool methods.
- [#712](https://github.com/NibiruChain/nibiru/pull/712) Add funding rate calculation and `FundingRateChangedEvent`.

### Upgrades

- [#725](https://github.com/NibiruChain/nibiru/pull/725) Add governance handler for creating new virtual pools.
- [#702](https://github.com/NibiruChain/nibiru/pull/702) Add upgrade handler for v0.10.0.

## [v0.9.2](https://github.com/NibiruChain/nibiru/releases/tag/v0.9.2) - 2022-07-11

### Improvements

- [#686](https://github.com/NibiruChain/nibiru/pull/686) Add changelog enforcer to github actions.
- [#681](https://github.com/NibiruChain/nibiru/pull/681) Remove automatic release and leave integration tests when merge into master.
- [#684](https://github.com/NibiruChain/nibiru/pull/684) Reorganize PerpKeeper methods.
- [#690](https://github.com/NibiruChain/nibiru/pull/690) Call `closePositionEntirely` from `ClosePosition`.
- [#689](https://github.com/NibiruChain/nibiru/pull/689) Apply funding rate calculation 48 times per day.

### API Breaking

- [#687](https://github.com/NibiruChain/nibiru/pull/687) Emit `PositionChangedEvent` upon changing margin.
- [#685](https://github.com/NibiruChain/nibiru/pull/685) Represent `PositionChangedEvent` bad debt as Coin.
- [#697](https://github.com/NibiruChain/nibiru/pull/697) Rename pricefeed keeper methods.
- [#689](https://github.com/NibiruChain/nibiru/pull/689) Change liquidation params to 2.5% liquidation fee ratio and 25% partial liquidation ratio.

### Testing

- [#695](https://github.com/NibiruChain/nibiru/pull/695) Add `OpenPosition` integration tests.
- [#692](https://github.com/NibiruChain/nibiru/pull/692) Add test coverage for Perp MsgServer methods.
