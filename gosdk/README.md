# Nibiru Go SDK - NibiruChain/nibiru/gosdk

A Golang client for interacting with the Nibiru blockchain. 

The Nibiru Go SDK extends the core blockchain logic with extensions to build
external clients for the Nibiru blockchain and easily access its query and
transaction types.

--- 

## Dev Notes - Nibiru Go SDK

### Finalizing "v1"

- [ ] Migrate to the [Nibiru repo](https://github.com/NibiruChain/nibiru) and archive this one.
- [ ] feat: add in transaction broadcasting
  - [x] initialize keyring obj with mnemonic
  - [x] initialize keyring obj with priv key
  - [x] write a test that sends a bank transfer.
  - [ ] write a test that submits a text gov proposal.
- [x] docs: for grpc.go
- [ ] docs: for clients.go

### Usage Guides & Dev Ex

- [ ] Create a quick start guide: Connecting to Nibiru, querying Nibiru, sending
transactions on Nibiru
- [ ] Write usage examples
  - [ ] Creating an account and keyring
  - [ ] Querying balanaces
  - [ ] Broadcasting txs to transfer funds
  - [ ] Querying Wasm smart contracts
  - [ ] Broadcasting txs to transfer funds
- [x] impl Tendermint RPC client
- [x] refactor: DRY improvements on the QueryClient initialization
- [x] ci: Add go tests to CI
- [x] ci: Add code coverage to CI
- [x] ci: Add linting to CI

### Feature Backlog

- [ ] impl wallet abstraction for the keyring
- [ ] epic: storing transaction history storage 

### Question Brain-dump

Q: Should gosdk run as a binary?  

No, or at least, not initially. Since the software required to operate a full
node has more cumbersome dependencies like RocksDB that involve C-Go and compled
build steps, we may benefit from splitting "start" command from the bulk of the
subcommands available on teh Nibiru CLI. This would make it much easier to have a
command line tool that builds on Linux, Windows, Mac.

Q: Should there be a way to run queries with JSON-RPC 2 instead of GRPC?

We [implemented this in
python](https://github.com/NibiruChain/py-sdk/tree/v0.21.12/nibiru/jsonrpc)
without too much trouble, and it's not taxing to maintain. If we're going to
prioritize adding APIs for the CometBFT JSON-RPC methods, it should be in the
[Nibiru TypeScript SDK](https://github.com/NibiruChain/ts-sdk) first.
