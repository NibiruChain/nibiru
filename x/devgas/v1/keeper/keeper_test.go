package keeper_test

import (
	"testing"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/stretchr/testify/suite"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/x/common/testutil/testapp"
	devgaskeeper "github.com/NibiruChain/nibiru/v2/x/devgas/v1/keeper"
	devgastypes "github.com/NibiruChain/nibiru/v2/x/devgas/v1/types"
)

// BankKeeper defines the expected interface needed to retrieve account balances.
type BankKeeper interface {
	MintCoins(ctx sdk.Context, moduleName string, amt sdk.Coins) error
	SendCoins(ctx sdk.Context, fromAddr, toAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromAccountToModule(ctx sdk.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx sdk.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromModuleToModule(ctx sdk.Context, senderModule, recipientModule string, amt sdk.Coins) error
}

type KeeperTestSuite struct {
	suite.Suite

	ctx             sdk.Context
	app             *app.NibiruApp
	queryClient     devgastypes.QueryClient
	devgasMsgServer devgastypes.MsgServer
	wasmMsgServer   wasmtypes.MsgServer
}

func (s *KeeperTestSuite) SetupTest() {
	nibiruApp, ctx := testapp.NewNibiruTestAppAndContext()
	s.app = nibiruApp
	s.ctx = ctx

	queryHelper := baseapp.NewQueryServerTestHelper(
		s.ctx, s.app.InterfaceRegistry(),
	)
	devgastypes.RegisterQueryServer(
		queryHelper, devgaskeeper.NewQuerier(s.app.DevGasKeeper),
	)

	s.queryClient = devgastypes.NewQueryClient(queryHelper)
	s.devgasMsgServer = s.app.DevGasKeeper
	s.wasmMsgServer = wasmkeeper.NewMsgServerImpl(&s.app.WasmKeeper)
}

func (s *KeeperTestSuite) SetupSuite() {
	s.SetupTest()
}

func (s *KeeperTestSuite) FundAccount(
	ctx sdk.Context, addr sdk.AccAddress, amounts sdk.Coins,
) error {
	return testapp.FundAccount(s.app.BankKeeper, ctx, addr, amounts)
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}
