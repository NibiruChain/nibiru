package keeper_test

import (
	"testing"

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
	types.RegisterQueryServer(queryHelper, keeper.NewQuerier(*suite.app.EpochsKeeper))
	suite.queryClient = types.NewQueryClient(queryHelper)
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}
