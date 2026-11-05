#!/usr/bin/env bash
# OSS-Fuzz build script for the Nibiru BFT native Go fuzz tests. OSS-Fuzz runs
# it from the repository root inside its Go base-builder image, which provides
# compile_native_go_fuzzer, $OUT, and go-118-fuzz-build. Nibiru is not
# enrolled in OSS-Fuzz yet. Upstream reference:
# https://github.com/google/oss-fuzz/blob/master/projects/tendermint/build.sh
set -euo pipefail

readonly FUZZ_ROOT="github.com/NibiruChain/nibiru/v2/bft"

# build_go_fuzzer <package dir under bft/> <FuzzFunc> <fuzzer name>
# Zips the committed seeds in testdata/fuzz/<FuzzFunc> into the fuzzer's seed
# corpus, then builds the libFuzzer binary.
build_go_fuzzer() {
  local pkg_dir="$1"
  local function="$2"
  local fuzzer="$3"

  go run github.com/orijtech/otils/corpus2ossfuzz@latest \
    -o "$OUT/${fuzzer}_seed_corpus.zip" \
    -corpus "bft/$pkg_dir/testdata/fuzz/$function"
  compile_native_go_fuzzer "$FUZZ_ROOT/$pkg_dir" "$function" "$fuzzer"
}

go get github.com/AdamKorcz/go-118-fuzz-build/testing

build_go_fuzzer test/fuzz/mempool/v0 FuzzMempoolV0 mempool_v0_fuzzer
build_go_fuzzer test/fuzz/mempool/v1 FuzzMempoolV1 mempool_v1_fuzzer
build_go_fuzzer test/fuzz/p2p/addrbook FuzzP2PAddrBook p2p_addrbook_fuzzer
build_go_fuzzer test/fuzz/p2p/pex FuzzP2PPex p2p_pex_fuzzer
build_go_fuzzer test/fuzz/p2p/secret_connection FuzzP2PSecretConnection \
  p2p_secret_connection_fuzzer
build_go_fuzzer test/fuzz/rpc/jsonrpc/server FuzzRPCJSONRPCServer \
  rpc_jsonrpc_server_fuzzer
build_go_fuzzer libs/pubsub/query FuzzPubsubQuery pubsub_query_fuzzer
