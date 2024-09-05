# Mocks Generation

To generate mocks, install `mockery` tool:
https://vektra.github.io/mockery/latest/installation/

```bash
cd x/evm
mockery \
  --name QueryClient \
  --filename evm_query_client.go \
  --output ../../eth/rpc/backend/mocks \
  --structname EVMQueryClient
```
