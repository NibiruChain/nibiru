# USDC Migration Report

Reconstruct the v2.23 collection and repayment record from a saved Comet
`block_results` response. Only the returned BeginBlock events are read. No
pre-upgrade balance file, holder registry query, wallet, or signing key is needed.

```sh
curl -fsS 'https://rpc.nibiru.fi/block_results?height=UPGRADE_HEIGHT' > block-results.json
go run ./contrib/scripts/usdc-migration-report -input block-results.json > redistribution.json
```

Replace `UPGRADE_HEIGHT` with the executed height and use an RPC retaining that
block's results. The program accepts a full JSON-RPC response or its result object,
checks the reported block height against the migration frame, and writes JSON
only after the event accounting reconciles. RPC errors or missing/inconsistent
events return a nonzero exit status rather than a partial repayment file.

`collected_base_units`, `repayment_base_units`, and `custody_recovery_base_units`
are exact integer quantities. Legacy USDC uses six decimals. Each account records
its original bank/ERC20 representation, original address, actual collected amount,
and disposition. Failed, zero-balance, and excluded accounts remain visible but
do not count as repayments. Bank conversions must have their original holder and
backing-transfer events; conversion plus forwarding is counted only once.

The output describes principal allocations after the bridge round trip. It does
not broadcast repayments or decide bridge costs, token approvals, or funding.
