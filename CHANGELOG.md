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
"Client Breaking" for breaking CLI commands and REST routes used by end-users.
"API Breaking" for breaking exported APIs used by developers building on SDK.
"State Machine Breaking" for any changes that result in a different AppState 
given same genesisState and txList.
Ref: https://keepachangelog.com/en/1.0.0/
-->

# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## Unreleased

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
