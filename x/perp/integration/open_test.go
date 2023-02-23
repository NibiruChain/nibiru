package integration_test

import (
	"github.com/NibiruChain/nibiru/x/common/testutil/genesis"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	perptypes "github.com/NibiruChain/nibiru/x/perp/types"
	"testing"
)

func TestHappyPath(t *testing.T) {
	//nibiruApp, ctx := testapp.NewNibiruTestAppAndContext(false)
	//
	//nibiruApp.
	gen := genesis.NewTestGenesisState()

	perpRawGenesis := gen[perptypes.ModuleName]
	perp := perptypes.GenesisState{}

	appChain := testapp.NewNibiruTestApp()
}
