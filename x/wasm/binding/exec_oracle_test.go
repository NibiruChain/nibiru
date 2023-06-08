package binding_test

import (
	"testing"
	"time"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/x/wasm/binding"
	"github.com/NibiruChain/nibiru/x/wasm/binding/cw_struct"
	"github.com/NibiruChain/nibiru/x/wasm/binding/wasmbin"
)

func TestSuiteOracleExecutor_RunAll(t *testing.T) {
	suite.Run(t, new(TestSuiteOracleExecutor))
}

type TestSuiteOracleExecutor struct {
	suite.Suite

	nibiru           app.NibiruApp
	contractDeployer sdk.AccAddress
	exec             binding.ExecutorOracle
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

	wasmkeeper.NewMsgServerImpl(wasmkeeper.NewDefaultPermissionKeeper(nibiru.WasmKeeper))
	s.contract = ContractMap[wasmbin.WasmKeyController]
	s.exec = binding.ExecutorOracle{
		Oracle: nibiru.OracleKeeper,
	}
}

func (s *TestSuiteOracleExecutor) TestExecuteOracleParams() {
	period := sdk.NewInt(1234)
	cwMsg := &cw_struct.EditOracleParams{
		VotePeriod: &period,
	}

	// Vote Period
	params, err := s.nibiru.OracleKeeper.Params.Get(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(uint64(1_000), params.VotePeriod)

	err = s.exec.SetOracleParams(cwMsg, s.ctx)
	s.Require().NoError(err)

	params, err = s.nibiru.OracleKeeper.Params.Get(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(uint64(1234), params.VotePeriod)

	// Vote Threshold
	params, err = s.nibiru.OracleKeeper.Params.Get(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(sdk.OneDec().Quo(sdk.NewDec(3)), params.VoteThreshold)

	threshold := sdk.MustNewDecFromStr("0.4")
	cwMsg = &cw_struct.EditOracleParams{
		VoteThreshold: &threshold,
	}

	err = s.exec.SetOracleParams(cwMsg, s.ctx)
	s.Require().NoError(err)

	params, err = s.nibiru.OracleKeeper.Params.Get(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(threshold, params.VoteThreshold)

	// Reward Band
	params, err = s.nibiru.OracleKeeper.Params.Get(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(sdk.NewDecWithPrec(2, 2), params.RewardBand)

	band := sdk.MustNewDecFromStr("0.5")
	cwMsg = &cw_struct.EditOracleParams{
		RewardBand: &band,
	}

	err = s.exec.SetOracleParams(cwMsg, s.ctx)
	s.Require().NoError(err)

	params, err = s.nibiru.OracleKeeper.Params.Get(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(band, params.RewardBand)

	// White List
	params, err = s.nibiru.OracleKeeper.Params.Get(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(14, len(params.Whitelist))

	whitelist := []string{"aave:usdc", "sol:usdc"}
	cwMsg = &cw_struct.EditOracleParams{
		Whitelist: whitelist,
	}
	err = s.exec.SetOracleParams(cwMsg, s.ctx)
	s.Require().NoError(err)

	params, err = s.nibiru.OracleKeeper.Params.Get(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(2, len(params.Whitelist))

	// Slash Fraction
	params, err = s.nibiru.OracleKeeper.Params.Get(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(sdk.NewDecWithPrec(5, 3), params.SlashFraction)

	slashFraction := sdk.MustNewDecFromStr("0.5")
	cwMsg = &cw_struct.EditOracleParams{
		SlashFraction: &slashFraction,
	}

	err = s.exec.SetOracleParams(cwMsg, s.ctx)
	s.Require().NoError(err)

	params, err = s.nibiru.OracleKeeper.Params.Get(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(slashFraction, params.SlashFraction)

	// Slash Window
	params, err = s.nibiru.OracleKeeper.Params.Get(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(uint64(3600), params.SlashWindow)

	slashWindow := sdk.NewInt(2)
	cwMsg = &cw_struct.EditOracleParams{
		SlashWindow: &slashWindow,
	}

	err = s.exec.SetOracleParams(cwMsg, s.ctx)
	s.Require().NoError(err)

	params, err = s.nibiru.OracleKeeper.Params.Get(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(slashWindow.Uint64(), params.SlashWindow)

	// Min valid per window
	params, err = s.nibiru.OracleKeeper.Params.Get(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(sdk.NewDecWithPrec(69, 2), params.MinValidPerWindow)

	minValidPerWindow := sdk.MustNewDecFromStr("0.5")
	cwMsg = &cw_struct.EditOracleParams{
		MinValidPerWindow: &minValidPerWindow,
	}

	err = s.exec.SetOracleParams(cwMsg, s.ctx)
	s.Require().NoError(err)

	params, err = s.nibiru.OracleKeeper.Params.Get(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(minValidPerWindow, params.MinValidPerWindow)

	// Twap lookback window
	params, err = s.nibiru.OracleKeeper.Params.Get(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(time.Minute*15, params.TwapLookbackWindow)

	twapLookbackWindow := sdk.NewInt(int64(time.Second * 30))
	cwMsg = &cw_struct.EditOracleParams{
		TwapLookbackWindow: &twapLookbackWindow,
	}

	err = s.exec.SetOracleParams(cwMsg, s.ctx)
	s.Require().NoError(err)

	params, err = s.nibiru.OracleKeeper.Params.Get(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(time.Duration(twapLookbackWindow.Int64()), params.TwapLookbackWindow)

	// Min Voters
	params, err = s.nibiru.OracleKeeper.Params.Get(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(uint64(4), params.MinVoters)

	minVoters := sdk.NewInt(2)
	cwMsg = &cw_struct.EditOracleParams{
		MinVoters: &minVoters,
	}

	err = s.exec.SetOracleParams(cwMsg, s.ctx)
	s.Require().NoError(err)

	params, err = s.nibiru.OracleKeeper.Params.Get(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(minVoters.Uint64(), params.MinVoters)

	// Validator Fee Ratio
	params, err = s.nibiru.OracleKeeper.Params.Get(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(sdk.NewDecWithPrec(5, 2), params.ValidatorFeeRatio)

	validatorFeeRatio := sdk.MustNewDecFromStr("0.7")
	cwMsg = &cw_struct.EditOracleParams{
		ValidatorFeeRatio: &validatorFeeRatio,
	}

	err = s.exec.SetOracleParams(cwMsg, s.ctx)
	s.Require().NoError(err)

	params, err = s.nibiru.OracleKeeper.Params.Get(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(validatorFeeRatio, params.ValidatorFeeRatio)
}
