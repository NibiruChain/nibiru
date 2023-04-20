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

## Unreleased

### Breaking

* [#1270](https://github.com/NibiruChain/nibiru/pull/1270) - refactor(proto)!: lint protos and standardize versioning
* [#1271](https://github.com/NibiruChain/nibiru/pull/1271) - refactor(perp)!: vpool → perp/amm #2 | imports and renames
* [#1269](https://github.com/NibiruChain/nibiru/pull/1269) - refactor(perp)!: merge x/util with x/perp
* [#1267](https://github.com/NibiruChain/nibiru/pull/1267) - refactor(perp)!: vpool → perp/amm #1 | Moves types, keeper, and cli
* [#1243](https://github.com/NibiruChain/nibiru/pull/1243) - feat(vpool): sqrt of liquidity depth tracked on pool
* [#1220](https://github.com/NibiruChain/nibiru/pull/1220) - feat: reduce gas fees when posting price
* [#1229](https://github.com/NibiruChain/nibiru/pull/1229) - feat: upgrade ibc to v4.2.0 and wasm v0.30.0
* [#1254](https://github.com/NibiruChain/nibiru/pull/1254) - feat: add bias field into vpool
* [#1255](https://github.com/NibiruChain/nibiru/pull/1255) - feat: add peg multiplier field into vpool, which for now defaults to 1
* [#1255](https://github.com/NibiruChain/nibiru/pull/1281) - feat: add peg multiplier to the pricing logic

### Improvements

* [#1248](https://github.com/NibiruChain/nibiru/pull/1248) - refactor(common): Combine x/testutil and x/common/testutil.
* [#1245](https://github.com/NibiruChain/nibiru/pull/1245) - fix(localnet.sh): force localnet.sh to work even if Coingecko is down
* [#1230](https://github.com/NibiruChain/nibiru/pull/1230) - chore(deps): Bump github.com/holiman/uint256 from 1.2.1 to 1.2.2
* [#1240](https://github.com/NibiruChain/nibiru/pull/1240) - ci: Test `make proto-gen` when the proto gen scripts or .proto files change
* [#1199](https://github.com/NibiruChain/nibiru/pull/1199) - chore(deps): bump golang.org/x/net from 0.4.0 to 0.7.0
* [#1211](https://github.com/NibiruChain/nibiru/pull/1211) - chore(deps): Bump github.com/stretchr/testify from 1.8.1 to 1.8.2
* [#1203](https://github.com/NibiruChain/nibiru/pull/1203) - ci: make chaosnet pull nibiru image if --build is not specified
* [#1197](https://github.com/NibiruChain/nibiru/pull/1197) - feat: add fees into events in spot module.
  * add `fees` field into `EventPoolCreated` event.
  * add `fees` field into `EventPoolExited` event.
  * add `fee` field into `EventAssetsSwapped` event.
* [#1222](https://github.com/NibiruChain/nibiru/pull/1222) - chore(deps): bump google.golang.org/protobuf from 1.28.2-0.20220831092852-f930b1dc76e8 to 1.29.0
* [#1223](https://github.com/NibiruChain/nibiru/pull/1223) - chore(deps): bump github.com/golang/protobuf from 1.5.2 to 1.5.3
* [#1205](https://github.com/NibiruChain/nibiru/pull/1205) - test: first testing framework skeleton and example
* [#1228](https://github.com/NibiruChain/nibiru/pull/1228) - feat: update github.com/CosmWasm/wasmd 0.29.2
* [#1244](https://github.com/NibiruChain/nibiru/pull/1244) - feat: add typed event for oracle post price
* [#1237](https://github.com/NibiruChain/nibiru/pull/1237) - feat: reduce gas on openposition
* [#1231](https://github.com/NibiruChain/nibiru/pull/1231) - chore(deps): bump github.com/cosmos/ibc-go/v4 from 4.2.0 to 4.3.0 #1231
* [#1256](https://github.com/NibiruChain/nibiru/pull/1256) - chore(deps): bump github.com/spf13/cobra from 1.6.1 to 1.7.0

### Bug Fixes

* [#1210](https://github.com/NibiruChain/nibiru/pull/1210) - fix(ci): fix docker push workflow

## [v0.19.2](https://github.com/NibiruChain/nibiru/releases/tag/v0.19.2) - 2023-02-24

### Features

* [#1187](https://github.com/NibiruChain/nibiru/pull/1187) - feat(oracle): default vote threshold and min voters
* [#1276](https://github.com/NibiruChain/nibiru/pull/1276) - feat: add ewma function
* [#1284](https://github.com/NibiruChain/nibiru/pull/1284) - feat: fails if base and quote reserves are not equal on CreatePool

### API Breaking

* [#1196](https://github.com/NibiruChain/nibiru/pull/1196) - refactor(spot)!: default whitelisted asset and query cli
* [#1195](https://github.com/NibiruChain/nibiru/pull/1195) - feat(perp)!: Add `MultiLiquidation` feature for perps
* [#1158](https://github.com/NibiruChain/nibiru/pull/1158) - feat(asset-registry)!: Add `AssetRegistry`
* [#1171](https://github.com/NibiruChain/nibiru/pull/1171) - refactor(asset)!: Replace `common.AssetPair` with `asset.Pair`.
* [#1164](https://github.com/NibiruChain/nibiru/pull/1164) - refactor: remove client interface for liquidate msg
* [#1173](https://github.com/NibiruChain/nibiru/pull/1173) - refactor(spot)!: replace `x/dex` module with `x/spot`.
* [#1176](https://github.com/NibiruChain/nibiru/pull/1176) - refactor(spot)!: replace `x/dex` module with `x/spot`.

### State Machine Breaking

* [#1154](https://github.com/NibiruChain/nibiru/pull/1154) - refactor(asset-pair)!: refactors `common.AssetPair` as an extension of string
* [#1156](https://github.com/NibiruChain/nibiru/pull/1156) - refactor: remove lockup & incentivation module

### Improvements

* [#1197](https://github.com/NibiruChain/nibiru/pull/1197) - refactor(testutil): clean up `x/common/testutil` test setup code
* [#1193](https://github.com/NibiruChain/nibiru/pull/1193) - refactor(oracle): clean up `x/oracle/keeper` tests
* [#1192](https://github.com/NibiruChain/nibiru/pull/1192) - feat: chaosnet docker-compose
* [#1191](https://github.com/NibiruChain/nibiru/pull/1191) - fix(oracle): default whitelisted pairs
* [#1189](https://github.com/NibiruChain/nibiru/pull/1189) - ci(codecov): add Codecov reporting
* [#1184](https://github.com/NibiruChain/nibiru/pull/1184) - docs(oracle): proto type docs, (2) spec clean-up, and (3) remove panic case
* [#1181](https://github.com/NibiruChain/nibiru/pull/1181) - refactor(oracle): keeper method locations
* [#1180](https://github.com/NibiruChain/nibiru/pull/1180) - refactor(oracle): whitelist refactor
* [#1179](https://github.com/NibiruChain/nibiru/pull/1179) - refactor(oracle): types refactor for validator performance map and whitelist map
* [#1161](https://github.com/NibiruChain/nibiru/pull/1161) - refactor: migrate simapp tests to use main app
* [#1134](https://github.com/NibiruChain/nibiru/pull/1134) - refactor: remove panics from vpool and spillovers from the perp module. It's now impossible to call functions in x/perp that would panic in vpool.
* [#1127](https://github.com/NibiruChain/nibiru/pull/1127) - refactor: remove unnecessary panics from x/dex and x/stablecoin
* [#1126](https://github.com/NibiruChain/nibiru/pull/1126) - refactor(perp): remove unnecessary panics
* [#1138](https://github.com/NibiruChain/nibiru/pull/1138) - refactor: put Makefile workflows in separate directory
* [#1126](https://github.com/NibiruChain/nibiru/pull/1126) - test(oracle): stop the tyrannical behavior of TestFuzz_PickReferencePair
* [#1135](https://github.com/NibiruChain/nibiru/pull/1135) - fix: add genesis oracle prices to localnet
* [#1141](https://github.com/NibiruChain/nibiru/pull/1141) - refactor(oracle): rename variables for readability
* [#1146](https://github.com/NibiruChain/nibiru/pull/1146) - fix: local docker-compose network
* [#1145](https://github.com/NibiruChain/nibiru/pull/1145) - chore: add USD quote asset
* [#1160](https://github.com/NibiruChain/nibiru/pull/1160) - feat: generic set
* [#1139](https://github.com/NibiruChain/nibiru/pull/1139) - feat: add default oracle whitelisted pairs
* [#1032](https://github.com/NibiruChain/nibiru/pull/1107) - ci: Create e2e wasm contract test
* [#1144](https://github.com/NibiruChain/nibiru/pull/1144) - ci: release for linux and darwin (arm64 and amd64)
* [#1165](https://github.com/NibiruChain/nibiru/pull/1165) - chore(deps): bump cosmos-sdk to [v0.45.12](https://github.com/cosmos/cosmos-sdk/blob/release/v0.45.x/CHANGELOG.md#v04512---2023-01-23)
* [#1149](https://github.com/NibiruChain/nibiru/pull/1149) - chore(deps): Bump [github.com/btcsuite/btcd](https://github.com/btcsuite/btcd) from 0.22.1 to 0.22.2
* [#1089](https://github.com/NibiruChain/nibiru/pull/1089) - refactor(deps): Bump [github.com/holiman/uint256](https://github.com/holiman/uint256) from 1.1.1 to 1.2.1 (syntax changes)
* [#1188](https://github.com/NibiruChain/nibiru/pull/1188) - fix(spot): remove A precision and clean up borked logic
* [#1190](https://github.com/NibiruChain/nibiru/pull/1190) - ci(release): fix TM_VERSION not being set on releases
* [#1218](https://github.com/NibiruChain/nibiru/pull/1218) - ci(release): Publish chaosnet image when tagging a release

### Bug Fixes

* [#1194](https://github.com/NibiruChain/nibiru/pull/1194) - fix(oracle): local min voters
* [#1126](https://github.com/NibiruChain/nibiru/pull/1126) - test(oracle): stop the tyrannical behavior of TestFuzz_PickReferencePair
* [#1131](https://github.com/NibiruChain/nibiru/pull/1131) - fix(oracle): use correct distribution module account
* [#1151](https://github.com/NibiruChain/nibiru/pull/1151) - fix(dex): fix swap calculation for stableswap pools
* [#1210](https://github.com/NibiruChain/nibiru/pull/1210) - fix(ci): fix docker push workflow
* [#1212](https://github.com/NibiruChain/nibiru/pull/1212) - fix(spot): gracefully handle join spot pool with wrong tokens denom
* [#1219](https://github.com/NibiruChain/nibiru/pull/1219) - fix(ci): use chaosnet image on chaosnet docker compose

## [v0.16.3](https://github.com/NibiruChain/nibiru/releases/tag/v0.16.3)

### Features

* [#1115](https://github.com/NibiruChain/nibiru/pull/1115) - feat: improve single asset join calculation
* [#1117](https://github.com/NibiruChain/nibiru/pull/1117) - feat: wire multi-liquidate transaction
* [#1120](https://github.com/NibiruChain/nibiru/pull/1120) - feat: replace pricefeed with oracle

### Bug Fixes

* [#1113](https://github.com/NibiruChain/nibiru/pull/1113) - fix: fix quick simulation issue
* [#1114](https://github.com/NibiruChain/nibiru/pull/1114) - fix(dex): fix single asset join
* [#1116](https://github.com/NibiruChain/nibiru/pull/1116) - fix(dex): unfroze pool when LP share supply of 0
* [#1124](https://github.com/NibiruChain/nibiru/pull/1124) - fix(dex): fix unexpected panic in stableswap calcs

## [v0.16.2](https://github.com/NibiruChain/nibiru/releases/tag/v0.16.2) - Dec 13, 2022

### Features

* [#1032](https://github.com/NibiruChain/nibiru/pull/1032) - feeder: add price provide API and bitfinex price source
* [#1038](https://github.com/NibiruChain/nibiru/pull/1038) - feat(dex): add single asset join
* [#1050](https://github.com/NibiruChain/nibiru/pull/1050) - feat(dex): add stableswap pools
* [#1058](https://github.com/NibiruChain/nibiru/pull/1058) - feature: use collections external lib
* [#1082](https://github.com/NibiruChain/nibiru/pull/1082) - feat(vpool): Add gov proposal for editing the sswap invariant of a vpool..
* [#1092](https://github.com/NibiruChain/nibiru/pull/1092) - refactor(dex)!: revive dex module using intermediate test app
* [#1097](https://github.com/NibiruChain/nibiru/pull/1097) - feat(perp): Track and expose the net size of a pair with a query
* [#1105](https://github.com/NibiruChain/nibiru/pull/1105) - feat(perp): Add (notional) volume to metrics state

### API Breaking

* [#1074](https://github.com/NibiruChain/nibiru/pull/1074) - feat(vpool): Add gov proposal for editing the vpool config without changing the reserves.

### State Machine Breaking

* [#1102](https://github.com/NibiruChain/nibiru/pull/1102) - refactor(perp)!: replace CumulativePremiumFractions array with single value

### Breaking Changes

* [#1074](https://github.com/NibiruChain/nibiru/pull/1074) - feat(vpool): Add gov proposal for editing the vpool config without changing the reserves.

### Improvements

* [#1111](https://github.com/NibiruChain/nibiru/pull/1111) - feat(vpool)!: Use flags and certain default values instead of unnamed args for add-genesis-vpool to improve ease of use
* [#1046](https://github.com/NibiruChain/nibiru/pull/1046) - remove: feeder. The price feeder was moved to an external repo.
* [#1015](https://github.com/NibiruChain/nibiru/pull/1015) - feat(dex): throw error when swap output amount is less than 1
* [#1018](https://github.com/NibiruChain/nibiru/pull/1018) - chore(dex): refactor to match best practice
* [#1024](https://github.com/NibiruChain/nibiru/pull/1024) - refactor(oracle): remove Pair and PairList
* [#1034](https://github.com/NibiruChain/nibiru/pull/1034) - refactor(proto): use proto-typed events x/dex
* [#1035](https://github.com/NibiruChain/nibiru/pull/1035) - refactor(proto): use proto-typed events for epochs
* [#1014](https://github.com/NibiruChain/nibiru/pull/1014) - refactor(oracle): full refactor of EndBlock UpdateExchangeRates() long function
* [#1054](https://github.com/NibiruChain/nibiru/pull/1054) - chore(deps): Bump github.com/cosmos/ibc-go/v3 from 3.3.0 to 3.4.0
* [#1043](https://github.com/NibiruChain/nibiru/pull/1043) - chore(deps): Bump github.com/spf13/cobra from 1.6.0 to 1.6.1
* [#1056](https://github.com/NibiruChain/nibiru/pull/1056) - chore(deps): Bump github.com/prometheus/client_golang from 1.13.0 to 1.13.1
* [#1055](https://github.com/NibiruChain/nibiru/pull/1055) - chore(deps): Bump github.com/spf13/viper from 1.13.0 to 1.14.0
* [#1061](https://github.com/NibiruChain/nibiru/pull/1061) - feat(cmd): hard-code block time parameters in the Tendermint config
* [#1068](https://github.com/NibiruChain/nibiru/pull/1068) - refactor(vpool)!: Remove ReserveSnapshot from the vpool genesis state since reserves are taken automatically on vpool initialization.
* [#1064](https://github.com/NibiruChain/nibiru/pull/1064) - test(wasm): add test for Cosmwasm
* [#1075](https://github.com/NibiruChain/nibiru/pull/1075) - feat(dex): remove possibility to create multiple pools with the same assets
* [#1080](https://github.com/NibiruChain/nibiru/pull/1080) - feat(perp): Add exchanged notional to the position changed event #1080
* [#1082](https://github.com/NibiruChain/nibiru/pull/1082) - feat(localnet.sh): Set genesis prices based on real BTC and ETH prices
* [#1086](https://github.com/NibiruChain/nibiru/pull/1086) - refactor(perp)!: Removed unused field, `LiquidationPenalty`, from `PositionChangedEvent`
* [#1093](https://github.com/NibiruChain/nibiru/pull/1093) - simulation(dex): add simulation tests for stableswap pools
* [#1091](https://github.com/NibiruChain/nibiru/pull/1091) - refactor: Use common.Precision instead of 1_000_000 in the codebase
* [#1109](https://github.com/NibiruChain/nibiru/pull/1109) - refactor(vpool)!: Condense swap SwapXForY and SwapYForX events into SwapEvent

### Bug Fixes

* [#1100](https://github.com/NibiruChain/nibiru/pull/1100) - fix(oracle): fix flaky oracle test
* [#1110](https://github.com/NibiruChain/nibiru/pull/1110) - fix(dex): fix dex issue on unsorted join pool

### CI

* [#1088](https://github.com/NibiruChain/nibiru/pull/1088) - ci: build cross binaries

## v0.15.0

### CI

* [#785](https://github.com/NibiruChain/nibiru/pull/785) - ci: create simulations job

### State Machine Breaking

* [#994](https://github.com/NibiruChain/nibiru/pull/994) - x/oracle refactor to use collections
* [#991](https://github.com/NibiruChain/nibiru/pull/991) - collections refactoring of keys and values
* [#978](https://github.com/NibiruChain/nibiru/pull/978) - x/vpool move state logic to collections
* [#977](https://github.com/NibiruChain/nibiru/pull/977) - x/perp add whitelisted liquidators
* [#960](https://github.com/NibiruChain/nibiru/pull/960) - x/common validate asset pair denoms
* [#952](https://github.com/NibiruChain/nibiru/pull/952) - x/perp move state logic to collections
* [#872](https://github.com/NibiruChain/nibiru/pull/872) - x/perp remove module balances from genesis
* [#878](https://github.com/NibiruChain/nibiru/pull/878) - rename `PremiumFraction` to `FundingRate`
* [#900](https://github.com/NibiruChain/nibiru/pull/900) - refactor x/vpool snapshot state management
* [#904](https://github.com/NibiruChain/nibiru/pull/904) - refactor: change Pool name to VPool in vpool module
* [#894](https://github.com/NibiruChain/nibiru/pull/894) - add the collections package!
* [#897](https://github.com/NibiruChain/nibiru/pull/897) - x/pricefeed - use collections.
* [#933](https://github.com/NibiruChain/nibiru/pull/933) - refactor(perp): remove whitelist and simplify state keys
* [#959](https://github.com/NibiruChain/nibiru/pull/959) - feat(vpool): complete genesis import export
  * removed Params from genesis.
  * added pair into ReserveSnapshot type.
  * added validation of snapshots and snapshots in genesis.
* [#975](https://github.com/NibiruChain/nibiru/pull/975) - fix(perp): funding payment calculations
* [#976](https://github.com/NibiruChain/nibiru/pull/976) - refactor(epochs): refactor to increase readability and some tests
  * EpochInfo.CurrentEpoch changed from int64 to uint64.

### API Breaking

* [#880](https://github.com/NibiruChain/nibiru/pull/880) - refactor `PostRawPrice` return values
* [#900](https://github.com/NibiruChain/nibiru/pull/900) - fix x/vpool twap calculation to be bounded in time
* [#919](https://github.com/NibiruChain/nibiru/pull/919) - refactor(proto): vpool module files consistency
  * MarkPriceChanged renamed to MarkPriceChangedEvent
* [#875](https://github.com/NibiruChain/nibiru/pull/875) - x/perp add MsgMultiLiquidate
* [#979](https://github.com/NibiruChain/nibiru/pull/979) - refactor and clean VPool.

### Improvements

* [#1044](https://github.com/NibiruChain/nibiru/pull/1044) - feat(wasm): cosmwasm module integration
* [#858](https://github.com/NibiruChain/nibiru/pull/858) - fix trading limit ratio check; checks in both directions on both quote and base assets
* [#865](https://github.com/NibiruChain/nibiru/pull/865) - refactor(vpool): clean up interface for CmdGetBaseAssetPrice to use add and remove as directions
* [#868](https://github.com/NibiruChain/nibiru/pull/868) - refactor dex integration tests to be independent between them
* [#876](https://github.com/NibiruChain/nibiru/pull/876) - chore(deps): bump github.com/spf13/viper from 1.12.0 to 1.13.0
* [#879](https://github.com/NibiruChain/nibiru/pull/879) - test(perp): liquidate cli test and genesis fix for testutil initGenFiles
* [#889](https://github.com/NibiruChain/nibiru/pull/889) - feat: decouple keeper from servers in pricefeed module
* [#886](https://github.com/NibiruChain/nibiru/pull/886) - feat: decouple keeper from servers in perp module
* [#901](https://github.com/NibiruChain/nibiru/pull/901) - refactor(vpool): remove `GetUnderlyingPrice` method
* [#902](https://github.com/NibiruChain/nibiru/pull/902) - refactor(common): improve usability of `common.AssetPair`
* [#913](https://github.com/NibiruChain/nibiru/pull/913) - chore(epochs): update x/epochs module
* [#911](https://github.com/NibiruChain/nibiru/pull/911) - test(perp): add `MsgOpenPosition` simulation tests
* [#917](https://github.com/NibiruChain/nibiru/pull/917) - refactor(proto): perp module files consistency
* [#920](https://github.com/NibiruChain/nibiru/pull/920) - refactor(proto): pricefeed module files consistency
* [#926](https://github.com/NibiruChain/nibiru/pull/926) - feat: use spot twap for funding rate calculation
* [#932](https://github.com/NibiruChain/nibiru/pull/932) - refactor(perp): rename premium fraction to funding rate
* [#963](https://github.com/NibiruChain/nibiru/pull/963) - test: add collections api tests
* [#971](https://github.com/NibiruChain/nibiru/pull/971) - chore: use upstream 99designs/keyring module
* [#964](https://github.com/NibiruChain/nibiru/pull/964) - test(vpool): refactor flaky vpool cli test
* [#956](https://github.com/NibiruChain/nibiru/pull/956) - test(perp): partial liquidate unit test
* [#981](https://github.com/NibiruChain/nibiru/pull/981) - chore(testutil): clean up x/testutil packages
* [#980](https://github.com/NibiruChain/nibiru/pull/980) - test(perp): add `MsgClosePosition`, `MsgAddMargin`, and `MsgRemoveMargin` simulation tests
* [#987](https://github.com/NibiruChain/nibiru/pull/987) - feat: create a query that directly returns all module accounts without pagination or iteration
* [#982](https://github.com/NibiruChain/nibiru/pull/982) - improvements for pricefeed genesis
* [#989](https://github.com/NibiruChain/nibiru/pull/989) - test(perp): cli test for AddMargin
* [#1001](https://github.com/NibiruChain/nibiru/pull/1001) - chore(deps): bump github.com/spf13/cobra from 1.5.0 to 1.6.0
* [#1013](https://github.com/NibiruChain/nibiru/pull/1013) - test(vpool): more calc twap tests and documentation
* [#1012](https://github.com/NibiruChain/nibiru/pull/1012) - test(vpool): make vpool simulation with random parameters

### Features

* [#1019](https://github.com/NibiruChain/nibiru/pull/1019) - add fields to the snapshot reserve event
* [#1010](https://github.com/NibiruChain/nibiru/pull/1010) - feeder: initialize oracle feeder core logic
* [#966](https://github.com/NibiruChain/nibiru/pull/966) - collections: add indexed map
* [#852](https://github.com/NibiruChain/nibiru/pull/852) - feat(genesis): add cli command to add pairs at genesis
* [#861](https://github.com/NibiruChain/nibiru/pull/861) - feat: query cumulative funding payments
* [#985](https://github.com/NibiruChain/nibiru/pull/985) - feat: query all active positions for a trader
* [#997](https://github.com/NibiruChain/nibiru/pull/997) - feat: emit `ReserveSnapshotSavedEvent` in vpool EndBlocker
* [#1011](https://github.com/NibiruChain/nibiru/pull/1011) - feat(perp): add DonateToEF cli command
* [#1044](https://github.com/NibiruChain/nibiru/pull/1044) - feat(wasm): cosmwasm module integration

### Fixes

* [#1023](https://github.com/NibiruChain/nibiru/pull/1023) - collections: golang compiler bug
* [#1017](https://github.com/NibiruChain/nibiru/pull/1017) - collections: correctly reports value type and key in case of not found errors.
* [#857](https://github.com/NibiruChain/nibiru/pull/857) - x/perp add proper stateless genesis validation checks
* [#874](https://github.com/NibiruChain/nibiru/pull/874) - fix --home issue with unsafe-reset-all command, updating tendermint to v0.34.21
* [#892](https://github.com/NibiruChain/nibiru/pull/892) - chore: fix localnet script
* [#925](https://github.com/NibiruChain/nibiru/pull/925) - fix(vpool): snapshot iteration
* [#930](https://github.com/NibiruChain/nibiru/pull/930) - fix(vpool): snapshot iteration on mark twap
* [#911](https://github.com/NibiruChain/nibiru/pull/911) - fix(perp): handle issue where no vpool snapshots are found
* [#958](https://github.com/NibiruChain/nibiru/pull/930) - fix(pricefeed): add twap to prices query
* [#961](https://github.com/NibiruChain/nibiru/pull/961) - fix(perp): wire the funding rate query
* [#993](https://github.com/NibiruChain/nibiru/pull/993) - fix(vpool): fluctuation limit check
* [#1000](https://github.com/NibiruChain/nibiru/pull/1000) - chore: bump cosmos-sdk to v0.45.9 to fix ibc bug
* [#1002](https://github.com/NibiruChain/nibiru/pull/1002) - fix: update go.mod dependencies to fix the protocgen script

## v0.14.0

### API Breaking

* [#830](https://github.com/NibiruChain/nibiru/pull/830) - test(vpool): Make missing fields for 'query vpool all-pools' display as empty strings.
  * Improve test coverage of functions used in the query server.
  * Added 'pair' field to the `all-pools` to make the prices array easier to digest
* [#878](https://github.com/NibiruChain/nibiru/pull/878) - rename `funding-payments` query to `funding-rate`

### Improvements

* [#837](https://github.com/NibiruChain/nibiru/pull/837) - simplify makefile, removing unused module creation and usage of new command to add vpool at genesis
* [#836](https://github.com/NibiruChain/nibiru/pull/836) - refactor(genesis): DRY improvements and functions added to localnet.sh for readability
* [#842](https://github.com/NibiruChain/nibiru/pull/842) - use self-hosted runner
* [#843](https://github.com/NibiruChain/nibiru/pull/843) - add timeout to github actions integration tests
* [#847](https://github.com/NibiruChain/nibiru/pull/847) - add command in localnet to whitelist oracle
* [#848](https://github.com/NibiruChain/nibiru/pull/848) - add check max leverage on add vpool in genesis command

### Fixes

* [#850](https://github.com/NibiruChain/nibiru/pull/850) - x/vpool - properly validate vpools at genesis
* [#854](https://github.com/NibiruChain/nibiru/pull/854) - add buildx to the docker release workflow

### Features

* [#827](https://github.com/NibiruChain/nibiru/pull/827) - feat(genesis): add cli command to add vpool at genesis
* [#838](https://github.com/NibiruChain/nibiru/pull/838) - feat(genesis): add cli command to whitelist oracles at genesis
* [#846](https://github.com/NibiruChain/nibiru/pull/846) - x/oracle remove reference pair

## [v0.13.0](https://github.com/NibiruChain/nibiru/releases/tag/v0.13.0) - 2022-08-16

## API Breaking

* [#831](https://github.com/NibiruChain/nibiru/pull/831) - remove modules that are not used in testnet

### CI

* [#795](https://github.com/NibiruChain/nibiru/pull/795) - integration tests run when PR is approved
* [#826](https://github.com/NibiruChain/nibiru/pull/826) - create and push docker image on release

### Improvements

* [#798](https://github.com/NibiruChain/nibiru/pull/798) - fix integration tests caused by PR #786
* [#801](https://github.com/NibiruChain/nibiru/pull/801) - remove unused pair constants
* [#788](https://github.com/NibiruChain/nibiru/pull/788) - add --overwrite flag to the nibid init call of localnet.sh
* [#804](https://github.com/NibiruChain/nibiru/pull/804) - bump ibc-go to v3.1.1
* [#817](https://github.com/NibiruChain/nibiru/pull/817) - Make post prices transactions gasless for whitelisted oracles
* [#818](https://github.com/NibiruChain/nibiru/pull/818) - fix(localnet.sh): add max leverage to vpools in genesis to fix open-position
* [#819](https://github.com/NibiruChain/nibiru/pull/819) - add golangci-linter using docker in Makefile
* [#835](https://github.com/NibiruChain/nibiru/pull/835) - x/oracle cleanup code

### Features

* [#839](https://github.com/NibiruChain/nibiru/pull/839) - x/oracle rewarding
* [#791](https://github.com/NibiruChain/nibiru/pull/791) Add the x/oracle module
* [#811](https://github.com/NibiruChain/nibiru/pull/811) Return the index twap in `QueryPrice` cmd
* [#813](https://github.com/NibiruChain/nibiru/pull/813) - (vpool): Expose mark price, mark TWAP, index price, and k (swap invariant) in the all-pools query
* [#816](https://github.com/NibiruChain/nibiru/pull/816) - Remove tobin tax from x/oracle
* [#810](https://github.com/NibiruChain/nibiru/pull/810) - feat(x/perp): expose 'marginRatioIndex' and block number on QueryPosition
* [#832](https://github.com/NibiruChain/nibiru/pull/832) - x/oracle app wiring

### Documentation

* [#814](https://github.com/NibiruChain/nibiru/pull/814) - docs(perp): Added events specification for the perp module.

## [v0.12.1](https://github.com/NibiruChain/nibiru/releases/tag/v0.12.1) - 2022-08-04

* [#796](https://github.com/NibiruChain/nibiru/pull/796) - fix bug that caused that epochKeeper was nil when running epoch hook from Perp module
* [#793](https://github.com/NibiruChain/nibiru/pull/793) - add a vpool parameter to limit leverage in open position

## [v0.12.0](https://github.com/NibiruChain/nibiru/releases/tag/v0.12.0) - 2022-08-03

### Improvements

* [#775](https://github.com/NibiruChain/nibiru/pull/775) - bump google.golang.org/protobuf from 1.28.0 to 1.28.1
* [#768](https://github.com/NibiruChain/nibiru/pull/768) - add simulation tests to make file
* [#767](https://github.com/NibiruChain/nibiru/pull/767) - add fluctuation limit checks on `OpenPosition`.
* [#786](https://github.com/NibiruChain/nibiru/pull/786) - add genesis params in localnet script.
* [#770](https://github.com/NibiruChain/nibiru/pull/770) - Return err in case of zero time elapsed and zero snapshots on `GetCurrentTWAP` func. If zero time has elapsed, and snapshots exists, return the instantaneous average.

### Bug Fixes

* [#766](https://github.com/NibiruChain/nibiru/pull/766) - Fixed margin ratio calculation for trader position.
* [#776](https://github.com/NibiruChain/nibiru/pull/776) - Fix a bug where the user could open infinite leverage positions
* [#779](https://github.com/NibiruChain/nibiru/pull/779) - Fix issue with released tokens being invalid in `ExitPool`

### Testing

* [#782](https://github.com/NibiruChain/nibiru/pull/782) - replace GitHub test workflows to use make commands
* [#784](https://github.com/NibiruChain/nibiru/pull/784) - fix runsim
* [#783](https://github.com/NibiruChain/nibiru/pull/783) - sanitise inputs for msg swap simulations

## [v0.11.0](https://github.com/NibiruChain/nibiru/releases/tag/v0.11.0) - 2022-07-29

### Documentation

* [#701](https://github.com/NibiruChain/nibiru/pull/701) Add release process guide

### Improvements

* [#715](https://github.com/NibiruChain/nibiru/pull/715) - remove redundant perp.Keeper.SetPosition parameters
* [#718](https://github.com/NibiruChain/nibiru/pull/718) - add guard clauses on OpenPosition (leverage and quote amount != 0)
* [#728](https://github.com/NibiruChain/nibiru/pull/728) - add dependabot file into the project.
* [#723](https://github.com/NibiruChain/nibiru/pull/723) - refactor perp keeper's `RemoveMargin` method
* [#730](https://github.com/NibiruChain/nibiru/pull/730) - update localnet script.
* [#736](https://github.com/NibiruChain/nibiru/pull/736) - Bumps [github.com/spf13/cast](https://github.com/spf13/cast) from 1.4.1 to 1.5.0
* [#735](https://github.com/NibiruChain/nibiru/pull/735) - Bump github.com/spf13/cobra from 1.4.0 to 1.5.0
* [#729](https://github.com/NibiruChain/nibiru/pull/729) - move maintenance margin to the vpool module
* [#741](https://github.com/NibiruChain/nibiru/pull/741) - remove unused code and refactored variable names.
* [#742](https://github.com/NibiruChain/nibiru/pull/742) - Vpools are not tradeable if they have invalid oracle prices.
* [#739](https://github.com/NibiruChain/nibiru/pull/739) - Bump github.com/spf13/viper from 1.11.0 to 1.12.0

### API Breaking

* [#721](https://github.com/NibiruChain/nibiru/pull/721) - Updated proto property names to adhere to standard snake_casing and added Unlock REST endpoint
* [#724](https://github.com/NibiruChain/nibiru/pull/724) - Add position fields in `ClosePositionResponse`.
* [#737](https://github.com/NibiruChain/nibiru/pull/737) - Renamed from property to avoid python name clash

### State Machine Breaking

* [#733](https://github.com/NibiruChain/nibiru/pull/733) - Bump github.com/cosmos/ibc-go/v3 from 3.0.0 to 3.1.0
* [#741](https://github.com/NibiruChain/nibiru/pull/741) - Rename `epoch_identifier` param to `funding_rate_interval`.
* [#745](https://github.com/NibiruChain/nibiru/pull/745) - Updated pricefeed twap calc to use bounded time

### Bug Fixes

* [#746](https://github.com/NibiruChain/nibiru/pull/746) - Pin cosmos-sdk version to v0.45 for proto generation.

## [v0.10.0](https://github.com/NibiruChain/nibiru/releases/tag/v0.10.0) - 2022-07-18

### Improvements

* [#705](https://github.com/NibiruChain/nibiru/pull/705) Refactor PerpKeeper's `AddMargin` method to accept individual fields instead of the entire Msg object.

### API Breaking

* [#709](https://github.com/NibiruChain/nibiru/pull/709) Add fields to `OpenPosition` response.
* [#707](https://github.com/NibiruChain/nibiru/pull/707) Add fluctuation limit checks in vpool methods.
* [#712](https://github.com/NibiruChain/nibiru/pull/712) Add funding rate calculation and `FundingRateChangedEvent`.

### Upgrades

* [#725](https://github.com/NibiruChain/nibiru/pull/725) Add governance handler for creating new virtual pools.
* [#702](https://github.com/NibiruChain/nibiru/pull/702) Add upgrade handler for v0.10.0.

## [v0.9.2](https://github.com/NibiruChain/nibiru/releases/tag/v0.9.2) - 2022-07-11

### Improvements

* [#686](https://github.com/NibiruChain/nibiru/pull/686) Add changelog enforcer to github actions.
* [#681](https://github.com/NibiruChain/nibiru/pull/681) Remove automatic release and leave integration tests when merge into master.
* [#684](https://github.com/NibiruChain/nibiru/pull/684) Reorganize PerpKeeper methods.
* [#690](https://github.com/NibiruChain/nibiru/pull/690) Call `closePositionEntirely` from `ClosePosition`.
* [#689](https://github.com/NibiruChain/nibiru/pull/689) Apply funding rate calculation 48 times per day.

### API Breaking

* [#687](https://github.com/NibiruChain/nibiru/pull/687) Emit `PositionChangedEvent` upon changing margin.
* [#685](https://github.com/NibiruChain/nibiru/pull/685) Represent `PositionChangedEvent` bad debt as Coin.
* [#697](https://github.com/NibiruChain/nibiru/pull/697) Rename pricefeed keeper methods.
* [#689](https://github.com/NibiruChain/nibiru/pull/689) Change liquidation params to 2.5% liquidation fee ratio and 25% partial liquidation ratio.

### Testing

* [#695](https://github.com/NibiruChain/nibiru/pull/695) Add `OpenPosition` integration tests.
* [#692](https://github.com/NibiruChain/nibiru/pull/692) Add test coverage for Perp MsgServer methods.
