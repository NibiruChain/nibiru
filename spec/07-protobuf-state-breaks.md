# What causes protobuf state breaks on a node

- Status: active
- Code scope: `bft/proto/`, `proto/`, `lib/cosmos-sdk/proto/`,
  `lib/ibc-go/proto/`, and the generated `*.pb.go` files
- Source of truth: the `.proto` files, the proto3 wire format, and the hashing
  code in `bft/types/block.go` and `bft/types/results.go`

A node breaks when it reads bytes that were written by a different definition
than the one it was compiled with. Those bytes live in many places: the block
store, the state DB, the app's IAVL store, peers' p2p messages, gRPC clients,
and key files on disk. If the reader and writer disagree, the node either fails
to decode, or decodes cleanly and computes a different hash. The second case is
worse, because it shows up as an app hash mismatch and a halted chain, not as a
parse error.

This spec lists what is safe to change and what is not. It was written when
CometBFT v0.37.18 was vendored into `bft/`. That change moved every Go import
path and broke nothing, because nothing in the first group below reaches the
wire.

## Safe: changes that only affect generated Go code

These change where Go code lives or what the Go types look like. The bytes on
the wire stay identical, so old and new binaries agree.

- **`option go_package`.** This only tells `protoc-gen-gogo` which Go package
  to write. Example: changing
  `option go_package = "github.com/cometbft/cometbft/proto/tendermint/types";`
  to `".../nibiru/v2/bft/proto/tendermint/types"` in
  `bft/proto/tendermint/types/types.proto`. Every `tendermint.types.Header` is
  encoded exactly as before.
- **Go import paths.** Rewriting
  `import cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"` to the
  `bft/` path changes nothing outside the compiler. This is the whole vendoring
  change.
- **The directory generated code lands in.** Generating into `bft/proto/...`
  instead of `lib/cosmos-sdk/...` is fine, as long as only one copy is linked.
  See [Two copies in one binary](#two-copies-in-one-binary) below.
- **`gogoproto.customname`.** This renames the Go field only. Example:
  `string chain_id = 2 [(gogoproto.customname) = "ChainID"];` gives the Go
  field `ChainID` instead of `ChainId`. The wire still carries field number 2.
- **`gogoproto.casttype`.** This changes the Go type of a `bytes` or scalar
  field, not its encoding. Example: the four options that now point at
  `.../v2/bft/libs/bytes.HexBytes`, such as
  `proto/cosmwasm/wasm/v1/types.proto:132`. `HexBytes` is still `[]byte` on the
  wire.
- **`gogoproto.stdtime` and `gogoproto.stdduration`.** These swap the Go type
  between `*types.Timestamp` and `time.Time`. The wire is still a
  `google.protobuf.Timestamp`.
- **Renaming a field in the `.proto` file, binary encoding only.** The binary
  wire format carries field numbers, never names. Renaming `last_block_id` to
  `prev_block_id` in `Header` would not change the bytes or the header hash. It
  does change JSON output, so see the next section before doing it.
- **Comments, formatting, and file order inside a `.proto`.**

The vendoring commit also left the old `go_package` string inside each
generated file's gzipped descriptor, because the `.pb.go` files were not
regenerated. That string is never sent or stored, so it is harmless. The first
regeneration from `bft/proto` will show it as a diff.

## Breaking: changes that reach the wire or stored data

Each of these changes bytes that another node, a client, or the node's own
disk depends on.

### Names that are written down somewhere

Full names show up in `Any` type URLs, gRPC routes, and the global type
registry. Renaming the proto package or a message changes all of them.

- **Proto package names.** `package tendermint.types;` must stay. Renaming it
  to `package nibiru.bft.types;` changes the full name of every message in it,
  from `tendermint.types.Header` to `nibiru.bft.types.Header`.
- **Message full names inside `Any`.** The SDK stores many values as
  `google.protobuf.Any`, whose `type_url` is the message full name. Example: a
  validator's consensus key in `x/staking` state is stored with type URL
  `/cosmos.crypto.ed25519.PubKey`. Rename that message and every validator
  record in state fails to unpack.
- **gRPC service paths.** The route is built from the package, service, and
  method names. Example: `/tendermint.rpc.grpc.BroadcastAPI/BroadcastTx` in
  `bft/proto/tendermint/rpc/grpc/types.pb.go:268`, and the SDK's
  `/cosmos.base.tendermint.v1beta1.Service/GetLatestBlock`. Rename either and
  every client calling it gets `Unimplemented`.
- **Amino JSON type names.** These are not proto, but they break the same way.
  Example: `PubKeyName = "tendermint/PubKeyEd25519"` in
  `bft/crypto/ed25519/ed25519.go:22`. That string is written into every
  validator's `priv_validator_key.json` and into genesis files. Change it and a
  node can no longer load its own signing key.
- **JSON field names.** Field names appear in JSON. Renaming a field, or
  changing `gogoproto.jsontag`, changes REST and RPC output, genesis files, and
  any signing mode that signs JSON, such as the SDK's legacy amino JSON sign
  mode. Old signatures stop verifying.

### Field numbers and types

The binary format is a list of `(field number, wire type, value)` entries.
Changing either number or type changes what the bytes mean.

- **Changing a field number.** If `int64 height = 3;` in `Header` becomes
  `= 9`, old nodes read new headers with no height, and new nodes read old
  headers the same way.
- **Reusing a retired field number.** Upstream reserves numbers it removed so
  nobody reuses them. Example: `reserved 3;  // was TimeIotaMs` in
  `bft/proto/tendermint/types/params.proto:29`. Giving number 3 to a new field
  would make old stored params decode into the wrong field.
- **Changing a field's type to one with a different wire type.** `int64` to
  `string`, or a scalar to a message, makes old bytes undecodable.
- **Changing between compatible-looking types.** `int32` to `uint32` or
  `sint32` decodes without an error but gives different values for negative
  numbers. `int64` to `fixed64` changes the wire type entirely.
- **Changing a field from singular to `repeated`, or to `oneof`.** Some of
  these decode, but the meaning of stored data changes.

### Bytes that get hashed

These changes keep decoding working but change a hash that every validator
must agree on. The result is a consensus failure, not a parse error.

- **Anything in the block header hash.** `Header.Hash()` in
  `bft/types/block.go:446` hashes the proto encoding of each header field
  separately: `Version`, `ChainID`, `Height`, `Time`, `LastBlockID`, and the
  rest. Adding a field to `tendermint.version.Consensus`, or changing how
  `BlockID` encodes, changes every block hash.
- **Fields included in `LastResultsHash`.** `NewResults` in
  `bft/types/results.go:13` keeps only the deterministic fields of each
  `ResponseDeliverTx`: `Code`, `Data`, `GasWanted`, and `GasUsed`. Adding a
  field to `ResponseDeliverTx` is safe, because it gets stripped. Adding it to
  that deterministic list, or changing how one of those four encodes, is a
  consensus break.
- **`gogoproto.nullable = false` on a message field.** This looks like a
  Go-only option, but it changes encoding. A non-nullable embedded message is
  always written, even when empty, so it adds a zero-length field to the bytes.
  A nullable one is skipped when nil. Flipping it on a field that ends up in a
  hash, such as `PartSetHeader part_set_header = 2 [(gogoproto.nullable) = false];`
  in `BlockID`, changes that hash.
- **App state written to IAVL.** Every module stores proto-encoded values, and
  the app hash covers them. A non-deterministic encoding change in any stored
  message, in any module, shows up as an app hash mismatch at the first block
  that writes it.

## Two copies in one binary

Linking two generated copies of the same `.proto` files breaks the node at
startup, before it reads any data.

- Each generated file registers its message and enum names in a global
  registry when the program starts.
- `cosmos/gogoproto` v1.7.2 panics on a duplicate enum:
  `panic("proto: duplicate enum registered: " + typeName)` in
  `proto/properties.go:525` of the module cache.
- Example: during the vendoring, `bft/proto/tendermint/types` and upstream
  `github.com/cometbft/cometbft/proto/tendermint/types` both register
  `tendermint.types.BlockIDFlag`. Any binary that imports both panics.
- Duplicate messages only log a warning, so the enums are what catch this.

That is why the import switch outside `bft/` had to be one sweep, and why
`go_mod_test.go` bans `github.com/cometbft/cometbft`.

## How to check a proto change

- Run `buf breaking` against the previous release tag, for wire and JSON
  compatibility.
- Ask whether the change touches anything that is hashed: the header, block
  ID, results, or app state.
- Sync a node from the previous release over p2p, and replay blocks from a
  snapshot. During the vendoring, a released `nibid` on upstream CometBFT synced
  224 blocks from an in-tree node and reached the tip. That shows the wire
  format and app hashes still matched.
- Load an existing `priv_validator_key.json` and genesis file with the new
  binary.
