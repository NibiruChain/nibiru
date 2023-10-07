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

### Features 

* [#1596](https://github.com/NibiruChain/nibiru/pull/1596) - epic(tokenfactory):
  State transitions, collections, genesis import and export, and app wiring
  * [#1607](https://github.com/NibiruChain/nibiru/pull/1607) - Token factory
    transaction messages for CreateDenom, ChangeAdmin, and UpdateModuleParams 
  * [#1620](https://github.com/NibiruChain/nibiru/pull/1620) - Token factory
    transaction messages for Mint and Burn

### State Machine Breaking 

* [#1609](https://github.com/NibiruChain/nibiru/pull/1609) - refactor(app)!: Remove x/stablecoin module.
* [#1613](https://github.com/NibiruChain/nibiru/pull/1613) - feat(app)!: enforce min commission by changing default and genesis validation
* [#1615](https://github.com/NibiruChain/nibiru/pull/1613) - feat(ante)!: Ante
  handler to add a maximum commission rate of 25% for validators.
* [#1616](https://github.com/NibiruChain/nibiru/pull/1616) - fix(app)!:
  Add custom wasm snapshotter for proper state exports
* [#1617](https://github.com/NibiruChain/nibiru/pull/1617) - fix(app)!:
  non-nil snapshot manager is not guarantted in testapp

### Improvements

* [#1610](https://github.com/NibiruChain/nibiru/pull/1610) - refactor(app):
  Simplify app.go with less redundant imports using struct embedding.
* [#1614](https://github.com/NibiruChain/nibiru/pull/1614) - refactor(proto): Use
  explicit namespacing on proto imports for #1608

### Dependencies
- Bump `github.com/prometheus/client_golang` from 1.16.0 to 1.17.0 ([#1605](https://github.com/NibiruChain/nibiru/pull/1605))
- Bump `bufbuild/buf-setup-action` from 1.26.1 to 1.27.0 ([#1624](https://github.com/NibiruChain/nibiru/pull/1624))
- Bump `stefanzweifel/git-auto-commit-action` from 4 to 5 ([#1625](https://github.com/NibiruChain/nibiru/pull/1625))

## [v0.21.10]

### Features

* [#1575](https://github.com/NibiruChain/nibiru/pull/1575) - feat(perp): Add trader volume tracking
* [#1463](https://github.com/NibiruChain/nibiru/pull/1463) - feat(oracle): add genesis pricefeeder delegation
* [#1479](https://github.com/NibiruChain/nibiru/pull/1479) - feat(perp): implement `PartialClose`
* [#1498](https://github.com/NibiruChain/nibiru/pull/1498) - feat: add cli to change root sudo command
* [#1501](https://github.com/NibiruChain/nibiru/pull/1501) - feat(localnet.sh): (1) Make it possible to run while offline. (2) Implement --no-build option to use the script with the current `nibid` installed.
* [#1501](https://github.com/NibiruChain/nibiru/pull/1501) - feat(proto): add Python buf generation logic for py-sdk
* [#1503](https://github.com/NibiruChain/nibiru/pull/1503) - feat(wasm): add Oracle Exchange Rate query for wasm
* [#1543](https://github.com/NibiruChain/nibiru/pull/1543) - epic(devgas): devgas module for incentivizing smart contract
* [#1559](https://github.com/NibiruChain/nibiru/pull/1559) - feat: add versions to markets to allow to disable them
* [#1585](https://github.com/NibiruChain/nibiru/pull/1585) - feat: include flag versioned in query markets to allow to query disabled markets
* [#1594](https://github.com/NibiruChain/nibiru/pull/1594) - feat: add user discounts

### Improvements

* [#1466](https://github.com/NibiruChain/nibiru/pull/1466) - refactor(perp): `PositionLiquidatedEvent`
* [#1494](https://github.com/NibiruChain/nibiru/pull/1494) - feat: create cli to add sudo account into genesis
* [#1493](https://github.com/NibiruChain/nibiru/pull/1493) - fix(perp): allow `ClosePosition` when there is bad debt
* [#1500](https://github.com/NibiruChain/nibiru/pull/1500) - refactor(perp): clean up reverse market order mechanics
* [#1506](https://github.com/NibiruChain/nibiru/pull/1506) - refactor(oracle): Implement OrderedMap and use it for iterating through maps in x/oracle
* [#1502](https://github.com/NibiruChain/nibiru/pull/1502) - feat: add ledger build support
* [#1495](https://github.com/NibiruChain/nibiru/pull/1495) - feat: add genmsg module
* [#1517](https://github.com/NibiruChain/nibiru/pull/1517) - test: add more tests to x/hooks
* [#1518](https://github.com/NibiruChain/nibiru/pull/1518) - test: add more tests to x/perp
* [#1519](https://github.com/NibiruChain/nibiru/pull/1519) - test: add more tests to x/perp keeper
* [#1520](https://github.com/NibiruChain/nibiru/pull/1520) - feat(wasm): no op handler + tests with updated contracts
* [#1521](https://github.com/NibiruChain/nibiru/pull/1521) - test(sudo): increase unit test coverage
* [#1523](https://github.com/NibiruChain/nibiru/pull/1523) - chore: bump cosmos-sdk to v0.47.4
* [#1527](https://github.com/NibiruChain/nibiru/pull/1527) - test(common): add docs for testutil and increase test coverage
* [#1536](https://github.com/NibiruChain/nibiru/pull/1536) - test(perp): add more tests to perp module and cli
* [#1533](https://github.com/NibiruChain/nibiru/pull/1533) - feat(perp): add differential fields to PositionChangedEvent
* [#1541](https://github.com/NibiruChain/nibiru/pull/1541) - feat(perp): add clamp to premium fractions
* [#1555](https://github.com/NibiruChain/nibiru/pull/1555) - feat(devgas): Convert legacy ABCI events to typed proto events
* [#1558](https://github.com/NibiruChain/nibiru/pull/1558) - feat(perp): paginated query to read the position store  
* [#1554](https://github.com/NibiruChain/nibiru/pull/1554) - refactor: runs gofumpt formatter, which has nice conventions: go install mvdan.cc/gofumpt@latest
* [#1574](https://github.com/NibiruChain/nibiru/pull/1574) - chore(goreleaser): update wasmvm to v1.4.0
* [#1463](https://github.com/NibiruChain/nibiru/pull/1463) - feat(oracle): add genesis pricefeeder delegation
* [#1466](https://github.com/NibiruChain/nibiru/pull/1466) - refactor(perp): `PositionLiquidatedEvent`
* [#1462](https://github.com/NibiruChain/nibiru/pull/1462) - fix(perp): Add pair to liquidation failed event.
* [#1424](https://github.com/NibiruChain/nibiru/pull/1424) - feat(perp): Add change type and exchanged margin to position changed events.
* [#1390](https://github.com/NibiruChain/nibiru/pull/1390) - fix(localnet.sh): Fix genesis market initialization + add force exits on failure
* [#1340](https://github.com/NibiruChain/nibiru/pull/1340) - feat(wasm): Enforce x/sudo contract permission checks on the shifter contract + integration tests
* [#1317](https://github.com/NibiruChain/nibiru/pull/1317) - feat(testutil): Use secp256k1 algo for private key generation in common/testutil.
* [#1322](https://gitub.com/NibiruChain/nibiru/pull/1322) - build(deps): Bumps github.com/armon/go-metrics from 0.4.0 to 0.4.1.
* [#1321](https://github.com/NibiruChain/nibiru/pull/1321) - build(deps): bump github.com/prometheus/client_golang from 1.15.0 to 1.15.1
* [#1295](https://github.com/NibiruChain/nibiru/pull/1295) - refactor(app): Organize keepers, store keys, and module manager initialization in app.go
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
* [#1289](https://github.com/NibiruChain/nibiru/pull/1289) - feat: SqrtDepth equal to base reserves when pool creation
* [#1290](https://github.com/NibiruChain/nibiru/pull/1290) - refactor: fix quote/base reserve naming convention
* [#1311](https://github.com/NibiruChain/nibiru/pull/1311) - feat(perp): add PerpKeeperV2
* [#1308](https://github.com/NibiruChain/nibiru/pull/1308) - feat(perp): ensure there's no int overflow in liq depth calculation
* [#1311](https://github.com/NibiruChain/nibiru/pull/1311) - feat(perp): add Calc and Twap methods
* [#1319](https://github.com/NibiruChain/nibiru/pull/1319) - test: add integration test actions
* [#1329](https://github.com/NibiruChain/nibiru/pull/1329) - feat(perp): add PerpKeeperV2 withdraw methods
* [#1328](https://github.com/NibiruChain/nibiru/pull/1328) - feat(perp): add PerpKeeperV2 swap methods
* [#1331](https://github.com/NibiruChain/nibiru/pull/1331) - refactor(perp): create perp v1 type package and module package
* [#1333](https://github.com/NibiruChain/nibiru/pull/1333) - feat(perp): add basic clearing house functions
* [#1332](https://github.com/NibiruChain/nibiru/pull/1332) - feat(perp): add hooks to update funding rate
* [#1334](https://github.com/NibiruChain/nibiru/pull/1334) - feat(perp): add PerpKeeperV2 `ClosePosition`
* [#1335](https://github.com/NibiruChain/nibiru/pull/1335) - refactor(perp): move remaining perpv1 files to v1 directory
* [#1338](https://github.com/NibiruChain/nibiru/pull/1338) - feat(perp): V2 OpenPosition
* [#1344](https://github.com/NibiruChain/nibiru/pull/1344) - feat(perp): PerpKeeperV2 `AddMargin` and `RemoveMargin`
* [#1345](https://github.com/NibiruChain/nibiru/pull/1345) - feat(perp): PerpV2 QueryServer
* [#1343](https://github.com/NibiruChain/nibiru/pull/1343) - feat(perp): add PerpKeeperV2 `MultiLiquidate`
* [#1352](https://github.com/NibiruChain/nibiru/pull/1352) - feat(perp): add PerpKeeperV2 `MsgServer`
* [#1350](https://github.com/NibiruChain/nibiru/pull/1350) - feat(perp): `EditPriceMultiplier` and `EditSwapInvariant`
* [#1341](https://github.com/NibiruChain/nibiru/pull/1341) - feat(bindings/oracle): add bindings for oracle module params
* [#1361](https://github.com/NibiruChain/nibiru/pull/1361) - feat(perp): add `PerpV2` module
* [#1363](https://github.com/NibiruChain/nibiru/pull/1363) - feat(perp): wire `PerpV2` module
* [#1365](https://github.com/NibiruChain/nibiru/pull/1365) - refactor(perp): split `perp` module into v1/ and v2/
* [#1366](https://github.com/NibiruChain/nibiru/pull/1366) - feat: fix bindings test in cw_test
* [#1362](https://github.com/NibiruChain/nibiru/pull/1362) - feat(perp): add `perpv2` cli
* [#1369](https://github.com/NibiruChain/nibiru/pull/1369) - refactor(oracle): divert rewards from `perpv2` instead of `perpv1`
* [#1370](https://github.com/NibiruChain/nibiru/pull/1370) - feat(perp): `perpv2` `CreatePool` method
* [#1371](https://github.com/NibiruChain/nibiru/pull/1371) - feat: realize bad debt when a user tries to close his position
* [#1373](https://github.com/NibiruChain/nibiru/pull/1373) - feat(perp): `perpv2` `add-genesis-perp-market` CLI command
* [#1381](https://github.com/NibiruChain/nibiru/pull/1381) - chore(deps): Bump github.com/cosmos/cosmos-sdk to 0.45.16
* [#1405](https://github.com/NibiruChain/nibiru/pull/1405) - ci: use Buf to build protos
* [#1406](https://github.com/NibiruChain/nibiru/pull/1406) - feat(perp): emit additional event info
* [#1419](https://github.com/NibiruChain/nibiru/pull/1419) - fix(spot): add pools to genesis state
* [#1408](https://github.com/NibiruChain/nibiru/pull/1408) - feat(spot): idempotent events
* [#1420](https://github.com/NibiruChain/nibiru/pull/1420) - refactor(oracle): update default params
* [#1421](https://github.com/NibiruChain/nibiru/pull/1421) - feat(oracle): add expiry time to oracle prices
* [#1422](https://github.com/NibiruChain/nibiru/pull/1422) - fix(oracle): handle zero oracle rewards
* [#1426](https://github.com/NibiruChain/nibiru/pull/1426) - refactor(perp): remove price fluctuation limit check
* [#1423](https://github.com/NibiruChain/nibiru/pull/1423) - fix: remove panics from abci hooks
* [#1579](https://github.com/NibiruChain/nibiru/pull/1579) - chore(proto): Add a buf.gen.rs.yaml and corresponding script to create Rust types for Wasm Stargate messages

### Bug Fixes

* [#1459](https://github.com/NibiruChain/nibiru/pull/1459) - fix(spot): wire `x/spot` msgService into app router
* [#1467](https://github.com/NibiruChain/nibiru/pull/1467) - fix(oracle): make `calcTwap` safer
* [#1464](https://github.com/NibiruChain/nibiru/pull/1464) - fix(gov): wire legacy proposal handlers
* [#1586](https://github.com/NibiruChain/nibiru/pull/1586) - fix(sudo): make messages compatible with `Amino`
* [#1210](https://github.com/NibiruChain/nibiru/pull/1210) - fix(ci): fix docker push workflow
* [#1337](https://github.com/NibiruChain/nibiru/pull/1337) - fix(ci): fix dockerfile with rocksdb
* [#1379](https://github.com/NibiruChain/nibiru/pull/1379) - feat(perp): check for denom in add/remove margin
* [#1383](https://github.com/NibiruChain/nibiru/pull/1383) - feat: enforce contract to be whitelisted when calling perp bindings
* [#1397](https://github.com/NibiruChain/nibiru/pull/1397) - fix: ensure margin is high enough when removing it
* [#1417](https://github.com/NibiruChain/nibiru/pull/1417) - fix: run end blocker on block end for perp v2
* [#1425](https://github.com/NibiruChain/nibiru/pull/1425) - fix: remove positions from state when closed with reverse position
* [#1441](https://github.com/NibiruChain/nibiru/pull/1441) - fix(oracle): ignore abstain votes in std dev calculation
* [#1446](https://github.com/NibiruChain/nibiru/pull/1446) - fix(cmd): Add custom InitCmd to set set desired Tendermint consensus params for each node.
* [#1452](https://github.com/NibiruChain/nibiru/pull/1452) - fix(oracle): continue with abci hook during error
* [#1451](https://github.com/NibiruChain/nibiru/pull/1451) - fix(perp): decrease position with zero size

### State Machine Breaking

* [#1473](https://github.com/NibiruChain/nibiru/pull/1473) - refactor(perp)!: rename `OpenPosition` to `MarketOrder`
* [#1477](https://github.com/NibiruChain/nibiru/pull/1477) - refactor(oracle)!: Move away from deprecated events to typed events in x/oracle

### API Breaking

* [#1380](https://github.com/NibiruChain/nibiru/pull/1380) - feat(wasm): Add CreateMarket admin call for the controller contract
* [#1359](https://github.com/NibiruChain/nibiru/pull/1359) - feat(perp): Add InsuranceFundWithdraw admin call with corresponding smart contract
* [#1356](https://github.com/NibiruChain/nibiru/pull/1356) - build: Regress wasmvm (v1.1.1), tendermint (v0.34.24), and Cosmos-SDK (v0.45.14) dependencies
* [#1346](https://github.com/NibiruChain/nibiru/pull/1346) - build: Upgrade wasmvm (v1.2.1), tendermint (v0.34.26), and Cosmos-SDK (v0.45.14) dependencies
* [#1317](https://github.com/NibiruChain/nibiru/pull/1317) - feat(sudo): Implement and test CLI commands for tx and queries. 
* [#1307](https://github.com/NibiruChain/nibiru/pull/1307) - feat(sudo): Create the x/sudo module + integration tests
* [#1299](https://github.com/NibiruChain/nibiru/pull/1299) - feat(wasm): Add peg shift bindings
* [#1292](https://github.com/NibiruChain/nibiru/pull/1292) - feat(wasm): Add module bindings for execute calls in x/perp: OpenPosition, ClosePosition, AddMargin, RemoveMargin.
* [#1287](https://github.com/NibiruChain/nibiru/pull/1287) - feat(wasm): Add module bindings for custom queries in x/perp: Reserves, AllMarkets, BasePrice, PremiumFraction, Metrics, PerpParams, PerpModuleAccounts
* [#1282](https://github.com/NibiruChain/nibiru/pull/1282) - feat(inflation)!: add inflation module
* [#1270](https://github.com/NibiruChain/nibiru/pull/1270) - refactor(proto)!: lint protos and standardize versioning
* [#1271](https://github.com/NibiruChain/nibiru/pull/1271) - refactor(perp)!: vpool → perp/amm #2 | imports and renames
* [#1269](https://github.com/NibiruChain/nibiru/pull/1269) - refactor(perp)!: merge x/util with x/perp
* [#1267](https://github.com/NibiruChain/nibiru/pull/1267) - refactor(perp)!: vpool → perp/amm #1 | Moves types, keeper, and cli
* [#1243](https://github.com/NibiruChain/nibiru/pull/1243) - feat(vpool): sqrt of liquidity depth tracked on pool
* [#1220](https://github.com/NibiruChain/nibiru/pull/1220) - feat: reduce gas fees when posting price
* [#1229](https://github.com/NibiruChain/nibiru/pull/1229) - feat: upgrade ibc to v4.2.0 and wasm v0.30.0
* [#1254](https://github.com/NibiruChain/nibiru/pull/1254) - feat: add bias field into vpool
* [#1255](https://github.com/NibiruChain/nibiru/pull/1255) - feat: add peg multiplier field into vpool, which for now defaults to 1
* [#1281](https://github.com/NibiruChain/nibiru/pull/1281) - feat: add peg multiplier to the pricing logic
* [#1291](https://github.com/NibiruChain/nibiru/pull/1291) - refactor(perp)!: add perp v2 state protos
* [#1296](https://github.com/NibiruChain/nibiru/pull/1296) - refactor(perp)!: update perp v2 state protos
* [#1298](https://github.com/NibiruChain/nibiru/pull/1298) - refactor(perp)!: remove `MaxOracleSpreadRatio` from Perpv2
* [#1302](https://github.com/NibiruChain/nibiru/pull/1302) - refactor(oracle)!: price snapshot start time inclusive
* [#1301](https://github.com/NibiruChain/nibiru/pull/1301) - fix(epochs)!: correct epoch start time
* [#1304](https://github.com/NibiruChain/nibiru/pull/1304) - feat: db backend - rocksdb
* [#1305](https://github.com/NibiruChain/nibiru/pull/1305) - refactor(perp!): Remove unnecessary protos
* [#1312](https://github.com/NibiruChain/nibiru/pull/1312) - feat(wasm): wire depth shift handler to the wasm router
* [#1306](https://github.com/NibiruChain/nibiru/pull/1306) - feat(perp): complete perp v2 types
* [#1309](https://github.com/NibiruChain/nibiru/pull/1309) - feat: minimum swap amount set to $1
* [#1336](https://github.com/NibiruChain/nibiru/pull/1336) - feat: move oracle params out of params subspace and onto the keeper
* [#1315](https://github.com/NibiruChain/nibiru/pull/1315) - feat: oracle rewards distribution every week
* [#1342](https://github.com/NibiruChain/nibiru/pull/1342) - feat(perp): market not enabled can only be used to close out existing positions
* [#1367](https://github.com/NibiruChain/nibiru/pull/1367) - feat: wire enable market to wasm
* [#1382](https://github.com/NibiruChain/nibiru/pull/1382) - refactor(perp)!: remove `perpv1`
* [#1385](https://github.com/NibiruChain/nibiru/pull/1385) - test(perp): add clearing house negative tests
* [#1388](https://github.com/NibiruChain/nibiru/pull/1388) - refactor(perp)!: idempotent position changed event
* [#1387](https://github.com/NibiruChain/nibiru/pull/1387) - feat: upgrade to Cosmos SDK v0.46.10
* [#1413](https://github.com/NibiruChain/nibiru/pull/1413) - fix(perp): provide descriptive errors when all liquidations fail in MultiLiquidate
* [#1427](https://github.com/NibiruChain/nibiru/pull/1427) - refactor(perp)!: PositionChangedEvent `MarginToUser`
* [#1407](https://github.com/NibiruChain/nibiru/pull/1407) - feat!: upgrade to Cosmos SDK v0.47.3

### Dependencies

- Bump `robinraju/release-downloader` from 1.6 to 1.8 (#1326)
- Bump `pozetroninc/github-action-get-latest-release` from 0.6.0 to 0.7.0 (#1325)
- Bump `technote-space/get-diff-action` from 4 to 6 (#1327)
- Bump `actions/setup-go` from 3 to 4 (#1324)
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

## [v0.19.2](https://github.com/NibiruChain/nibiru/releases/tag/v0.19.2) - 2023-02-24

### Features

* [#1187](https://github.com/NibiruChain/nibiru/pull/1187) - feat(oracle): default vote threshold and min voters
* [#1276](https://github.com/NibiruChain/nibiru/pull/1276) - feat: add ewma function
* [#1284](https://github.com/NibiruChain/nibiru/pull/1284) - feat: fails if base and quote reserves are not equal on CreatePool
* [#1286](https://github.com/NibiruChain/nibiru/pull/1286) - feat: bias is zero when creating pool

### API Breaking

* [#1196](https://github.com/NibiruChain/nibiru/pull/1196) - refactor(spot)!: default whitelisted asset and query cli
* [#1195](https://github.com/NibiruChain/nibiru/pull/1195) - feat(perp)!: Add `MultiLiquidation` feature for perps
* [#1158](https://github.com/NibiruChain/nibiru/pull/1158) - feat(asset-registry)!: Add `AssetRegistry`
* [#1171](https://github.com/NibiruChain/nibiru/pull/1171) - refactor(asset)!: Replace `common.AssetPair` with `asset.Pair`.
* [#1164](https://github.com/NibiruChain/nibiru/pull/1164) - refactor: remove client interface for liquidate msg
* [#1173](https://github.com/NibiruChain/nibiru/pull/1173) - refactor(spot)!: replace `x/dex` module with `x/spot`.
* [#1176](https://github.com/NibiruChain/nibiru/pull/1176) - refactor(spot)!: replace `x/dex` module with `x/spot`.

### State Machine Breaking

* [#xxx](https://github.com/NibiruChain/nibiru/pull/xxx) - fix(wasm)!: call `ValidateBasic` before all `sdk.Msg` calls for the bindings-perp contract + remove sudo permissioning
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
* [#1283](https://github.com/NibiruChain/nibiru/pull/1283) - chore(deps): bump github.com/prometheus/client_golang from 1.14.0 to 1.15.0

### Bug Fixes

* [#1194](https://github.com/NibiruChain/nibiru/pull/1194) - fix(oracle): local min voters
* [#1126](https://github.com/NibiruChain/nibiru/pull/1126) - test(oracle): stop the tyrannical behavior of TestFuzz_PickReferencePair
* [#1131](https://github.com/NibiruChain/nibiru/pull/1131) - fix(oracle): use correct distribution module account
* [#1151](https://github.com/NibiruChain/nibiru/pull/1151) - fix(dex): fix swap calculation for stableswap pools
* [#1210](https://github.com/NibiruChain/nibiru/pull/1210) - fix(ci): fix docker push workflow
* [#1212](https://github.com/NibiruChain/nibiru/pull/1212) - fix(spot): gracefully handle join spot pool with wrong tokens denom
* [#1219](https://github.com/NibiruChain/nibiru/pull/1219) - fix(ci): use chaosnet image on chaosnet docker compose
* [#1414](https://github.com/NibiruChain/nibiru/pull/1414) - fix(oracle): Add deterministic map iterations to avoid consensus failure.

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