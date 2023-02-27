package integration_test

import (
	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/asset"
	testutilevents "github.com/NibiruChain/nibiru/x/common/testutil"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	vpooltypes "github.com/NibiruChain/nibiru/x/vpool/types"
	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"testing"
)

type GivenAction interface {
	Do(app *app.NibiruApp, ctx sdk.Context)
}

type CreatePoolAction struct {
	Pair asset.Pair

	Quote sdk.Dec
}

func TestHappyPath(t *testing.T) {
	nibiruApp, ctx := testapp.NewNibiruTestAppAndContext(true)

	err := nibiruApp.VpoolKeeper.CreatePool(
		ctx,
		"ubtc:uusdc",
		sdk.NewDec(1000),
		sdk.NewDec(100),
		vpooltypes.DefaultVpoolConfig(),
	)
	require.NoError(t, err)

	// open short position Alice
	aliceAccount := testutilevents.AccAddress()
	simapp.FundAccount(
		nibiruApp.BankKeeper,
		ctx,
		aliceAccount,
	)

}
