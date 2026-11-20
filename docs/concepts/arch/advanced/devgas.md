---
order: 4
title: "Dev Gas (Retired) | Nibiru"
description: >-
  The devgas module shared contract execution fees with smart contract
  developers. Nibiru retired it.
---

# Dev Gas (Retired)

Module `devgas` is retired. It paid a contract's deployer a share of the gas
fees on transactions that called the contract. Zero-gas transactions pay no
fee, so there is nothing to share.

Registering, updating, and canceling a fee share all fail, and transactions pay
out no fee share. The read-only queries under `nibiru q devgas` and the
[REST endpoints](../../../dev/rest-api/nibiru-devgas.md) still work.
