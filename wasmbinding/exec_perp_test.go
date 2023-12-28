package wasmbinding_test

import (
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/wasmbinding"
	"github.com/NibiruChain/nibiru/wasmbinding/bindings"
	"github.com/NibiruChain/nibiru/wasmbinding/wasmbin"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	"github.com/NibiruChain/nibiru/x/common/testutil/genesis"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	oracletypes "github.com/NibiruChain/nibiru/x/oracle/types"
	perpv2types "github.com/NibiruChain/nibiru/x/perp/v2/types"
)

func TestSuitePerpExecutor_RunAll(t *testing.T) {
	suite.Run(t, new(TestSuitePerpExecutor))
}

type TestSuitePerpExecutor struct {
	suite.Suite

	nibiru           *app.NibiruApp
	ctx              sdk.Context
	contractDeployer sdk.AccAddress
	exec             *wasmbinding.ExecutorPerp

	contractPerp sdk.AccAddress
	ratesMap     map[asset.Pair]sdk.Dec
	happyFields  ExampleFields
}

func SetExchangeRates(
	testSuite *suite.Suite,
	nibiru *app.NibiruApp,
	ctx sdk.Context,
) (exchangeRateMap map[asset.Pair]sdk.Dec) {
	s := testSuite
	exchangeRateTuples := []oracletypes.ExchangeRateTuple{
		{
			Pair:         asset.Registry.Pair(denoms.ETH, denoms.NUSD),
			ExchangeRate: sdk.NewDec(1_000),
		},
		{
			Pair:         asset.Registry.Pair(denoms.NIBI, denoms.NUSD),
			ExchangeRate: sdk.NewDec(10),
		},
	}

	for _, exchangeRateTuple := range exchangeRateTuples {
		pair := exchangeRateTuple.Pair
		exchangeRate := exchangeRateTuple.ExchangeRate
		nibiru.OracleKeeper.SetPrice(ctx, pair, exchangeRate)

		rate, err := nibiru.OracleKeeper.ExchangeRates.Get(ctx, pair)
		s.Assert().NoError(err)
		s.Assert().EqualValues(exchangeRate, rate.ExchangeRate)
	}

	return oracletypes.ExchangeRateTuples(exchangeRateTuples).ToMap()
}

type ExampleFields struct {
	Pair   string
	Trader sdk.AccAddress
	Dec    sdk.Dec
	Int    sdkmath.Int
}

func GetHappyFields() ExampleFields {
	return ExampleFields{
		Pair:   asset.Registry.Pair(denoms.ETH, denoms.NUSD).String(),
		Trader: sdk.AccAddress([]byte("trader")),
		Dec:    sdk.NewDec(50),
		Int:    sdk.NewInt(420),
	}
}

func SetupPerpGenesis() app.GenesisState {
	genesisState := genesis.NewTestGenesisState(app.MakeEncodingConfig())
	genesisState = genesis.AddOracleGenesis(genesisState)
	genesisState = genesis.AddPerpV2Genesis(genesisState)
	return genesisState
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
		sdk.NewCoin(perpv2types.TestingCollateralDenomNUSD, sdk.NewInt(420_000*69)),
		sdk.NewCoin(denoms.USDT, sdk.NewInt(420_000*69)),
	)
	s.NoError(testapp.FundAccount(nibiru.BankKeeper, ctx, sender, coins))

	nibiru, ctx = SetupAllContracts(s.T(), sender, nibiru, ctx)
	s.nibiru = nibiru
	s.ctx = ctx
	s.contractPerp = ContractMap[wasmbin.WasmKeyPerpBinding]

	s.NoError(testapp.FundAccount(nibiru.BankKeeper, ctx, s.contractPerp, coins))
	s.exec = &wasmbinding.ExecutorPerp{
		PerpV2: nibiru.PerpKeeperV2,
	}
	s.nibiru.PerpKeeperV2.Collateral.Set(s.ctx, perpv2types.TestingCollateralDenomNUSD)
	s.NoError(testapp.FundAccount(nibiru.BankKeeper, ctx, s.contractPerp, coins))

	s.OnSetupEnd()
}

func (s *TestSuitePerpExecutor) OnSetupEnd() {
	s.contractPerp = ContractMap[wasmbin.WasmKeyPerpBinding]
	s.ratesMap = SetExchangeRates(&s.Suite, s.nibiru, s.ctx)
}

// Happy path coverage
func (s *TestSuitePerpExecutor) TestPerpExecutorHappy() {
	for _, err := range []error{
		s.DoInsuranceFundWithdrawTest(sdk.NewInt(69), s.contractDeployer),
		s.DoCreateMarketTest(asset.MustNewPair("ufoo:ubar")),
		s.DoCreateMarketTestWithParams(asset.MustNewPair("ufoo2:ubar")),
	} {
		s.NoError(err)
	}
}

func (s *TestSuitePerpExecutor) DoInsuranceFundWithdrawTest(
	amt sdkmath.Int, to sdk.AccAddress,
) error {
	cwMsg := &bindings.InsuranceFundWithdraw{
		Amount: amt,
		To:     to.String(),
	}

	err := testapp.FundModuleAccount(
		s.nibiru.BankKeeper,
		s.ctx,
		perpv2types.PerpEFModuleAccount,
		sdk.NewCoins(sdk.NewCoin(perpv2types.TestingCollateralDenomNUSD, sdk.NewInt(420))),
	)
	s.NoError(err)

	return s.exec.InsuranceFundWithdraw(cwMsg, s.ctx)
}

func (s *TestSuitePerpExecutor) DoCreateMarketTest(pair asset.Pair) error {
	cwMsg := &bindings.CreateMarket{
		Pair:         pair.String(),
		PegMult:      sdk.NewDec(2_500),
		SqrtDepth:    sdk.NewDec(1_000),
		MarketParams: nil,
	}

	return s.exec.CreateMarket(cwMsg, s.ctx)
}

func (s *TestSuitePerpExecutor) DoCreateMarketTestWithParams(pair asset.Pair) error {
	cwMsg := &bindings.CreateMarket{
		Pair:      pair.String(),
		PegMult:   sdk.NewDec(2_500),
		SqrtDepth: sdk.NewDec(1_000),
		MarketParams: &bindings.MarketParams{
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
			OraclePair:                      pair.String(),
		},
	}

	return s.exec.CreateMarket(cwMsg, s.ctx)
}

func (s *TestSuitePerpExecutor) TestSadPaths_Nil() {
	err := s.exec.InsuranceFundWithdraw(nil, s.ctx)
	s.Error(err)
}

func (s *TestSuitePerpExecutor) TestSadPath_InsuranceFundWithdraw() {
	fundsToWithdraw := sdk.NewCoin(perpv2types.TestingCollateralDenomNUSD, sdk.NewInt(69_000))

	err := s.DoInsuranceFundWithdrawTest(fundsToWithdraw.Amount, s.contractDeployer)
	s.Error(err)
}

func (s *TestSuitePerpExecutor) TestSadPaths_InvalidPair() {
	sadPair := asset.Pair("ftt:ust:doge")
	pair := sadPair

	for _, err := range []error{
		s.DoCreateMarketTest(pair),
		s.DoCreateMarketTestWithParams(pair),
	} {
		s.Error(err)
	}
}
