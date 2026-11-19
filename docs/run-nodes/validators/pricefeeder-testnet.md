---
order: 3
description: >-
  Validators no longer run a pricefeeder. Nibiru retired native oracle feeder
  duties in v2.12.
---

# Pricefeeder (Retired)

{{ $frontmatter.description }}

Native module `x/oracle` is deprecated as of v2.12. Validators don't vote on
prices, delegate feeder consent, or get slashed for missed votes, and the
module's messages return an error. There is no pricefeeder to install, so skip
it when setting up a validator.

Prices now come from the Sai oracle. See
[Querying Prices From CosmWasm](../../dev/cw/oracle.md) and the
[Oracle Precompile](../../evm/precompiles/oracle.md) for how contracts read
them, and the [Sai developer documentation](https://sai.fun/docs/dev/) for the
oracle itself.
