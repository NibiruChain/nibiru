package embeds_test

import (
	_ "embed"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/evm/embeds"
)

//go:embed artifacts/contracts/TestERC20.sol/TestERC20.json
var testErc20Json []byte

var SmartContract_ERC20Minter = embeds.CompiledEvmContract{
	Name:      "TestERC20.sol",
	EmbedJSON: testErc20Json,
}

func TestLoadContracts(t *testing.T) {
	require.NotPanics(t, func() {
		SmartContract_ERC20Minter.MustLoad()
	})
}
