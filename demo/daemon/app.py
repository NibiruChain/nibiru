from lib.operator import Operations, Operation


def perp_params(result: dict):
    out = result["params"]

    out["maintenance_margin_ratio"] = float(out["maintenance_margin_ratio"])

    million = 1_000_000
    out["toll_ratio"] = int(out["toll_ratio"]) / million
    out["spread_ratio"] = int(out["spread_ratio"]) / million
    out["liquidation_fee"] = int(out["liquidation_fee"]) / million
    out["partial_liquidation_ratio"] = int(out["partial_liquidation_ratio"]) / million
    return out


def main():
    operation_info = [
        {
            "name": "Query perp params",
            "table_name": "perp_params",
            "args": ["q", "perp", "params"],
            "interval": 2,
            "etl_fn": perp_params,
        },
    ]

    operations = Operations([Operation(**op_info) for op_info in operation_info])
    operations.launch()


if __name__ == "__main__":
    main()
    while True:
        pass
