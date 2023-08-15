package keeper_test

import (
	"testing"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/stretchr/testify/suite"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/x/devgas/v1/keeper"
	"github.com/NibiruChain/nibiru/x/devgas/v1/types"
)

// BankKeeper defines the expected interface needed to retrieve account balances.
type BankKeeper interface {
	MintCoins(ctx sdk.Context, moduleName string, amt sdk.Coins) error
	SendCoins(ctx sdk.Context, fromAddr, toAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromAccountToModule(ctx sdk.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx sdk.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromModuleToModule(ctx sdk.Context, senderModule, recipientModule string, amt sdk.Coins) error
}

type IntegrationTestSuite struct {
	suite.Suite

	ctx               sdk.Context
	app               *app.NibiruApp
	bankKeeper        BankKeeper
	accountKeeper     types.AccountKeeper
	queryClient       types.QueryClient
	feeShareMsgServer types.MsgServer
	wasmMsgServer     wasmtypes.MsgServer
}

func (s *IntegrationTestSuite) SetupTest() {
	nibiruApp, ctx := testapp.NewNibiruTestAppAndContext()
	s.app = nibiruApp
	s.ctx = ctx

	queryHelper := baseapp.NewQueryServerTestHelper(s.ctx, s.app.InterfaceRegistry())
	types.RegisterQueryServer(queryHelper, keeper.NewQuerier(s.app.DevGasKeeper))

	s.queryClient = types.NewQueryClient(queryHelper)
	s.bankKeeper = s.app.BankKeeper
	s.accountKeeper = s.app.AccountKeeper
	s.feeShareMsgServer = s.app.DevGasKeeper
	s.wasmMsgServer = wasmkeeper.NewMsgServerImpl(&s.app.WasmKeeper)
}

func (s *IntegrationTestSuite) SetupSuite() {
	s.SetupTest()
}

func (s *IntegrationTestSuite) FundAccount(
	ctx sdk.Context, addr sdk.AccAddress, amounts sdk.Coins,
) error {
	return testapp.FundAccount(s.app.BankKeeper, ctx, addr, amounts)
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
