package binding_test

import (
	"errors"
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/collections"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	perpv2types "github.com/NibiruChain/nibiru/x/perp/v2/types"
	"github.com/NibiruChain/nibiru/x/wasm/binding"
	"github.com/NibiruChain/nibiru/x/wasm/binding/cw_struct"
	"github.com/NibiruChain/nibiru/x/wasm/binding/wasmbin"
)

func TestSuitePerpExecutor_RunAll(t *testing.T) {
	suite.Run(t, new(TestSuitePerpExecutor))
}

type TestSuitePerpExecutor struct {
	suite.Suite

	nibiru           *app.NibiruApp
	ctx              sdk.Context
	contractDeployer sdk.AccAddress
	exec             *binding.ExecutorPerp

	contractPerp sdk.AccAddress
	ratesMap     map[asset.Pair]sdk.Dec
	happyFields  ExampleFields
}

func (s *TestSuitePerpExecutor) SetupSuite() {
	s.happyFields = GetHappyFields()
	sender := testutil.AccAddress()
	s.contractDeployer = sender

	genesisState := SetupPerpGenesis()
	nibiru := testapp.NewNibiruTestApp(genesisState)
	ctx := nibiru.NewContext(false, tmproto.Header{
		Height:  1,
		ChainID: "nibiru-wasmnet-1",
		Time:    time.Now().UTC(),
	})
	coins := sdk.NewCoins(
		sdk.NewCoin(denoms.NIBI, sdk.NewInt(1_000_000)),
		sdk.NewCoin(denoms.NUSD, sdk.NewInt(420_000*69)),
		sdk.NewCoin(denoms.USDT, sdk.NewInt(420_000*69)),
	)
	s.NoError(testapp.FundAccount(nibiru.BankKeeper, ctx, sender, coins))

	nibiru, ctx = SetupAllContracts(s.T(), sender, nibiru, ctx)
	s.nibiru = nibiru
	s.ctx = ctx
	s.contractPerp = ContractMap[wasmbin.WasmKeyPerpBinding]

	s.NoError(testapp.FundAccount(nibiru.BankKeeper, ctx, s.contractPerp, coins))
	s.exec = &binding.ExecutorPerp{
		PerpV2: nibiru.PerpKeeperV2,
	}
	s.NoError(testapp.FundAccount(nibiru.BankKeeper, ctx, s.contractPerp, coins))

	s.OnSetupEnd()
}

func (s *TestSuitePerpExecutor) OnSetupEnd() {
	s.contractPerp = ContractMap[wasmbin.WasmKeyPerpBinding]
	s.ratesMap = SetExchangeRates(&s.Suite, s.nibiru, s.ctx)
}

// Happy path coverage of MarketOrder, AddMargin, RemoveMargin, and ClosePosition
func (s *TestSuitePerpExecutor) TestOpenAddRemoveClose() {
	pair := asset.MustNewPair(s.happyFields.Pair)
	margin := sdk.NewCoin(denoms.NUSD, sdk.NewInt(69))
	incorrectMargin := sdk.NewCoin(denoms.USDT, sdk.NewInt(69))

	for _, err := range []error{
		s.DoMarketOrderTest(pair),
		s.DoAddMarginTest(pair, margin),
		s.DoAddIncorrectMarginTest(pair, incorrectMargin),
		s.DoRemoveIncorrectMarginTest(pair, incorrectMargin),
		s.DoRemoveMarginTest(pair, margin),
		s.DoClosePositionTest(pair),
		s.DoPegShiftTest(pair),
		s.DoInsuranceFundWithdrawTest(sdk.NewInt(69), s.contractDeployer),
		s.DoCreateMarketTest(asset.MustNewPair("ufoo:ubar")),
		s.DoCreateMarketTestWithParams(asset.MustNewPair("ufoo2:ubar")),
	} {
		s.NoError(err)
	}
}

func (s *TestSuitePerpExecutor) DoMarketOrderTest(pair asset.Pair) error {
	cwMsg := &cw_struct.MarketOrder{
		Pair:            pair.String(),
		IsLong:          false,
		QuoteAmount:     sdk.NewInt(4_200_000),
		Leverage:        sdk.NewDec(5),
		BaseAmountLimit: sdk.ZeroInt(),
	}

	_, err := s.exec.MarketOrder(cwMsg, s.contractPerp, s.ctx)
	if err != nil {
		return err
	}

	// Verify position exists with PerpKeeper
	_, err = s.exec.PerpV2.Positions.Get(
		s.ctx, collections.Join(pair, s.contractPerp),
	)
	if err != nil {
		return err
	}

	// Verify position exists with CustomQuerier - multi-position
	bindingQuery := cw_struct.BindingQuery{
		Positions: &cw_struct.PositionsRequest{
			Trader: s.contractPerp.String(),
		},
	}
	bindingRespMulti := new(cw_struct.PositionsRequest)
	_, err = DoCustomBindingQuery(
		s.ctx, s.nibiru, s.contractPerp, bindingQuery, bindingRespMulti,
	)
	if err != nil {
		return err
	}

	// Verify position exists with CustomQuerier - single position
	bindingQuery = cw_struct.BindingQuery{
		Position: &cw_struct.PositionRequest{
			Trader: s.contractPerp.String(),
			Pair:   pair.String(),
		},
	}
	bindingResp := new(cw_struct.PositionRequest)
	_, err = DoCustomBindingQuery(
		s.ctx, s.nibiru, s.contractPerp, bindingQuery, bindingResp,
	)

	return err
}

func (s *TestSuitePerpExecutor) DoAddMarginTest(
	pair asset.Pair, margin sdk.Coin,
) error {
	cwMsg := &cw_struct.AddMargin{
		Pair:   pair.String(),
		Margin: margin,
	}

	_, err := s.exec.AddMargin(cwMsg, s.contractPerp, s.ctx)
	return err
}

func (s *TestSuitePerpExecutor) DoAddIncorrectMarginTest(
	pair asset.Pair, margin sdk.Coin,
) error {
	cwMsg := &cw_struct.AddMargin{
		Pair:   pair.String(),
		Margin: margin,
	}

	_, err := s.exec.AddMargin(cwMsg, s.contractPerp, s.ctx)
	if err == nil {
		return errors.New("incorrect margin type should have failed")
	}
	return nil
}

func (s *TestSuitePerpExecutor) DoRemoveIncorrectMarginTest(
	pair asset.Pair, margin sdk.Coin,
) error {
	cwMsg := &cw_struct.RemoveMargin{
		Pair:   pair.String(),
		Margin: margin,
	}

	_, err := s.exec.RemoveMargin(cwMsg, s.contractPerp, s.ctx)
	if err == nil {
		return errors.New("incorrect margin type should have failed")
	}
	return nil
}

func (s *TestSuitePerpExecutor) DoRemoveMarginTest(
	pair asset.Pair, margin sdk.Coin,
) error {
	cwMsg := &cw_struct.RemoveMargin{
		Pair:   pair.String(),
		Margin: margin,
	}

	_, err := s.exec.RemoveMargin(cwMsg, s.contractPerp, s.ctx)
	return err
}

func (s *TestSuitePerpExecutor) DoClosePositionTest(pair asset.Pair) error {
	cwMsg := &cw_struct.ClosePosition{
		Pair: pair.String(),
	}

	_, err := s.exec.ClosePosition(cwMsg, s.contractPerp, s.ctx)
	return err
}

func (s *TestSuitePerpExecutor) DoPegShiftTest(pair asset.Pair) error {
	contractAddr := s.contractPerp
	cwMsg := &cw_struct.PegShift{
		Pair:    pair.String(),
		PegMult: sdk.NewDec(420),
	}

	err := s.exec.PegShift(cwMsg, contractAddr, s.ctx)
	return err
}

func (s *TestSuitePerpExecutor) DoDepthShiftTest(pair asset.Pair) error {
	cwMsg := &cw_struct.DepthShift{
		Pair:      pair.String(),
		DepthMult: sdk.NewDec(420),
	}

	err := s.exec.DepthShift(cwMsg, s.ctx)
	return err
}

func (s *TestSuitePerpExecutor) DoInsuranceFundWithdrawTest(
	amt sdkmath.Int, to sdk.AccAddress,
) error {
	cwMsg := &cw_struct.InsuranceFundWithdraw{
		Amount: amt,
		To:     to.String(),
	}

	err := testapp.FundModuleAccount(
		s.nibiru.BankKeeper,
		s.ctx,
		perpv2types.PerpEFModuleAccount,
		sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(420))),
	)
	s.NoError(err)

	return s.exec.InsuranceFundWithdraw(cwMsg, s.ctx)
}

func (s *TestSuitePerpExecutor) DoCreateMarketTest(pair asset.Pair) error {
	cwMsg := &cw_struct.CreateMarket{
		Pair:         pair.String(),
		PegMult:      sdk.NewDec(2_500),
		SqrtDepth:    sdk.NewDec(1_000),
		MarketParams: nil,
	}

	return s.exec.CreateMarket(cwMsg, s.ctx)
}

func (s *TestSuitePerpExecutor) DoCreateMarketTestWithParams(pair asset.Pair) error {
	cwMsg := &cw_struct.CreateMarket{
		Pair:      pair.String(),
		PegMult:   sdk.NewDec(2_500),
		SqrtDepth: sdk.NewDec(1_000),
		MarketParams: &cw_struct.MarketParams{
			Pair:                            pair.String(),
			Enabled:                         true,
			MaintenanceMarginRatio:          sdk.OneDec(),
			MaxLeverage:                     sdk.OneDec(),
			LatestCumulativePremiumFraction: sdk.OneDec(),
			ExchangeFeeRatio:                sdk.OneDec(),
			EcosystemFundFeeRatio:           sdk.OneDec(),
			LiquidationFeeRatio:             sdk.OneDec(),
			PartialLiquidationRatio:         sdk.OneDec(),
			FundingRateEpochId:              "hi",
			MaxFundingRate:                  sdk.OneDec(),
			TwapLookbackWindow:              sdk.OneInt(),
		},
	}

	return s.exec.CreateMarket(cwMsg, s.ctx)
}

func (s *TestSuitePerpExecutor) TestSadPaths_Nil() {
	var err error

	_, err = s.exec.MarketOrder(nil, nil, s.ctx)
	s.Error(err)

	_, err = s.exec.AddMargin(nil, nil, s.ctx)
	s.Error(err)

	_, err = s.exec.RemoveMargin(nil, nil, s.ctx)
	s.Error(err)

	_, err = s.exec.ClosePosition(nil, nil, s.ctx)
	s.Error(err)

	err = s.exec.PegShift(
		nil, sdk.AccAddress([]byte("contract")), s.ctx)
	s.Error(err)

	err = s.exec.DepthShift(nil, s.ctx)
	s.Error(err)

	err = s.exec.InsuranceFundWithdraw(nil, s.ctx)
	s.Error(err)
}

func (s *TestSuitePerpExecutor) DoSetMarketEnabledTest(
	pair asset.Pair, enabled bool,
) error {
	cwMsg := &cw_struct.SetMarketEnabled{
		Pair:    pair.String(),
		Enabled: enabled,
	}
	err := s.exec.SetMarketEnabled(cwMsg, s.ctx)
	if err != nil {
		return err
	}

	market, err := s.nibiru.PerpKeeperV2.GetMarket(s.ctx, pair)
	s.NoError(err)
	s.Equal(enabled, market.Enabled)
	return err
}

func (s *TestSuitePerpExecutor) TestSadPath_InsuranceFundWithdraw() {
	fundsToWithdraw := sdk.NewCoin(denoms.NUSD, sdk.NewInt(69_000))

	err := s.DoInsuranceFundWithdrawTest(fundsToWithdraw.Amount, s.contractDeployer)
	s.Error(err)
}

func (s *TestSuitePerpExecutor) TestSadPaths_InvalidPair() {
	sadPair := asset.Pair("ftt:ust:doge")
	pair := sadPair
	margin := sdk.NewCoin(denoms.NUSD, sdk.NewInt(69))

	for _, err := range []error{
		s.DoMarketOrderTest(pair),
		s.DoAddMarginTest(pair, margin),
		s.DoRemoveMarginTest(pair, margin),
		s.DoClosePositionTest(pair),
		s.DoPegShiftTest(pair),
		s.DoDepthShiftTest(pair),
		s.DoSetMarketEnabledTest(pair, true),
		s.DoSetMarketEnabledTest(pair, false),
		s.DoCreateMarketTest(pair),
		s.DoCreateMarketTestWithParams(pair),
	} {
		s.Error(err)
	}
}
