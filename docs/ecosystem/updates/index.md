---
title: Nibiru Advancement Notes (NANs) & Ecosystem Updates
description: >
  Detailed updates on the growth and development of the Nibiru blockchain
  ecosystem. Each one provides insight into onging and upcoming improvements to
  Nibiru.
  NANs are not marketing updates or improvement proposals. They're working notes
  that capture the context, decisions, tradeoffs, and lessons that
  comprise internal discusssions but get entirely lost in
  public-facing communication.
---

# Nibiru Advancement Notes (NANs)

> *[Nibiru Advancement Notes (NANs)](./index.md)*

{{ $frontmatter.description }}

- [2026-09-30 - NAN 008: Defending Wasm in Layers, and Reading EVM State from
  Wasm](./nan-008.md): NAN 008 covers Nibiru v2.19.0 and v2.20.0, two releases
  that protected the network from a Wasm runtime vulnerability while we prepared
  the repair. It also covers Wasm contracts reading EVM state, transaction
  lookup by either hash, and clearer fees on Sai.

- [2026-08-31 - NAN 007: A Month Spent on Speed, and Getting Block Time
  Back](./nan-007.md): NAN 007 covers August 2026, a month we spent on speed.
  Sai got faster to load and to trade, mainnet block time went back to about 1.5
  seconds, and Sai launched Points and referral accounting that follows the
  code.

- [2026-07-31 - NAN 006: Owning Our Dependencies, a Nonce-Aware Mempool, and
  24-Hour Votes](./nan-006.md): NAN 006 covers Nibiru v2.16.0 and v2.17.0. We
  moved the Cosmos SDK, IBC Go, the Wasm module, and the Wasm VM into the Nibiru
  repository, stopped spam from filling blocks with EVM transactions that can't
  execute, cut governance votes to 24 hours, and turned off authz.

- [2026-06-30 - NAN 005: Zero Gas in Browser Wallets, a Self-Governing Treasury,
  and the Return of 0x0801](./nan-005.md): NAN 005 covers Nibiru v2.14.0 and
  v2.15.0. New users can make gasless calls from ordinary browser wallets, the
  Foundation Treasury now manages its own signers, and the old oracle address
  0x0801 returns as a read-only price source so MIM OFT bridge fee quotes work
  again.

- [2026-04-30 - NAN 004: Retiring the Native Oracle with v2.12.0](./nan-004.md):
  NAN 004 covers Nibiru v2.12.0. Validators no longer vote on prices, because
  the chain's built-in oracle is retired. Mapping a token between Nibiru's
  Cosmos and EVM sides now needs approval on mainnet, and indexers no longer see
  events from failed EVM transactions. On Sai, stocks, ETFs, and commodities
  arrived in one listing wave.

- [2026-02-11 - NAN 003: A Smoother EVM for Sai, Passkey Signing, and Zero Gas with v2.11.0](./nan-003.md):
  This release makes Nibiru feel simpler at the surface and safer underneath. It
  fixes a confusing EVM reporting issue, expands gasless onboarding for approved
  contracts, ships building blocks for passkey-secured smart accounts, and includes
  a high-priority consensus security patch.

- [2025-09-28 - NAN 002: Liquidity Surges, New Omnichain Fungible Tokens, and Sai's Path to Mainnet](./nan-002.md): 
  Highlights major liquidity inflows on Nibiru, the launch of several
  new omnichain fungible tokens (OFTs), and progress on the Sai Perps app.

- [2025-09-19 - NAN 001: Genesis of NANs, Nibiru v2.7.0, and BTC on
Nibiru](./nan-001.md): NAN 001 introduces Nibiru Advancement Notes (NANs), a new
series offering a candid look at the tradeoffs, decisions, and progress shaping
Nibiru's evolution. NAN 001 covers Nibiru v2.7.0, cbBTC on Nibiru, and changes to
Sai.

- [Nibiru Ecosystem Update #3 - May 2025](./nibiru-eco-update-03.md): Nibiru V2
is live with its first wave of ecosystem app deployments. This update further
details Nibiru Lagrange Point, a roadmap for long-term scalability (including
Nibiru BFT and Nibiru Adaptive execution), and highlights progress across
real-world assets, structured products, and programs designed to drive durable
adoption.

- [Nibiru Ecosystem Update #2 - Jan 2025](./nibiru-eco-update-02.md):  Since the
last update, the Nibiru V2 (Nibiru EVM Upgrade) has reached feature completion,
passing two successful audits. Nibiru is set to showcase its MultiVM
architecture, which integrates both the Ethereum and WebAssembly virtual machines
in a unified environment. This update also introduces "Nibiru Lagrange Point," a
strategic roadmap extension designed to drive scalability, decentralization, and
innovation over the long term.

- [Nibiru Ecosystem Update #1 - Sep 2024](./nibiru-eco-update-01.md): Throughout
this time, Nibiru reached its V1 mainnet network launch and entered a new chapter
centered on shepherding quality teams to build, deploy on, and make Nibiru their
home.
