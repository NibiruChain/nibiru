package keeper_test

import (
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/epochs/keeper"
	"github.com/NibiruChain/nibiru/x/epochs/types"
	"github.com/NibiruChain/nibiru/x/testutil/testapp"
)

type KeeperTestSuite struct {
	suite.Suite

	app         *app.NibiruApp
	ctx         sdk.Context
	queryClient types.QueryClient
}

func (suite *KeeperTestSuite) SetupTest() {
	nibiruApp, ctx := testapp.NewNibiruAppAndContext(true)
	suite.app = nibiruApp
	suite.ctx = ctx

	queryHelper := baseapp.NewQueryServerTestHelper(suite.ctx, suite.app.InterfaceRegistry())
	types.RegisterQueryServer(queryHelper, keeper.NewQuerier(suite.app.EpochsKeeper))
	suite.queryClient = types.NewQueryClient(queryHelper)
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}

func (suite *KeeperTestSuite) TestEpochLifeCycle() {
	suite.SetupTest()

	epochInfo := types.EpochInfo{
		Identifier:            "monthly",
		StartTime:             time.Time{},
		Duration:              time.Hour * 24 * 30,
		CurrentEpoch:          0,
		CurrentEpochStartTime: time.Time{},
		EpochCountingStarted:  false,
	}
	suite.app.EpochsKeeper.SetEpochInfo(suite.ctx, epochInfo)
	epochInfoSaved := suite.app.EpochsKeeper.GetEpochInfo(suite.ctx, "monthly")
	suite.Require().Equal(epochInfo, epochInfoSaved)

	allEpochs := suite.app.EpochsKeeper.AllEpochInfos(suite.ctx)

	suite.Require().Len(allEpochs, 5)
	suite.Require().Equal("15 min", allEpochs[0].Identifier) // alphabetical order
	suite.Require().Equal("30 min", allEpochs[1].Identifier) // alphabetical order
	suite.Require().Equal("day", allEpochs[2].Identifier)
	suite.Require().Equal("monthly", allEpochs[3].Identifier)
	suite.Require().Equal("week", allEpochs[4].Identifier)
}
