package consensus

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"fmt"
	"io"
	"sync"

	wasmtypes "github.com/NibiruChain/nibiru/v2/x/wasm/types"
)

// ContractWasmSHA256 is the hex SHA-256 of ContractWasm, which is also the
// on-chain Wasm code checksum. It fixes the contract address
// (keeper.PredictContractAddr). The app finds the contract through the Wasm
// creator index (keeper.Keeper.ContractAddr), not that address, so rebuilding
// the embedded Wasm is safe; only published addresses go stale.
//
// This constant is the only copy of the hash in Go, and TestEmbeddedWasm
// checks it against the embedded bytes. To update the build, follow the
// "Updating the embedded build" steps in wasm-contracts/x-consensus/README.md
// and set this constant to the SHA-256 of the uncompressed .wasm.
const ContractWasmSHA256 = "f1ac9367879aac63755370c4ff779b8d684598f3a43021306764376ce5c3912e"

// contractWasmGz is the gzipped x-consensus release build that ships in
// nibid. Gzip cuts the embedded size by about two thirds and keeps rebuild
// diffs small, as for other gzipped Wasm in the repo.
//
//   - Source: wasm-contracts/x-consensus.
//   - Build: the release optimizer, just rs optimizer build --manifest-path
//     Cargo.toml -p x-consensus --output-dir wasm-contracts/artifacts
//     --mode release, so the embedded code can be rebuilt from source.
//   - Compression: gzip -9 -n, the same convention as the repo's gzipped Wasm
//     fixtures. -n drops the name and timestamp so recompressing gives the
//     same file. Only the decompressed bytes are checked; the .gz is
//     packaging and may differ between gzip versions.
//
//go:embed x_consensus.wasm.gz
var contractWasmGz []byte

// contractWasm decompresses contractWasmGz once, on first use.
var contractWasm = sync.OnceValue(func() []byte {
	return mustGunzip(contractWasmGz)
})

// ContractWasm returns the optimized x-consensus bytecode, decompressed from
// the embedded x_consensus.wasm.gz. Every caller (genesis and the v2.22
// upgrade through keeper.Keeper.EnsureXConsensus, and tests) works on the
// exact code the chain stores, whose SHA-256 is ContractWasmSHA256.
//
// Decompression is lazy, behind sync.OnceValue, rather than at package init.
// Only processes that create the contract need the bytes, so every other
// nibid command (keys, queries, tx signing, a node that is past the upgrade)
// skips the work and the 178 KB allocation. A corrupt archive therefore
// panics at the first call instead of at startup, and OnceValue repeats the
// panic on every later call. TestEmbeddedWasm calls it, so a broken archive
// fails CI before a release.
//
// Each call returns a fresh copy, so a caller that modifies the slice cannot
// change the code another caller stores.
func ContractWasm() []byte {
	return bytes.Clone(contractWasm())
}

// mustGunzip decompresses the embedded build, reading at most
// wasmtypes.MaxWasmSize bytes, the Wasm module's own limit on stored code, so
// the cap rejects nothing the chain would accept. It panics on a corrupt or
// oversized archive: the bytes are fixed at build time, so a failure is a
// broken release.
func mustGunzip(gz []byte) []byte {
	r, err := gzip.NewReader(bytes.NewReader(gz))
	if err != nil {
		panic(fmt.Sprintf("x-consensus: open embedded gzip: %v", err))
	}
	defer r.Close()
	code, err := io.ReadAll(io.LimitReader(r, int64(wasmtypes.MaxWasmSize)+1))
	if err != nil {
		panic(fmt.Sprintf("x-consensus: read embedded gzip: %v", err))
	}
	if len(code) > wasmtypes.MaxWasmSize {
		panic(fmt.Sprintf("x-consensus: embedded Wasm exceeds %d bytes",
			wasmtypes.MaxWasmSize))
	}
	return code
}
