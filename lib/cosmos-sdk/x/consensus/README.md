---
sidebar_position: 1
---

# `x/consensus`

> **Deprecated.** NibiruApp no longer uses this module. Nibiru's own
> [`x/consensus`](../../../../x/consensus) module owns the `consensus` store,
> serves the Params query, and rejects `MsgUpdateParams`, because the
> [x-consensus contract](../../../../wasm-contracts/x-consensus) owns the
> consensus params. The types moved to `x/consensus/types`. This copy is kept
> only for lib's own depinject test apps and the ibc-go simapps, and will be
> removed together with that test wiring.

Functionality to modify CometBFT's ABCI consensus params.
