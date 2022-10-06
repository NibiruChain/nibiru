# Client                    <!-- omit in toc -->

This page describes the queries structure of the module. These descriptions are accompanied by the documentation for their corresponding CLI commands.

- [GetVpoolReserveAssets](#getvpoolreserveassets)
  - [GetVpoolReserveAssets CLI command](#getvpoolreserveassets-cli-command)
- [GetVpools](#getvpools)
  - [GetVpools CLI command](#getvpools-cli-command)
- [GetBaseAssetPrice](#getbaseassetprice)
  - [GetBaseAssetPrice CLI command](#getbaseassetprice-cli-command)

---

## GetVpoolReserveAssets

`GetVpoolReserveAssets` defines a method for querying the reserve assets of a specific pool.
It returns the base and quote reserves for the pool.

### GetVpoolReserveAssets CLI command

```sh
nibid q vpool reserve-assets --pair
```

| Flag   | Description                      |
| ------ | -------------------------------- |
| `pair` | Identifier for the virtual pool. |

---

## GetVpools

`GetVpools` defines a method for querying the parameters and state for all pools.

### GetVpools CLI command

```sh
nibid q vpools
```

---

## GetBaseAssetPrice

`GetBaseAssetPrice` defines a method for querying the reserve assets of a specific pool.
It returns the base and quote reserves for the pool.

### GetBaseAssetPrice CLI command

```sh
nibid q vpool prices --pair --direction --base-asset-amount
```

| Flag                | Description                      |
| ------------------- | -------------------------------- |
| `pair`              | Identifier for the virtual pool. |
| `direction`         | Either `add` or `remove`.        |
| `base-asset-amount` | The amount to swap in base unit  |
