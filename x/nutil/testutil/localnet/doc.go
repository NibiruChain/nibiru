/*
Package localnet provides test helpers for connecting Go tests to an already-
running Nibiru localnet.

This package does not start or manage an in-process chain. Instead, it assumes a
local validator has already been bootstrapped, typically by
"contrib/scripts/localnet.sh", and exposes the fixed localnet contract that the
test suites in this repository rely on:

  - chain ID: "nibiru-localnet-0"
  - Tendermint RPC: "http://localhost:26657"
  - EVM JSON-RPC: "http://127.0.0.1:8545"
  - node home: "~/.nibid"
  - keyring backend: "test"
  - validator key name: "validator"
  - default tx fees: "1000unibi"
  - default tx gas: "5000000"

NewCLI builds a live Ethereum JSON-RPC client and `client.Context` against that
localnet, using the known validator key from the local test keyring. The
returned CLI value bundles:

  - query and tx command execution through Cobra handlers
  - block and tx wait helpers for delivered-state assertions
  - the live client.Context and tx defaults used by CLI-oriented tests
  - a local rpcapi.Backend for direct backend method calls
  - in-process typed `rpcapi` implementations for `eth`, `net`, and `debug`
  - an `ethclient.Client` for Ethereum JSON-RPC assertions

This helper is used by suites that exercise CLI, SDK, and Ethereum RPC behavior
against the running localnet instead of starting a fresh testnetwork.Network.
Because the underlying chain is persistent, callers should treat localnet state
as long-lived, create fresh per-test actors when needed, and avoid assumptions
that require a clean genesis on every run.

A typical flow is:

 1. gate the suite with nutil.EnsureLocalBlockchain()
 2. call localnet.NewCLI()
 3. use ExecQueryCmd(...) and ExecTxCmd(...) for in-process CLI handlers
 4. use WaitForNextBlock(), WaitForHeight(...), or WaitForTx(...) when the test
    depends on delivered state
 5. use EthRpcBackend or EvmRpcClient when the suite also needs direct backend
    or Ethereum JSON-RPC access
*/
package localnet
