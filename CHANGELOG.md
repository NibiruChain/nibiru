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
<!-- 
NOTE: The brackets around the word "Unreleased" are required to pass the [CI test
that checks if we updated the changelog. This is a convention from the [keep a
changelog format](https://keepachangelog.com/en/1.0.0/).  
See https://github.com/dangoslen/changelog-enforcer.
-->

- [#2353](https://github.com/NibiruChain/nibiru/pull/2353) - refactor(oracle): remove dead code from asset registry 
- [#2371](https://github.com/NibiruChain/nibiru/pull/2371) - feat(evm): fix UnmarshalJSON to accept ASCII hex strings
- [#2372](https://github.com/NibiruChain/nibiru/pull/2372) - feat(tokenfactory-cli): add CLI commands for set denom functions

### Dependencies
- Bump `base-x` from 3.0.10 to 3.0.11 ([#2355](https://github.com/NibiruChain/nibiru/pull/2355))
- Bump `pbkdf2` from 3.1.2 to 3.1.3 ([#2356](https://github.com/NibiruChain/nibiru/pull/2356))

## [v2.6.0](https://github.com/NibiruChain/nibiru/releases/tag/v2.6.0) - 2025-08-05

- [#2331](https://github.com/NibiruChain/nibiru/pull/2331) - test(evm-e2e): WNIBI tests for deposit, transfer and total supply
- [#2334](https://gittub.com/NibiruChain/nibiru/pull/2334) - feat(evm-embeds): Publish new version for `@nibiruchain/solidity@0.0.6`, which updates `NibiruOracleChainLinkLike.sol` to have additional methods used by Aave.
- [#2340](https://github.com/NibiruChain/nibiru/pull/2340) - fix: evm indexer proper parsing of the start block
- [#2344](https://gittub.com/NibiruChain/nibiru/pull/23344) - feat(evm): Add some evm messages into the evm codec.
- [#2346](https://gittub.com/NibiruChain/nibiru/pull/2346) - fix(buf-gen-rs): improve Rust proto binding generation script robustness and get it to work with a forked Cosmos-SDK dependency and exit correctly on failure
- [#2348](https://github.com/NibiruChain/nibiru/pull/2348) - fix(oracle): max expiration a label rather than an invalidation for additional query liveness
- [#2350](https://github.com/NibiruChain/nibiru/pull/2350) - fix(simapp): sim tests with empty validator set panic
- [#2352](https://github.com/NibiruChain/nibiru/pull/2352) - chore(token-registry): Add bank coin versions of USDC and USDT from Stargate and LayerZero, and update ErisEvm.sol to fix redeem
- [#2354](https://github.com/NibiruChain/nibiru/pull/2354) - chore: linter upgrade to v2
- [#2357](https://github.com/NibiruChain/nibiru/pull/2357) - fix: proper statedb isolation in nibiru bank_extension

### Dependencies
- Bump `form-data` from 4.0.1 to 4.0.4 ([#2347](https://github.com/NibiruChain/nibiru/pull/2347))
- Bump `golang.org/x/oauth2` from 0.16.0 to 0.27.0 ([#2342](https://github.com/NibiruChain/nibiru/pull/2342))
- Bump `undici` from 5.28.5 to 5.29.0 ([#2310](https://github.com/NibiruChain/nibiru/pull/2310))
- Bump `base-x` from 3.0.10 to 3.0.11 ([#2307](https://github.com/NibiruChain/nibiru/pull/2307))

## [v2.5.0](https://github.com/NibiruChain/nibiru/releases/tag/v2.5.0) - 2025-06-09

- [#2311](https://github.com/NibiruChain/nibiru/pull/2311) - refactor: use Go's built-in min and max functions to simplify logic
- [#2314](https://github.com/NibiruChain/nibiru/pull/2314) - refactor(upgrades): add public keepers to upgrade handlers + DRY improvements
- [#2315](https://github.com/NibiruChain/nibiru/pull/2315) - feat(upgrades): implement v2.5.0 upgrade handler that modifies the stNIBI ERC20 and Bank Coin metadata in place
- [#2316](https://github.com/NibiruChain/nibiru/pull/2316) - feat(ux): add GET behavior to the Ethereum JSON-RPC endpoints for Nibiru so they return info instead of a blank page or error.
- [#2324](https://github.com/NibiruChain/nibiru/pull/2324) - fix(evm): adjust the v2.5.0 upgrade handler to maintain the original stNIBI ERC20 contract's state.
- [#2327](https://github.com/NibiruChain/nibiru/pull/2327) - fix(eth): implement unmarshal json for TransactionReceipt
- [#2329](https://github.com/NibiruChain/nibiru/pull/2329) - fix(eth): use evm RPC in tx_info_test
- [#2328](https://github.com/NibiruChain/nibiru/pull/2328) - fix(evm): ensure StateDB doesn't persist between EVM calls

## [v2.4.0](https://github.com/NibiruChain/nibiru/releases/tag/v2.4.0) - 2025-05-29

- [#2274](https://github.com/NibiruChain/nibiru/pull/2274) - feat(evm)!: update to geth v1.13 with EIP-1153, PRECOMPILE_ADDRS, and transient storage support
- [#2275](https://github.com/NibiruChain/nibiru/pull/2275) - feat(evm)!: update to geth v1.14 with tracing updates and new StateDB methods.
  - This upgrade keeps Nibiru's EVM on the Berlin upgrade to avoid incompatibilities stemming from functionality specific to Ethereum's consensus setup. Namely, blobs (Cancun) and Verkle additions for zkEVM.
  - The jump to v1.14 was necessary to use an up-to-date "cockroach/pebble" DB dependency and leverage new generics features added in Go 1.23+.
- [#2289](https://github.com/NibiruChain/nibiru/pull/2289) - fix(eth-rpc): error propagation fixes and tests for the methods exposed by Nibiru's EVM JSON-RPC
- [#2290](https://github.com/NibiruChain/nibiru/pull/2290) - refactor: use importas linter for consistent imports
- [#2296](https://github.com/NibiruChain/nibiru/pull/2296) - chore(ci): use shell script for generating changelog in releases
- [#2297](https://github.com/NibiruChain/nibiru/pull/2297) - fix(evm): fix error handling for revert errors
- [#2298](https://github.com/NibiruChain/nibiru/pull/2298) - fix(eth-rpc): clean up error propagation and descriptions in eth namespace
- [#2300](https://github.com/NibiruChain/nibiru/pull/2300) - refactor(eth-rpc): combine rpc/backend and rpc/rpcapi since they essentially one package
- [#2301](https://github.com/NibiruChain/nibiru/pull/2301) - fix(.github): glob patterns broken in nibiru-go filter for dorny/paths-filter
- [#2306](https://github.com/NibiruChain/nibiru/pull/2306) - feat(evm): add v2.4.0 upgrade handler
- [#2303](https://github.com/NibiruChain/nibiru/pull/2303) - test(eth/rpc/rpcapi): increase coverage of the rpcapi package using JSON-RPC calls

### Dependencies

- Bump `golang.org/x/net` from 0.37.0 to 0.39.0. ([#2284](https://github.com/NibiruChain/nibiru/pull/2284))
- Bump `github.com/golang-jwt/jwt/v4` from 4.5.1 to 4.5.2 ([#2294](https://github.com/NibiruChain/nibiru/pull/2294))

## [v2.3.0](https://github.com/NibiruChain/nibiru/releases/tag/v2.3.0) - 2025-04-22

- [#2242](https://github.com/NibiruChain/nibiru/pull/2242) - feat(tokenfactory): tx msg SudoSetDenomMetadata
- [#2244](https://github.com/NibiruChain/nibiru/pull/2244) - refactor(test): update how tests are wired with `NewNibiruTestApp`
- [#2250](https://github.com/NibiruChain/nibiru/pull/2250) - refactor(ci): separate builds by platform and without goreleaser
- [#2251](https://github.com/NibiruChain/nibiru/pull/2251) - feat(evm): add ERC20 contract with metadata updates
- [#2249](https://github.com/NibiruChain/nibiru/pull/2249) - fix(evm): resetting gas meter for afterOp in bank extension
- [#2257](https://github.com/NibiruChain/nibiru/pull/2257) - fix: simulation tests by register interfaces for vesting and use correct app keys field
- [#2260](https://github.com/NibiruChain/nibiru/pull/2260) - feat(evm): add getErc20Address method to IFunToken
- [#2268](https://github.com/NibiruChain/nibiru/pull/2268) - fix(evm): gas limit for erc20 deploy
- [#2240](https://github.com/NibiruChain/nibiru/pull/2240) - feat: add depinject wiring and wire `x/auth` module
- [#2246](https://github.com/NibiruChain/nibiru/pull/2246) - feat: add depinject wiring for x/bank module
- [#2248](https://github.com/NibiruChain/nibiru/pull/2248) - feat: add depinject wiring for x/staking module
- [#2253](https://github.com/NibiruChain/nibiru/pull/2253) - feat: add depinject wiring for x/distribution module
- [#2254](https://github.com/NibiruChain/nibiru/pull/2254) - feat: add depinject wiring for x/crisis module
- [#2259](https://github.com/NibiruChain/nibiru/pull/2259) - feat: add depinject wiring for all sdk modules
- [#2261](https://github.com/NibiruChain/nibiru/pull/2261) - feat: add depinject wiring for x/sudo module
- [#2262](https://github.com/NibiruChain/nibiru/pull/2262) - feat: add depinject wiring for x/oracle module
- [#2263](https://github.com/NibiruChain/nibiru/pull/2263) - feat: add depinject wiring for x/epochs module
- [#2265](https://github.com/NibiruChain/nibiru/pull/2265) - feat: add depinject wiring for x/inflation module
- [#2266](https://github.com/NibiruChain/nibiru/pull/2266) - feat: add depinject wiring for x/evm module
- [#2272](https://github.com/NibiruChain/nibiru/pull/2272) - feat: add depinject wiring for x/tokenfactory module
- [#2271](https://github.com/NibiruChain/nibiru/pull/2271) - fix(ci): update tag-pattern for changelog step in releases
- [#2270](https://github.com/NibiruChain/nibiru/pull/2270) - refactor(app): remove private keeper struct and transient/mem keys from app
- [#2288](https://github.com/NibiruChain/nibiru/pull/2288) - chore(ci): add workflow to check for missing upgrade handler
- [#2278](https://github.com/NibiruChain/nibiru/pull/2278) - chore: migrate to cosmossdk.io/mathLegacyDec and cosmossdk.io/math.Int
- [#2293](https://github.com/NibiruChain/nibiru/pull/2293) - ci(release): pack nibid binary with no enclosing directory
- [#2292](https://github.com/NibiruChain/nibiru/pull/2292) - fix: use tmp directory for pre-instantiating app

## [v2.2.0](https://github.com/NibiruChain/nibiru/releases/tag/v2.2.0) - 2025-03-27

- [#2222](https://github.com/NibiruChain/nibiru/pull/2222) - fix(evm): evm indexer proper stopping of the indexer service
- [#2224](https://github.com/NibiruChain/nibiru/pull/2224) - fix(evm): suppressing error on missing block bloom event
- [#2238](https://github.com/NibiruChain/nibiru/pull/2238) - feat(evm-embeds): Add WNIBI.sol implementatino to contracts and related TypeScript and Solidity package updates for npm.
- [#2239](https://github.com/NibiruChain/nibiru/pull/2239) - feat(funtoken): update `FunToken.sendToBank` to accept EVM and nibi addresses.
- [#2241](https://github.com/NibiruChain/nibiru/pull/2241) - fix(evm): evm-tx-index cli fix to exclude most latest block
- [#2236](https://github.com/NibiruChain/nibiru/pull/2236) - chore: make function comment match function name and fix linter errors
- [#2243](https://github.com/NibiruChain/nibiru/pull/2243) - fix(deps): bump Go to v1.24, similar to [#1698](https://github.com/NibiruChain/nibiru/pull/1698)
- [#2250](https://github.com/NibiruChain/nibiru/pull/2250) - fix(upgrades): add missing 2.2.0 upgrade handler

### Dependencies

- Bump `axios` from 1.7.4 to 1.8.2 ([#2230](https://github.com/NibiruChain/nibiru/pull/2230))
- Bump `golang.org/x/net` from 0.33.0 to 0.37.0 ([#2233](https://github.com/NibiruChain/nibiru/pull/2233))
- chore: update golangci-lint version to v1.64.8 ([#2233](https://github.com/NibiruChain/nibiru/pull/2233))
- Bump `[golang.org/x/net](https://github.com/golang/net)` from 0.33.0 to 0.37.0. ([#2233](https://github.com/NibiruChain/nibiru/pull/2233))
- Bump `github.com/golang/glog` from 1.2.0 to 1.2.4 ([#2182](https://github.com/NibiruChain/nibiru/pull/2182))

## [v2.1.0](https://github.com/NibiruChain/nibiru/releases/tag/v2.1.0) - 2025-02-25

- [#2104](https://github.com/NibiruChain/nibiru/pull/2104) - chore: update chain IDs
- [#2202](https://github.com/NibiruChain/nibiru/pull/2202) - chore(build): add build tags and missing flags/variables
- [#2206](https://github.com/NibiruChain/nibiru/pull/2206) - ci(chaosnet): fix docker image build
- [#2207](https://github.com/NibiruChain/nibiru/pull/2207) - chore(ci): add cache for chaosnet builds
- [#2209](https://github.com/NibiruChain/nibiru/pull/2209) - refator(ci):
  Simplify GitHub actions based on conditional paths, removing the need for files like ".github/workflows/skip-unit-tests.yml".
- [#2211](https://github.com/NibiruChain/nibiru/pull/2211) - ci(chaosnet): avoid building on cache injected directories
- [#2212](https://github.com/NibiruChain/nibiru/pull/2212) - fix(evm): proper eth tx logs emission for funtoken operations
- [#2213](https://github.com/NibiruChain/nibiru/pull/2213) - chore(build): include lib versions on cache
- [#2214](https://github.com/NibiruChain/nibiru/pull/2214) - chore(wasm): bump wasmvm to `v1.5.8`
- [#2068](https://github.com/NibiruChain/nibiru/pull/2068) - feat: enable wasm light clients on IBC (08-wasm)
- [#2217](https://github.com/NibiruChain/nibiru/pull/2217) - fix: app-db-backend not recognized on prune command
- [#2219](https://github.com/NibiruChain/nibiru/pull/2219) - fix(evm): disable unprotected tx check in EVM ante handler
- [#2220](https://github.com/NibiruChain/nibiru/pull/2220) - fix(evm): improved marshaling of the eth tx receipt

## [v2.0.0-p1](https://github.com/NibiruChain/nibiru/releases/tag/v2.0.0-p1) - 2025-02-10

- fbcca386 fix: revert wasmvm to v1.5.0
- 533490d0 fix: revert testnet-1 chain id to 7210
- d8a10921 chore: update changelog for v2 EVM release

## [v2.0.0](https://github.com/NibiruChain/nibiru/releases/tag/v2.0.0) - 2025-02-10

- [#2119](https://github.com/NibiruChain/nibiru/pull/2119) - fix(evm): Guarantee
  that gas consumed during any send operation of the "NibiruBankKeeper" depends
  only on the "bankkeeper.BaseKeeper"'s gas consumption.
- [#2120](https://github.com/NibiruChain/nibiru/pull/2120) - fix: Use canonical hexadecimal strings for Eip155 address encoding
- [#2122](https://github.com/NibiruChain/nibiru/pull/2122) - test(evm): more bank extension tests and EVM ABCI integration tests to prevent regressions
- [#2124](https://github.com/NibiruChain/nibiru/pull/2124) - refactor(evm):
  Remove unnecessary argument in the `VerifyFee` function, which returns the token
  payment required based on the effective fee from the tx data. Improve
  documentation.
- [#2125](https://github.com/NibiruChain/nibiru/pull/2125) - feat(evm-precompile):Emit EVM events created to reflect the ABCI events that occur outside the EVM to make sure that block explorers and indexers can find indexed ABCI event information.
- [#2127](https://github.com/NibiruChain/nibiru/pull/2127) - fix(vesting): disabled built in auth/vesting module functionality
- [#2129](https://github.com/NibiruChain/nibiru/pull/2129) - fix(evm): issue with infinite recursion in erc20 funtoken contracts
- [#2130](https://github.com/NibiruChain/nibiru/pull/2130) - fix(evm): proper nonce management in statedb
- [#2132](https://github.com/NibiruChain/nibiru/pull/2132) - fix(evm): proper tx gas refund
- [#2134](https://github.com/NibiruChain/nibiru/pull/2134) - fix(evm): query of NIBI should use bank state, not the StateDB
- [#2139](https://github.com/NibiruChain/nibiru/pull/2139) - fix(evm): erc20 born funtoken: properly burn bank coins after converting coin back to erc20
- [#2140](https://github.com/NibiruChain/nibiru/pull/2140) - fix(bank): bank keeper extension now charges gas for the bank operations
- [#2141](https://github.com/NibiruChain/nibiru/pull/2141) - refactor: simplify account retrieval operation in `nibid q evm account`.
- [#2142](https://github.com/NibiruChain/nibiru/pull/2142) - fix(bank): add additional missing methods to the NibiruBankKeeper
- [#2144](https://github.com/NibiruChain/nibiru/pull/2144) - feat(token-registry): Implement strongly typed Nibiru Token Registry and generation command
- [#2145](https://github.com/NibiruChain/nibiru/pull/2145) - chore(token-registry): add xNIBI Astrovault LST to registry
- [#2147](https://github.com/NibiruChain/nibiru/pull/2147) - fix(simapp): manually add x/vesting Cosmos-SDK module types to the codec in simulation tests since they are expected by default
- [#2149](https://github.com/NibiruChain/nibiru/pull/2149) - feat(evm-oracle):
  add Solidity contract that we can use to expose the Nibiru Oracle in the
  ChainLink interface. Publish all precompiled contracts and ABIs on npm under
  the `@nibiruchain/solidity` package.
- [#2151](https://github.com/NibiruChain/nibiru/pull/2151) - feat(evm): randao support for evm
- [#2152](https://github.com/NibiruChain/nibiru/pull/2152) - fix(precompile): consume gas for precompile calls regardless of error
- [#2154](https://github.com/NibiruChain/nibiru/pull/2154) - fix(evm):
  JSON encoding for the `EIP55Addr` struct was not following the Go conventions and
  needed to include double quotes around the hexadecimal string.
- [#2156](https://github.com/NibiruChain/nibiru/pull/2156) - test(evm-e2e): add E2E test using the Nibiru Oracle's ChainLink impl
- [#2157](https://github.com/NibiruChain/nibiru/pull/2157) - fix(evm): Fix unit inconsistency related to AuthInfo.Fee and txData.Fee using effective fee
- [#2159](https://github.com/NibiruChain/nibiru/pull/2159) - chore(evm): Augment the Wasm msg handler so that wasm contracts cannot send MsgEthereumTx
- [#2160](https://github.com/NibiruChain/nibiru/pull/2160) - fix(evm-precompile): use bank.MsgServer Send in precompile IFunToken.bankMsgSend
- [#2161](https://github.com/NibiruChain/nibiru/pull/2161) - fix(evm): added tx logs events to the funtoken related txs
- [#2162](https://github.com/NibiruChain/nibiru/pull/2162) - test(testutil): try retrying for 'panic: pebbledb: closed'
- [#2167](https://github.com/NibiruChain/nibiru/pull/2167) - refactor(evm): removed blockGasUsed transient variable
- [#2168](https://github.com/NibiruChain/nibiru/pull/2168) - chore(evm-solidity): Move unrelated docs, gen-embeds, and add Solidity docs
- [#2165](https://github.com/NibiruChain/nibiru/pull/2165) - fix(evm): use Singleton StateDB pattern for EVM txs
- [#2169](https://github.com/NibiruChain/nibiru/pull/2169) - fix(evm): Better handling erc20 metadata
- [#2170](https://github.com/NibiruChain/nibiru/pull/2170) - chore: Remove redundant allowUnprotectedTxs
- [#2172](https://github.com/NibiruChain/nibiru/pull/2172) - chore: close iterator in IterateEpochInfo
- [#2173](https://github.com/NibiruChain/nibiru/pull/2173) - fix(evm): clear `StateDB` between calls
- [#2177](https://github.com/NibiruChain/nibiru/pull/2177) - fix(cmd): Continue from #2127 and unwire vesting flags and logic from genaccounts.go
- [#2176](https://github.com/NibiruChain/nibiru/pull/2176) - tests(evm): add dirty state tests from code4rena audit
- [#2180](https://github.com/NibiruChain/nibiru/pull/2180) - fix(evm): apply gas consumption across the entire EVM codebase at `CallContractWithInput`
- [#2183](https://github.com/NibiruChain/nibiru/pull/2183) - fix(evm): bank keeper extension gas meter type
- [#2184](https://github.com/NibiruChain/nibiru/pull/2184) - test(evm): e2e tests configuration enhancements
- [#2187](https://github.com/NibiruChain/nibiru/pull/2187) - fix(evm): fix eip55 address encoding
- [#2188](https://github.com/NibiruChain/nibiru/pull/2188) - refactor(evm): update logs emission
- [#2192](https://github.com/NibiruChain/nibiru/pull/2192) - fix(oracle): correctly handle misscount
- [#2197](https://github.com/NibiruChain/nibiru/pull/2197) - chore(evm): Create ethers v6 adapters for Nibiru. Publish as a library on npm (`@nibiruchain/evm-core`)
- [#2200](https://github.com/NibiruChain/nibiru/pull/2200) - fix(test): evm e2e oracle test fixed pair name

#### Nibiru EVM | Before Audit 2 - 2024-12-06

The codebase went through a third-party [Code4rena
Zenith](https://code4rena.com/zenith) Audit, running from 2024-10-07 until
2024-11-01 and including both a primary review period and mitigation/remission
period. This section describes code changes that occurred after that audit in
preparation for a second audit starting in November 2024.

- [#2074](https://github.com/NibiruChain/nibiru/pull/2074) - fix(evm-keeper): better utilize ERC20 metadata during FunToken creation. The bank metadata for a new FunToken mapping ties a connection between the Bank Coin's `DenomUnit` and the ERC20 contract metadata like the name, decimals, and symbol. This change brings parity between EVM wallets, such as MetaMask, and Interchain wallets like Keplr and Leap.
- [#2076](https://github.com/NibiruChain/nibiru/pull/2076) - fix(evm-gas-fees):
  Use effective gas price in RefundGas and make sure that units are properly
  reflected on all occurrences of "base fee" in the codebase. This fixes [#2059](https://github.com/NibiruChain/nibiru/issues/2059)
  and the [related comments from @Unique-Divine and @berndartmueller](https://github.com/NibiruChain/nibiru/issues/2059#issuecomment-2408625724).
- [#2084](https://github.com/NibiruChain/nibiru/pull/2084) - feat(evm-forge): foundry support and template for Nibiru EVM development
- [#2086](https://github.com/NibiruChain/nibiru/pull/2086) - fix(evm-precomples):
  Fix state consistency in precompile execution by ensuring proper journaling of
  state changes in the StateDB. This pull request makes sure that state is
  committed as expected, fixes the `StateDB.Commit` to follow its guidelines more
  closely, and solves for a critical state inconsistency producible from the
  FunToken.sol precompiled contract. It also aligns the precompiles to use
  consistent setup and dynamic gas calculations, addressing the following tickets.
  - <https://github.com/NibiruChain/nibiru/issues/2083>
  - <https://github.com/code-423n4/2024-10-nibiru-zenith/issues/43>
  - <https://github.com/code-423n4/2024-10-nibiru-zenith/issues/47>
- [#2088](https://github.com/NibiruChain/nibiru/pull/2088) - refactor(evm): remove outdated comment and improper error message text
- [#2089](https://github.com/NibiruChain/nibiru/pull/2089) - better handling of gas consumption within erc20 contract execution
- [#2090](https://github.com/NibiruChain/nibiru/pull/2090) - fix(evm): Account
  for (1) ERC20 transfers with tokens that return false success values instead of
  throwing an error and (2) ERC20 transfers with other operations that don't bring
  about the expected resulting balance for the transfer recipient.
- [#2091](https://github.com/NibiruChain/nibiru/pull/2091) - feat(evm): add fun token creation fee validation
- [#2093](https://github.com/NibiruChain/nibiru/pull/2093) - feat(evm): gas usage in precompiles: limits, local gas meters
- [#2092](https://github.com/NibiruChain/nibiru/pull/2092) - feat(evm): add validation for wasm multi message execution
- [#2094](https://github.com/NibiruChain/nibiru/pull/2094) - fix(evm): Following
  from the changs in #2086, this pull request implements a new `JournalChange`
  struct that saves a deep copy of the state multi store before each
  state-modifying, Nibiru-specific precompiled contract is called (`OnRunStart`).
  Additionally, we commit the `StateDB` there as well. This guarantees that the
  non-EVM and EVM state will be in sync even if there are complex, multi-step
  Ethereum transactions, such as in the case of an EthereumTx that influences the
  `StateDB`, then calls a precompile that also changes non-EVM state, and then EVM
  reverts inside of a try-catch.
- [#2095](https://github.com/NibiruChain/nibiru/pull/2095) - fix(evm): This
  change records NIBI (ether) transfers on the `StateDB` during precompiled
  contract calls using the `NibiruBankKeeper`, which is struct extension of
  the `bankkeeper.BaseKeeper` that is used throughout Nibiru.
  The `NibiruBankKeeper` holds a reference to the current EVM `StateDB` and records
  balance changes in wei as journal changes automatically. This guarantees that
  commits and reversions of the `StateDB` do not misalign with the state of the
  Bank module. This code change uses the `NibiruBankKeeper` on all modules that
  depend on x/bank, such as the EVM and Wasm modules.
- [#2097](https://github.com/NibiruChain/nibiru/pull/2097) - feat(evm): Add new query to get dated price from the oracle precompile
- [#2100](https://github.com/NibiruChain/nibiru/pull/2100) - refactor: cleanup statedb and precompile sections
- [#2098](https://github.com/NibiruChain/nibiru/pull/2098) - test(evm): statedb tests for race conditions within funtoken precompile
- [#2101](https://github.com/NibiruChain/nibiru/pull/2101) - fix(evm): tx receipt proper marshalling
- [#2105](https://github.com/NibiruChain/nibiru/pull/2105) - test(evm): precompile call with revert
- [#2106](https://github.com/NibiruChain/nibiru/pull/2106) - chore: scheduled basic e2e tests for evm testnet endpoint
- [#2107](https://github.com/NibiruChain/nibiru/pull/2107) - feat(evm-funtoken-precompile): Implement methods: balance, bankBalance, whoAmI
- [#2108](https://github.com/NibiruChain/nibiru/pull/2108) - fix(evm): removed deprecated root key from eth_getTransactionReceipt
- [#2110](https://github.com/NibiruChain/nibiru/pull/2110) - fix(evm): Restore StateDB to its state prior to ApplyEvmMsg call to ensure deterministic gas usage. This fixes an issue where the StateDB pointer field in NibiruBankKeeper was being updated during readonly query endpoints like eth_estimateGas, leading to non-deterministic gas usage in subsequent transactions.
- [#2111](https://github.com/NibiruChain/nibiru/pull/2111) - fix: e2e-evm-cron.yml
- [#2114](https://github.com/NibiruChain/nibiru/pull/2114) - fix(evm): make gas cost zero in conditional bank keeper flow
- [#2116](https://github.com/NibiruChain/nibiru/pull/2116) - fix(precompile-funtoken.go): Fixes a bug where the err != nil check is missing in the bankBalance precompile method
- [#2117](https://github.com/NibiruChain/nibiru/pull/2117) - fix(oracle): The
  timestamps resulting from ctx.WithBlock\* don't actually correspond to the block
  header information from specified blocks in the chain's history, so the oracle
  exchange rates need a way to correctly retrieve this information. This change
  fixes that discrepancy, giving the expected block timestamp for the EVM's oracle
  precompiled contract. The change also simplifies and corrects the code in x/oracle.

#### Nibiru EVM | Before Audit 1 - 2024-10-18

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
- [#1914](https://github.com/NibiruChain/nibiru/pull/1914) - refactor(evm): Remove dead code and document non-EVM ante handler
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
- [#2006](https://github.com/NibiruChain/nibiru/pull/2006) - test(evm): e2e tests for eth\_\* endpoints
- [#2008](https://github.com/NibiruChain/nibiru/pull/2008) - refactor(evm): clean up precompile setups
- [#2013](https://github.com/NibiruChain/nibiru/pull/2013) - chore(evm): Set appropriate gas value for the required gas of the "IFunToken.sol" precompile.
- [#2014](https://github.com/NibiruChain/nibiru/pull/2014) - feat(evm): Emit block bloom event in EndBlock hook.
- [#2017](https://github.com/NibiruChain/nibiru/pull/2017) - fix(evm): Fix DynamicFeeTx gas cap parameters
- [#2019](https://github.com/NibiruChain/nibiru/pull/2019) - chore(evm): enabled debug rpc api on localnet.
- [#2020](https://github.com/NibiruChain/nibiru/pull/2020) - test(evm): e2e tests for debug namespace
- [#2022](https://github.com/NibiruChain/nibiru/pull/2022) - feat(evm): debug_traceCall method implemented
- [#2023](https://github.com/NibiruChain/nibiru/pull/2023) - fix(evm)!: adjusted generation and parsing of the block bloom events
- [#2030](https://github.com/NibiruChain/nibiru/pull/2030) - refactor(eth/rpc): Delete unused code and improve logging in the eth and debug namespaces
- [#2031](https://github.com/NibiruChain/nibiru/pull/2031) - fix(evm): debug calls with custom tracer and tracer options
- [#2032](https://github.com/NibiruChain/nibiru/pull/2032) - feat(evm): ante handler to prohibit authz grant evm messages
- [#2039](https://github.com/NibiruChain/nibiru/pull/2039) - refactor(rpc-backend): remove unnecessary interface code
- [#2044](https://github.com/NibiruChain/nibiru/pull/2044) - feat(evm): evm tx indexer service implemented
- [#2045](https://github.com/NibiruChain/nibiru/pull/2045) - test(evm): backend tests with test network and real txs
- [#2053](https://github.com/NibiruChain/nibiru/pull/2053) - refactor(evm): converted untyped event to typed and cleaned up
- [#2054](https://github.com/NibiruChain/nibiru/pull/2054) - feat(evm-precompile): Precompile for one-way EVM calls to invoke/execute Wasm contracts.
- [#2060](https://github.com/NibiruChain/nibiru/pull/2060) - fix(evm-precompiles): add assertNumArgs validation
- [#2056](https://github.com/NibiruChain/nibiru/pull/2056) - feat(evm): add oracle precompile
- [#2065](https://github.com/NibiruChain/nibiru/pull/2065) - refactor(evm)!: Refactor out dead code from the evm.Params
- [#2135](https://github.com/NibiruChain/nibiru/pull/2135) - feat(evm): add precompile for calling bank to evm from evm

### State Machine Breaking (Other)

#### For next mainnet version

- [#1766](https://github.com/NibiruChain/nibiru/pull/1766) - refactor(app-wasmext)!: remove wasmbinding `CosmosMsg::Custom` bindings.
- [#1776](https://github.com/NibiruChain/nibiru/pull/1776) - feat(inflation): make inflation params a collection and add commands to update them
- [#1872](https://github.com/NibiruChain/nibiru/pull/1872) - chore(math): use cosmossdk.io/math to replace sdk types
- [#1874](https://github.com/NibiruChain/nibiru/pull/1874) - chore(proto): remove the proto stringer as per Cosmos SDK migration guidelines
- [#1932](https://github.com/NibiruChain/nibiru/pull/1932) - fix(gosdk): fix keyring import functions

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
- [#2050](https://github.com/NibiruChain/nibiru/pull/2050) - refactor(oracle): remove unused code and collapse empty client/cli directory

### Dependencies

- Bump `github.com/grpc-ecosystem/grpc-gateway/v2` from 2.18.1 to 2.19.1 ([#1767](https://github.com/NibiruChain/nibiru/pull/1767), [#1782](https://github.com/NibiruChain/nibiru/pull/1782))
- Bump `robinraju/release-downloader` from 1.8 to 1.11 ([#1783](https://github.com/NibiruChain/nibiru/pull/1783), [#1839](https://github.com/NibiruChain/nibiru/pull/1839), [#1948](https://github.com/NibiruChain/nibiru/pull/1948))
- Bump `github.com/prometheus/client_golang` from 1.17.0 to 1.18.0 ([#1750](https://github.com/NibiruChain/nibiru/pull/1750))
- Bump `golang.org/x/crypto` from 0.15.0 to 0.31.0 ([#1724](https://github.com/NibiruChain/nibiru/pull/1724), [#1843](https://github.com/NibiruChain/nibiru/pull/1843), [#2123](https://github.com/NibiruChain/nibiru/pull/2123))
- Bump `github.com/holiman/uint256` from 1.2.3 to 1.2.4 ([#1730](https://github.com/NibiruChain/nibiru/pull/1730))
- Bump `github.com/dvsekhvalnov/jose2go` from 1.5.0 to 1.6.0 ([#1733](https://github.com/NibiruChain/nibiru/pull/1733))
- Bump `github.com/spf13/cast` from 1.5.1 to 1.6.0 ([#1689](https://github.com/NibiruChain/nibiru/pull/1689))
- Bump `cosmossdk.io/math` from 1.1.2 to 1.4.0 ([#1676](https://github.com/NibiruChain/nibiru/pull/1676), [#2115](https://github.com/NibiruChain/nibiru/pull/2115))
- Bump `github.com/grpc-ecosystem/grpc-gateway/v2` from 2.18.0 to 2.18.1 ([#1675](https://github.com/NibiruChain/nibiru/pull/1675))
- Bump `actions/setup-go` from 4 to 5 ([#1696](https://github.com/NibiruChain/nibiru/pull/1696))
- Bump `golang` from 1.19 to 1.21 ([#1698](https://github.com/NibiruChain/nibiru/pull/1698))
- [#1678](https://github.com/NibiruChain/nibiru/pull/1678) - chore(deps): collections to v0.4.0 for math.Int value encoder
- Bump `golang.org/x/net` from 0.0.0-20220607020251-c690dde0001d to 0.33.0 ([#1849](https://github.com/NibiruChain/nibiru/pull/1849), [#2175](https://github.com/NibiruChain/nibiru/pull/2175))
- Bump `golang.org/x/net` from 0.20.0 to 0.23.0 ([#1850](https://github.com/NibiruChain/nibiru/pull/1850))
- Bump `github.com/supranational/blst` from 0.3.8-0.20220526154634-513d2456b344 to 0.3.11 ([#1851](https://github.com/NibiruChain/nibiru/pull/1851))
- Bump `golangci/golangci-lint-action` from 4 to 6 ([#1854](https://github.com/NibiruChain/nibiru/pull/1854), [#1867](https://github.com/NibiruChain/nibiru/pull/1867))
- Bump `github.com/hashicorp/go-getter` from 1.7.1 to 1.7.5 ([#1858](https://github.com/NibiruChain/nibiru/pull/1858), [#1938](https://github.com/NibiruChain/nibiru/pull/1938))
- Bump `github.com/btcsuite/btcd` from 0.23.3 to 0.24.2 ([#1862](https://github.com/NibiruChain/nibiru/pull/1862), [#2070](https://github.com/NibiruChain/nibiru/pull/2070))
- Bump `pozetroninc/github-action-get-latest-release` from 0.7.0 to 0.8.0 ([#1863](https://github.com/NibiruChain/nibiru/pull/1863))
- Bump `bufbuild/buf-setup-action` from 1.30.1 to 1.47.2 ([#1891](https://github.com/NibiruChain/nibiru/pull/1891), [#1900](https://github.com/NibiruChain/nibiru/pull/1900), [#1923](https://github.com/NibiruChain/nibiru/pull/1923), [#1972](https://github.com/NibiruChain/nibiru/pull/1972), [#1974](https://github.com/NibiruChain/nibiru/pull/1974), [#1988](https://github.com/NibiruChain/nibiru/pull/1988), [#2043](https://github.com/NibiruChain/nibiru/pull/2043), [#2057](https://github.com/NibiruChain/nibiru/pull/2057), [#2062](https://github.com/NibiruChain/nibiru/pull/2062), [#2069](https://github.com/NibiruChain/nibiru/pull/2069), [#2102](https://github.com/NibiruChain/nibiru/pull/2102), [#2113](https://github.com/NibiruChain/nibiru/pull/2113))
- Bump `axios` from 1.7.3 to 1.7.4 ([#2016](https://github.com/NibiruChain/nibiru/pull/2016))
- Bump `github.com/CosmWasm/wasmvm` from 1.5.0 to 1.5.5 ([#2047](https://github.com/NibiruChain/nibiru/pull/2047))
- Bump `docker/build-push-action` from 5 to 6 ([#1924](https://github.com/NibiruChain/nibiru/pull/1924))
- Bump `codecov/codecov-action` from 4 to 5 ([#2112](https://github.com/NibiruChain/nibiru/pull/2112))
- Bump `undici` from 5.28.4 to 5.28.5 ([#2174](https://github.com/NibiruChain/nibiru/pull/2174))

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

---

[LEGACY CHANGELOG](./LEGACY-CHANGELOG.md)
