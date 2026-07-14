//go:build cgo

package wasmvm

import (
	"github.com/NibiruChain/nibiru/v2/lib/wasmvm/internal/api"
)

func libwasmvmVersionImpl() (string, error) {
	return api.LibwasmvmVersion()
}
