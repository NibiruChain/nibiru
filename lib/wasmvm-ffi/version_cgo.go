//go:build cgo

package cosmwasm

import (
	"github.com/NibiruChain/nibiru/v2/lib/wasmvm-ffi/internal/api"
)

func libwasmvmVersionImpl() (string, error) {
	return api.LibwasmvmVersion()
}
