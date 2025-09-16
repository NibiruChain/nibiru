---
metaTitle: "Tokenomics | Nibiru (NIBI) Token Unlocks and Vesting"
---
# 💹 Tokenomics

This section goes over the tokenomics and release schedule of [NIBI](./nibi.md), the staking
and utility token of Nibiru Chain. This token plays a pivotal role in powering
the network’s Proof-of-Stake consensus, decentralized governance, and [payment for
computation](../concepts/gas.md) on the network. {synopsis}

<!-- {{toc}} -->

#### Table of Contents

- [Token Distribution: Overview](#token-distribution-overview)
- [Update History](#update-history)
- [Community-Centric Token Distribution](#community-centric-token-distribution)
- [Core Token Utility (NIBI)](#core-token-utility-nibi)
- [Common Questions](#common-questions)
- [Disclaimer](#disclaimer)

## Token Distribution: Overview

Nibiru has a fully-diluted token supply of 1.5B tokens. The tokens are
distributed among the following groups:

![](../img/token\_supply\_maturity.png)

| Split (%) | Group | Description | Schedule |
| --------- | ---- | --- | --- |
| 60        | Community               | Stakers, Development grants, hackathons, liquidity provision, partnerships, and general community governance | Non-linear schedule based on a normalized exponential function. |
| 15.3        | Core Contributors / Team                    | Core team, strategic advisors, and future hires. | Subject to employee token options. 0% TGE, vesting linearly. |
| 8.5       | Investors (Seed) | [Seed funding round in 2022](https://nibiru.fi/blog/posts/007-seed.html) co-led by Tribe Capital, Republic Capital, NGC Ventures, and Original Capital. | 0\% TGE. Cliff for 25\% of the allocation on Nov 8, 2024, followed by linear vest for the other 75\% over 36 months. |
| 8.2 | Investors (Post-seed) | For partners and private investors. Excess from this category remains in strategic reserve. | 0% TGE. Mixture of 24-month and 36-month linear unlocks. |
| 8.0 | Public Sale (CoinList) | [Community Sale on CoinList](./faq/coinlist.html). | 10% unlocked at launch. Linear vest for the other 90\% over 12 months.  |

## Update History  
- Nov 12, 2024 (Last Updated): Added labels for specific dates to the "Schedule"
column.
- Feb 26, 2024: Added documentation to explain how the coefficients are
produced for the community distribution on the Nibiru mainnet.
- Jan 28, 2024: Uploaded unlock schedules for all private investor allocations
and provided up-to-date charts.

## Community-Centric Token Distribution

The [NIBI token](./nibi.md) supply is engineered to foster long-term stability and community engagement. This Nibiru token release schedule is takes inspiration from other large Web3 protocols that came before Nibiru (e.g. Aptos, Uniswap, Solana, Binance, Sui). The majority of the NIBI token supply is allocated to the community to be used for groups like stakers, builders, liquidity providers, and ecosystem grants. 

The token supply is distributed over an 8-year time frame with the following release schedule.

![](../img/token\_release\_area.svg)

As more tokens are released into the ecosystem, Nibiru will be governed primarly by community members.

## Core Token Utility (NIBI)

<!-- ## Hello{hide} -->

The staking and utility token of Nibiru is called "NIBI". This token plays a pivotal role in powering the network’s Proof-of-Stake consensus and [decentralized governance](../concepts/gov/), a process by which NIBI stakers can decide on and bring about changes in the network. NIBI is also utilized to secure the network and pay for computation in the form of gas fees, essential for facilitating the creation of new blocks on the chain.

Token holders who stake or delegate their tokens to a validator operator for
purposes of securing the network and achieving consensus may receive staking
rewards.

### Community - Community Pool

- **Usage Incentives**: (Subject to decentralized governance) Likely used to incentivize liquidity provision or dApp usage on Nibiru Chain.
- **Builder Grants**: See [Nibiru Grants](https://nibiru.fi/ecosystem/grants).

### Community - Strategic Reserve

The treasury is constructed as a discretionary fund to ensure the stability of the protocol. The treasury will initially be managed by multi-sig wallets held by members of the core team or their related smart contracts.

### Implementation Details

- [Nibiru's Inflation Module](../arch/advanced/inflation.md) implements the community emmissions for the NIBI token. These emissions go to stakers, the Nibiru Foundation treasury, and the community reserve.
- [NibiruChain/tokenomics GitHub repo](https://github.com/NibiruChain/tokenomics) - This repo shows how the coefficients for the above module are computed.

## Common Questions

Q: When it says that unlocks/vests are linear, does this mean quarterly, monthly, or
another distribution timeline?

> All linear vesting is *continuous* and handled automatically by smart contracts following the math exactly such that a small amount of micro-NIBI unlocks every block. 

Q: Was there a pre-seed venture capital round?

> Nope, the seed round was the first ever venture capital round. 

Q: Are the private investors staking the vesting tokens that are still locked?

> No, there is a fair distribution of the token supply 

Q: From where is the emmission curve derived?

> It's a normalized exponential decay over the community portion of the total
> supply with a decay factor of 0.2. The exact curve is approximated on-chain
> with a 5-order polynomial and implmented in the `x/inflation` module from
> Nibiru's protocol code. There are more details on this [linked in the "Implementation Details" section](#implementation-details).

---

## Disclaimer

> This is not an offering or the solicitation of an offer to purchase tokens. This document may contain hypothetical, future/forward-looking and/or projected figures which are not guaranteed. Although we strive for exact accuracy, actual numbers may vary. The Nibiru Foundation (MTRX Services, Ltd.) makes no representation or warranty, express or implied, as to the completeness or accuracy of this presentation and it is subject to change without notice.
