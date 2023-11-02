package wasmbinding_test

import (
	"testing"
	"time"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/wasmbinding"
	"github.com/NibiruChain/nibiru/wasmbinding/bindings"
	"github.com/NibiruChain/nibiru/wasmbinding/wasmbin"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
)

func TestSuiteOracleExecutor_RunAll(t *testing.T) {
	suite.Run(t, new(TestSuiteOracleExecutor))
}

type TestSuiteOracleExecutor struct {
	suite.Suite

	nibiru           app.NibiruApp
	contractDeployer sdk.AccAddress
	exec             wasmbinding.ExecutorOracle
	contract         sdk.AccAddress
	ctx              sdk.Context
}

func (s *TestSuiteOracleExecutor) SetupSuite() {
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
		sdk.NewCoin(denoms.NIBI, sdk.TokensFromConsensusPower(1, sdk.DefaultPowerReduction)),
		sdk.NewCoin(denoms.NUSD, sdk.TokensFromConsensusPower(1, sdk.DefaultPowerReduction)),
	)
	s.NoError(testapp.FundAccount(nibiru.BankKeeper, ctx, sender, coins))

	nibiru, ctx = SetupAllContracts(s.T(), sender, nibiru, ctx)
	s.nibiru = *nibiru
	s.ctx = ctx

	wasmkeeper.NewMsgServerImpl(&nibiru.WasmKeeper)
	s.contract = ContractMap[wasmbin.WasmKeyController]
	s.exec = wasmbinding.ExecutorOracle{
		Oracle: nibiru.OracleKeeper,
	}
}

func (s *TestSuiteOracleExecutor) TestExecuteOracleParams() {
	period := sdk.NewInt(1234)
	cwMsg := &bindings.EditOracleParams{
		VotePeriod: &period,
	}

	// Vote Period
	params, err := s.nibiru.OracleKeeper.Params.Get(s.ctx)
	s.Require().NoError(err)
	s.Equal(uint64(1_000), params.VotePeriod)

	err = s.exec.SetOracleParams(cwMsg, s.ctx)
	s.Require().NoError(err)

	params, err = s.nibiru.OracleKeeper.Params.Get(s.ctx)
	s.Require().NoError(err)
	s.Equal(uint64(1234), params.VotePeriod)

	// Vote Threshold
	params, err = s.nibiru.OracleKeeper.Params.Get(s.ctx)
	s.Require().NoError(err)
	s.Equal(sdk.OneDec().Quo(sdk.NewDec(3)), params.VoteThreshold)

	threshold := sdk.MustNewDecFromStr("0.4")
	cwMsg = &bindings.EditOracleParams{
		VoteThreshold: &threshold,
	}

	err = s.exec.SetOracleParams(cwMsg, s.ctx)
	s.Require().NoError(err)

	params, err = s.nibiru.OracleKeeper.Params.Get(s.ctx)
	s.Require().NoError(err)
	s.Equal(threshold, params.VoteThreshold)

	// Reward Band
	params, err = s.nibiru.OracleKeeper.Params.Get(s.ctx)
	s.Require().NoError(err)
	s.Equal(sdk.NewDecWithPrec(2, 2), params.RewardBand)

	band := sdk.MustNewDecFromStr("0.5")
	cwMsg = &bindings.EditOracleParams{
		RewardBand: &band,
	}

	err = s.exec.SetOracleParams(cwMsg, s.ctx)
	s.Require().NoError(err)

	params, err = s.nibiru.OracleKeeper.Params.Get(s.ctx)
	s.Require().NoError(err)
	s.Equal(band, params.RewardBand)

	// White List
	params, err = s.nibiru.OracleKeeper.Params.Get(s.ctx)
	s.Require().NoError(err)
	s.Assert().Equal(6, len(params.Whitelist))

	whitelist := []string{"aave:usdc", "sol:usdc"}
	cwMsg = &bindings.EditOracleParams{
		Whitelist: whitelist,
	}
	err = s.exec.SetOracleParams(cwMsg, s.ctx)
	s.Require().NoError(err)

	params, err = s.nibiru.OracleKeeper.Params.Get(s.ctx)
	s.Require().NoError(err)
	s.Equal(2, len(params.Whitelist))

	// Slash Fraction
	params, err = s.nibiru.OracleKeeper.Params.Get(s.ctx)
	s.Require().NoError(err)
	s.Equal(sdk.NewDecWithPrec(5, 3), params.SlashFraction)

	slashFraction := sdk.MustNewDecFromStr("0.5")
	cwMsg = &bindings.EditOracleParams{
		SlashFraction: &slashFraction,
	}

	err = s.exec.SetOracleParams(cwMsg, s.ctx)
	s.Require().NoError(err)

	params, err = s.nibiru.OracleKeeper.Params.Get(s.ctx)
	s.Require().NoError(err)
	s.Equal(slashFraction, params.SlashFraction)

	// Slash Window
	params, err = s.nibiru.OracleKeeper.Params.Get(s.ctx)
	s.Require().NoError(err)
	s.Equal(uint64(3600), params.SlashWindow)

	slashWindow := sdk.NewInt(2)
	cwMsg = &bindings.EditOracleParams{
		SlashWindow: &slashWindow,
	}

	err = s.exec.SetOracleParams(cwMsg, s.ctx)
	s.Require().NoError(err)

	params, err = s.nibiru.OracleKeeper.Params.Get(s.ctx)
	s.Require().NoError(err)
	s.Equal(slashWindow.Uint64(), params.SlashWindow)

	// Min valid per window
	params, err = s.nibiru.OracleKeeper.Params.Get(s.ctx)
	s.Require().NoError(err)
	s.Equal(sdk.NewDecWithPrec(69, 2), params.MinValidPerWindow)

	minValidPerWindow := sdk.MustNewDecFromStr("0.5")
	cwMsg = &bindings.EditOracleParams{
		MinValidPerWindow: &minValidPerWindow,
	}

	err = s.exec.SetOracleParams(cwMsg, s.ctx)
	s.Require().NoError(err)

	params, err = s.nibiru.OracleKeeper.Params.Get(s.ctx)
	s.Require().NoError(err)
	s.Equal(minValidPerWindow, params.MinValidPerWindow)

	// Twap lookback window
	params, err = s.nibiru.OracleKeeper.Params.Get(s.ctx)
	s.Require().NoError(err)
	s.Equal(time.Minute*15, params.TwapLookbackWindow)

	twapLookbackWindow := sdk.NewInt(int64(time.Second * 30))
	cwMsg = &bindings.EditOracleParams{
		TwapLookbackWindow: &twapLookbackWindow,
	}

	err = s.exec.SetOracleParams(cwMsg, s.ctx)
	s.Require().NoError(err)

	params, err = s.nibiru.OracleKeeper.Params.Get(s.ctx)
	s.Require().NoError(err)
	s.Equal(time.Duration(twapLookbackWindow.Int64()), params.TwapLookbackWindow)

	// Min Voters
	params, err = s.nibiru.OracleKeeper.Params.Get(s.ctx)
	s.Require().NoError(err)
	s.Equal(uint64(4), params.MinVoters)

	minVoters := sdk.NewInt(2)
	cwMsg = &bindings.EditOracleParams{
		MinVoters: &minVoters,
	}

	err = s.exec.SetOracleParams(cwMsg, s.ctx)
	s.Require().NoError(err)

	params, err = s.nibiru.OracleKeeper.Params.Get(s.ctx)
	s.Require().NoError(err)
	s.Equal(minVoters.Uint64(), params.MinVoters)

	// Validator Fee Ratio
	params, err = s.nibiru.OracleKeeper.Params.Get(s.ctx)
	s.Require().NoError(err)
	s.Equal(sdk.NewDecWithPrec(5, 2), params.ValidatorFeeRatio)

	validatorFeeRatio := sdk.MustNewDecFromStr("0.7")
	cwMsg = &bindings.EditOracleParams{
		ValidatorFeeRatio: &validatorFeeRatio,
	}

	err = s.exec.SetOracleParams(cwMsg, s.ctx)
	s.Require().NoError(err)

	params, err = s.nibiru.OracleKeeper.Params.Get(s.ctx)
	s.Require().NoError(err)
	s.Equal(validatorFeeRatio, params.ValidatorFeeRatio)
}
