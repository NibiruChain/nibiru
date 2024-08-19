package embeds_test

import (
	_ "embed"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
)

func TestLoadContracts(t *testing.T) {
	require.NotPanics(t, func() {
		embeds.SmartContract_ERC20Minter.MustLoad()
		embeds.SmartContract_FunToken.MustLoad()
		embeds.SmartContract_TestERC20.MustLoad()
	})
}
