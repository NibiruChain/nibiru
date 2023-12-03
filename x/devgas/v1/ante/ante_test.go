package ante_test

import (
	"fmt"
	"strings"
	"testing"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdkclienttx "github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/stretchr/testify/suite"

	sdk "github.com/cosmos/cosmos-sdk/types"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	devgasante "github.com/NibiruChain/nibiru/x/devgas/v1/ante"
	devgastypes "github.com/NibiruChain/nibiru/x/devgas/v1/types"
)

type AnteTestSuite struct {
	suite.Suite
}

func TestAnteSuite(t *testing.T) {
	testapp.EnsureNibiruPrefix()
	suite.Run(t, new(AnteTestSuite))
}

func (suite *AnteTestSuite) TestFeeLogic() {
	// We expect all to pass
	feeCoins := sdk.NewCoins(sdk.NewCoin("unibi", sdk.NewInt(500)), sdk.NewCoin("utoken", sdk.NewInt(250)))

	testCases := []struct {
		name               string
		incomingFee        sdk.Coins
		govPercent         sdk.Dec
		numContracts       int
		expectedFeePayment sdk.Coins
	}{
		{
			"100% fee / 1 contract",
			feeCoins,
			sdk.NewDecWithPrec(100, 2),
			1,
			sdk.NewCoins(sdk.NewCoin("unibi", sdk.NewInt(500)), sdk.NewCoin("utoken", sdk.NewInt(250))),
		},
		{
			"100% fee / 2 contracts",
			feeCoins,
			sdk.NewDecWithPrec(100, 2),
			2,
			sdk.NewCoins(sdk.NewCoin("unibi", sdk.NewInt(250)), sdk.NewCoin("utoken", sdk.NewInt(125))),
		},
		{
			"100% fee / 10 contracts",
			feeCoins,
			sdk.NewDecWithPrec(100, 2),
			10,
			sdk.NewCoins(sdk.NewCoin("unibi", sdk.NewInt(50)), sdk.NewCoin("utoken", sdk.NewInt(25))),
		},
		{
			"67% fee / 7 contracts",
			feeCoins,
			sdk.NewDecWithPrec(67, 2),
			7,
			sdk.NewCoins(sdk.NewCoin("unibi", sdk.NewInt(48)), sdk.NewCoin("utoken", sdk.NewInt(24))),
		},
		{
			"50% fee / 1 contracts",
			feeCoins,
			sdk.NewDecWithPrec(50, 2),
			1,
			sdk.NewCoins(sdk.NewCoin("unibi", sdk.NewInt(250)), sdk.NewCoin("utoken", sdk.NewInt(125))),
		},
		{
			"50% fee / 2 contracts",
			feeCoins,
			sdk.NewDecWithPrec(50, 2),
			2,
			sdk.NewCoins(sdk.NewCoin("unibi", sdk.NewInt(125)), sdk.NewCoin("utoken", sdk.NewInt(62))),
		},
		{
			"50% fee / 3 contracts",
			feeCoins,
			sdk.NewDecWithPrec(50, 2),
			3,
			sdk.NewCoins(sdk.NewCoin("unibi", sdk.NewInt(83)), sdk.NewCoin("utoken", sdk.NewInt(42))),
		},
		{
			"25% fee / 2 contracts",
			feeCoins,
			sdk.NewDecWithPrec(25, 2),
			2,
			sdk.NewCoins(sdk.NewCoin("unibi", sdk.NewInt(62)), sdk.NewCoin("utoken", sdk.NewInt(31))),
		},
		{
			"15% fee / 3 contracts",
			feeCoins,
			sdk.NewDecWithPrec(15, 2),
			3,
			sdk.NewCoins(sdk.NewCoin("unibi", sdk.NewInt(25)), sdk.NewCoin("utoken", sdk.NewInt(12))),
		},
		{
			"1% fee / 2 contracts",
			feeCoins,
			sdk.NewDecWithPrec(1, 2),
			2,
			sdk.NewCoins(sdk.NewCoin("unibi", sdk.NewInt(2)), sdk.NewCoin("utoken", sdk.NewInt(1))),
		},
	}

	for _, tc := range testCases {
		coins := devgasante.FeePayLogic(tc.incomingFee, tc.govPercent, tc.numContracts)

		for _, coin := range coins {
			for _, expectedCoin := range tc.expectedFeePayment {
				if coin.Denom == expectedCoin.Denom {
					suite.Require().Equal(expectedCoin.Amount.Int64(), coin.Amount.Int64(), tc.name)
				}
			}
		}
	}
}

func (suite *AnteTestSuite) TestDevGasPayout() {
	txGasCoins := sdk.NewCoins(
		sdk.NewCoin("unibi", sdk.NewInt(1_000)),
		sdk.NewCoin("utoken", sdk.NewInt(500)),
	)

	_, addrs := testutil.PrivKeyAddressPairs(11)
	contracts := addrs[:5]
	withdrawAddrs := addrs[5:10]
	deployerAddr := addrs[10]
	wasmExecMsgs := []*wasmtypes.MsgExecuteContract{
		{Contract: contracts[0].String()},
		{Contract: contracts[1].String()},
		{Contract: contracts[2].String()},
		{Contract: contracts[3].String()},
		{Contract: contracts[4].String()},
	}
	devGasForWithdrawer := func(
		contractIdx int, withdrawerIdx int,
	) devgastypes.FeeShare {
		return devgastypes.FeeShare{
			ContractAddress:   contracts[contractIdx].String(),
			DeployerAddress:   deployerAddr.String(),
			WithdrawerAddress: withdrawAddrs[withdrawerIdx].String(),
		}
	}

	testCases := []struct {
		name                    string
		devGasState             []devgastypes.FeeShare
		wantWithdrawerRoyalties sdk.Coins
		wantErr                 bool
		setup                   func() (*app.NibiruApp, sdk.Context)
	}{
		{
			name: "1 contract, 1 exec, 1 withdrawer",
			devGasState: []devgastypes.FeeShare{
				devGasForWithdrawer(0, 0),
			},
			// The expected royalty is gas / num_withdrawers / 2. Thus, We
			// divide gas by (num_withdrawers * 2). The 2 comes from 50% split.
			// wantWithdrawerRoyalties: num_withdrawers * 2 = 2
			wantWithdrawerRoyalties: txGasCoins.QuoInt(sdk.NewInt(2)),
			wantErr:                 false,
			setup: func() (*app.NibiruApp, sdk.Context) {
				bapp, ctx := testapp.NewNibiruTestAppAndContext()
				err := testapp.FundModuleAccount(
					bapp.BankKeeper, ctx, authtypes.FeeCollectorName, txGasCoins)
				suite.NoError(err)
				return bapp, ctx
			},
		},
		{
			name: "1 contract, 4 exec, 2 withdrawer",
			devGasState: []devgastypes.FeeShare{
				devGasForWithdrawer(0, 0),
				devGasForWithdrawer(1, 0),
				devGasForWithdrawer(2, 1),
				devGasForWithdrawer(3, 1),
			},
			// The expected royalty is gas / num_withdrawers / 2. Thus, We
			// divide gas by (num_withdrawers * 2). The 2 comes from 50% split.
			// wantWithdrawerRoyalties: num_withdrawers * 2 = 4
			wantWithdrawerRoyalties: txGasCoins.QuoInt(sdk.NewInt(4)),
			wantErr:                 false,
			setup: func() (*app.NibiruApp, sdk.Context) {
				bapp, ctx := testapp.NewNibiruTestAppAndContext()
				err := testapp.FundModuleAccount(
					bapp.BankKeeper, ctx, authtypes.FeeCollectorName, txGasCoins)
				suite.NoError(err)
				return bapp, ctx
			},
		},
		{
			name: "err: empty fee collector module account",
			devGasState: []devgastypes.FeeShare{
				devGasForWithdrawer(0, 0),
			},
			// The expected royalty is gas / num_withdrawers / 2. Thus, We
			// divide gas by (num_withdrawers * 2). The 2 comes from 50% split.
			// wantWithdrawerRoyalties: num_withdrawers * 2 = 2
			wantWithdrawerRoyalties: txGasCoins.QuoInt(sdk.NewInt(2)),
			wantErr:                 true,
			setup: func() (*app.NibiruApp, sdk.Context) {
				bapp, ctx := testapp.NewNibiruTestAppAndContext()
				return bapp, ctx
			},
		},
		{
			name:        "happy: no registered dev gas contracts",
			devGasState: []devgastypes.FeeShare{},
			// The expected royalty is gas / num_withdrawers / 2. Thus, We
			// divide gas by (num_withdrawers * 2). The 2 comes from 50% split.
			// wantWithdrawerRoyalties: num_withdrawers * 2 = 2
			wantWithdrawerRoyalties: txGasCoins.QuoInt(sdk.NewInt(2)),
			wantErr:                 false,
			setup: func() (*app.NibiruApp, sdk.Context) {
				bapp, ctx := testapp.NewNibiruTestAppAndContext()
				return bapp, ctx
			},
		},
	}

	var nextMockAnteHandler sdk.AnteHandler = func(
		ctx sdk.Context, tx sdk.Tx, simulate bool,
	) (newCtx sdk.Context, err error) {
		return ctx, nil
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			bapp, ctx := tc.setup()
			ctx = ctx.WithChainID("mock-chain-id")
			anteDecorator := devgasante.NewDevGasPayoutDecorator(
				bapp.BankKeeper, bapp.DevGasKeeper,
			)

			t.Log("set dev gas state based on test case")
			for _, devGas := range tc.devGasState {
				bapp.DevGasKeeper.SetFeeShare(ctx, devGas)
			}

			t.Log("build tx and call AnteHandle")
			encCfg := app.MakeEncodingConfig()
			txMsgs := []sdk.Msg{}
			for _, wasmExecMsg := range wasmExecMsgs {
				txMsgs = append(txMsgs, wasmExecMsg)
			}
			txBuilder, err := sdkclienttx.Factory{}.
				WithFees(txGasCoins.String()).
				WithChainID(ctx.ChainID()).
				WithTxConfig(encCfg.TxConfig).
				BuildUnsignedTx(txMsgs...)
			suite.NoError(err)
			tx := txBuilder.GetTx()
			simulate := true
			ctx, err = anteDecorator.AnteHandle(
				ctx, tx, simulate, nextMockAnteHandler,
			)
			if tc.wantErr {
				suite.Error(err)
				return
			}
			suite.NoError(err)

			t.Log("tc withdrawers should have the expected funds")
			for _, devGas := range tc.devGasState {
				withdrawerCoins := bapp.BankKeeper.SpendableCoins(
					ctx, devGas.GetWithdrawerAddr(),
				)
				wantWithdrawerRoyalties := tc.wantWithdrawerRoyalties.Sub(
					sdk.NewInt64Coin(txGasCoins[0].Denom, 1),
					sdk.NewInt64Coin(txGasCoins[1].Denom, 1),
				)
				suite.True(
					withdrawerCoins.IsAllGTE(wantWithdrawerRoyalties),
					strings.Join([]string{
						fmt.Sprintf("withdrawerCoins: %v\n", withdrawerCoins),
						fmt.Sprintf("tc.wantWithdrawerRoyalties: %v\n", tc.wantWithdrawerRoyalties),
					}, " "),
				)
			}
		})
	}
}
