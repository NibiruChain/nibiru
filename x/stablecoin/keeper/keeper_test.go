package keeper_test

import (
	"testing"

	"github.com/MatrixDao/matrix/app"
	"github.com/MatrixDao/matrix/x/stablecoin/types"
	"github.com/MatrixDao/matrix/x/testutil"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"

	// For integration testing
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"
)

// TODO: Move to CLI for integrations
type KeeperTestSuite struct {
	suite.Suite

	ctx sdk.Context
	app *app.MatrixApp

	clientCtx   client.Context
	queryClient types.QueryClient
}

func TestKeeperTestSuite(t *testing.T) {
	var keeperTestSuite *KeeperTestSuite = new(KeeperTestSuite)
	suite.Run(t, keeperTestSuite)

	// Connects Ginkgo to Gomega. When a matcher fails, the fail handler passed
	// to RegisterFailHandler is called.
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Keeper Suite")
}

func (suite *KeeperTestSuite) SetupTest() {
	suite._doSetupTest()
}

func (suite *KeeperTestSuite) _doSetupTest() {
	matrixApp, ctx := testutil.NewMatrixApp()
	suite.app = matrixApp
	suite.ctx = ctx

	queryGrpcClientConn := baseapp.NewQueryServerTestHelper(
		suite.ctx, suite.app.InterfaceRegistry(),
	)
	types.RegisterQueryServer(queryGrpcClientConn, suite.app.StablecoinKeeper)
	suite.queryClient = types.NewQueryClient(queryGrpcClientConn)
}
