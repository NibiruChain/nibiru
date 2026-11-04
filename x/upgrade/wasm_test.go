package upgrade

import (
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEmbeddedWasm(t *testing.T) {
	code := ContractWasm()
	require.Greater(t, len(code), 8)
	require.Equal(t, []byte{0, 97, 115, 109}, code[:4])
	require.Equal(t, ContractWasmSHA256, fmt.Sprintf("%x", sha256.Sum256(code)))
	code[0] = 255
	require.Equal(t, byte(0), ContractWasm()[0], "callers cannot mutate embedded code")
}
