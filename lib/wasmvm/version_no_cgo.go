//go:build !cgo

package wasmvm

import (
	"fmt"
)

func libwasmvmVersionImpl() (string, error) {
	return "", fmt.Errorf("libwasmvm unavailable since cgo is disabled")
}
