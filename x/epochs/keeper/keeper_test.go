package keeper_test

import (
	"testing"
	"time"

	"github.com/MatrixDao/matrix/app"
	"github.com/MatrixDao/matrix/x/epochs/keeper"
	"github.com/MatrixDao/matrix/x/epochs/types"
	"github.com/MatrixDao/matrix/x/testutil"
	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"
)

type KeeperTestSuite struct {
	suite.Suite

	app         *app.MatrixApp
	ctx         sdk.Context
	queryClient types.QueryClient
}

func (suite *KeeperTestSuite) SetupTest() {
	matrixApp, ctx := testutil.NewMatrixApp()
	suite.app = matrixApp
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

	suite.Require().Len(allEpochs, 3)
	suite.Require().Equal(allEpochs[0].Identifier, "day") // alphabetical order
	suite.Require().Equal(allEpochs[1].Identifier, "monthly")
	suite.Require().Equal(allEpochs[2].Identifier, "week")
}
