package keeper_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	tfkeeper "github.com/NibiruChain/nibiru/x/tokenfactory/keeper"
	tftypes "github.com/NibiruChain/nibiru/x/tokenfactory/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ suite.SetupTestSuite = (*TestSuite)(nil)

type TestSuite struct {
	suite.Suite

	ctx sdk.Context
	app *app.NibiruApp

	keeper  tfkeeper.Keeper
	querier tfkeeper.Querier

	genesis tftypes.GenesisState
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(TestSuite))
}

// SetupTest: Runs before each test in the suite. It initializes a fresh app
// and ctx.
func (s *TestSuite) SetupTest() {
	nibiruApp, ctx := testapp.NewNibiruTestAppAndContext()
	s.app = nibiruApp
	s.ctx = ctx
	s.keeper = s.app.TokenFactoryKeeper
	s.genesis = *tftypes.DefaultGenesis()

	s.querier = s.keeper.Querier()
}

func (s *TestSuite) GoCtx() context.Context { return sdk.WrapSDKContext(s.ctx) }
