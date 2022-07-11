package keeper_test

import (
	gocontext "context"
	"time"

	"github.com/NibiruChain/nibiru/x/epochs/types"
)

func (suite *KeeperTestSuite) TestQueryEpochInfos() {
	suite.SetupTest()
	queryClient := suite.queryClient

	chainStartTime := suite.ctx.BlockTime()

	// Invalid param
	epochInfosResponse, err := queryClient.EpochInfos(gocontext.Background(), &types.QueryEpochsInfoRequest{})
	suite.Require().NoError(err)
	suite.Require().Len(epochInfosResponse.Epochs, 4)

	// check if EpochInfos are correct
	suite.Require().Equal(epochInfosResponse.Epochs[2].Identifier, "day")
	suite.Require().Equal(epochInfosResponse.Epochs[2].StartTime, chainStartTime)
	suite.Require().Equal(epochInfosResponse.Epochs[2].Duration, time.Hour*24)
	suite.Require().Equal(epochInfosResponse.Epochs[2].CurrentEpoch, int64(0))
	suite.Require().Equal(epochInfosResponse.Epochs[2].CurrentEpochStartTime, chainStartTime)
	suite.Require().Equal(epochInfosResponse.Epochs[2].EpochCountingStarted, false)
	suite.Require().Equal(epochInfosResponse.Epochs[3].Identifier, "week")
	suite.Require().Equal(epochInfosResponse.Epochs[3].StartTime, chainStartTime)
	suite.Require().Equal(epochInfosResponse.Epochs[3].Duration, time.Hour*24*7)
	suite.Require().Equal(epochInfosResponse.Epochs[3].CurrentEpoch, int64(0))
	suite.Require().Equal(epochInfosResponse.Epochs[3].CurrentEpochStartTime, chainStartTime)
	suite.Require().Equal(epochInfosResponse.Epochs[3].EpochCountingStarted, false)
}
