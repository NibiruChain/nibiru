package keeper_test

import (
	"testing"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/stablecoin/types"
	"github.com/NibiruChain/nibiru/x/testutil"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"

	// For integration testing
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// TODO: Move to CLI for integrations
type KeeperTestSuite struct {
	suite.Suite

	ctx sdk.Context
	app *app.NibiruApp

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
	nibiruApp, ctx := testutil.NewNibiruApp(true)
	suite.app = nibiruApp
	suite.ctx = ctx

	queryGrpcClientConn := baseapp.NewQueryServerTestHelper(
		suite.ctx, suite.app.InterfaceRegistry(),
	)
	types.RegisterQueryServer(queryGrpcClientConn, suite.app.StablecoinKeeper)
	suite.queryClient = types.NewQueryClient(queryGrpcClientConn)
}

// Params

func TestGetAndSetParams(t *testing.T) {

	var testName string

	testName = "Get default Params"
	t.Run(testName, func(t *testing.T) {
		nibiruApp, ctx := testutil.NewNibiruApp(true)
		stableKeeper := &nibiruApp.StablecoinKeeper

		params := types.DefaultParams()
		stableKeeper.SetParams(ctx, params)

		require.EqualValues(t, params, stableKeeper.GetParams(ctx))
	})

	testName = "Get non-default params"
	t.Run(testName, func(t *testing.T) {
		nibiruApp, ctx := testutil.NewNibiruApp(true)
		stableKeeper := &nibiruApp.StablecoinKeeper

		collRatio := sdk.MustNewDecFromStr("0.5")
		feeRatio := collRatio
		feeRatioEF := collRatio
		bonusRateRecoll := sdk.MustNewDecFromStr("0.002")
		params := types.NewParams(
			collRatio, feeRatio, feeRatioEF, bonusRateRecoll)
		stableKeeper.SetParams(ctx, params)

		require.EqualValues(t, params, stableKeeper.GetParams(ctx))
	})

	testName = "Calling Get without setting causes a panic"
	t.Run(testName, func(t *testing.T) {
		nibiruApp, ctx := testutil.NewNibiruApp(false)
		stableKeeper := &nibiruApp.StablecoinKeeper

		require.Panics(
			t,
			func() { stableKeeper.GetParams(ctx) },
		)
	})

}
