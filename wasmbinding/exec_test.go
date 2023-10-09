package wasmbinding_test

import (
	"encoding/json"
	"testing"
	"time"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/wasmbinding/bindings"
	"github.com/NibiruChain/nibiru/wasmbinding/wasmbin"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	"github.com/NibiruChain/nibiru/x/common/testutil/genesis"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/x/oracle/types"
	perpv2types "github.com/NibiruChain/nibiru/x/perp/v2/types"
	sudokeeper "github.com/NibiruChain/nibiru/x/sudo/keeper"
	sudotypes "github.com/NibiruChain/nibiru/x/sudo/types"
)

// Keeper only used for testing, never for production
type TestOnlySudoKeeper struct {
	sudokeeper.Keeper
}

// SetSudoContracts overwrites the state. This function is a convenience
// function for testing with permissioned contracts in other modules..
func (k TestOnlySudoKeeper) SetSudoContracts(contracts []string, ctx sdk.Context) {
	k.Sudoers.Set(ctx, sudotypes.Sudoers{
		Root:      "",
		Contracts: contracts,
	})
}

func TestSuiteExecutor_RunAll(t *testing.T) {
	suite.Run(t, new(TestSuiteExecutor))
}

func DoCustomBindingExecute(
	ctx sdk.Context,
	nibiru *app.NibiruApp,
	contract sdk.AccAddress,
	sender sdk.AccAddress,
	cwMsg bindings.NibiruMsg,
	funds sdk.Coins,
) (contractRespBz []byte, err error) {
	jsonCwMsg, err := json.Marshal(cwMsg)
	if err != nil {
		return contractRespBz, err
	}

	if err := funds.Validate(); err != nil {
		return contractRespBz, err
	}

	return wasmkeeper.NewDefaultPermissionKeeper(nibiru.WasmKeeper).
		Execute(ctx, contract, sender, jsonCwMsg, funds)
}

func (s *TestSuiteExecutor) ExecuteAgainstContract(
	contract sdk.AccAddress, execMsg bindings.NibiruMsg,
) (contractRespBz []byte, err error) {
	return DoCustomBindingExecute(
		s.ctx, s.nibiru, contract, s.contractDeployer, execMsg, sdk.Coins{})
}

type TestSuiteExecutor struct {
	suite.Suite

	nibiru           *app.NibiruApp
	ctx              sdk.Context
	contractDeployer sdk.AccAddress

	keeper     TestOnlySudoKeeper
	wasmKeeper *wasmkeeper.PermissionedKeeper

	contractPerp       sdk.AccAddress
	contractController sdk.AccAddress
	contractShifter    sdk.AccAddress
	happyFields        ExampleFields
}

func (s *TestSuiteExecutor) SetupSuite() {
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
		sdk.NewCoin(denoms.NIBI, sdk.NewInt(10_000_000)),
		sdk.NewCoin(denoms.NUSD, sdk.NewInt(420_000*69)),
	)

	s.NoError(testapp.FundAccount(nibiru.BankKeeper, ctx, sender, coins))

	nibiru, ctx = SetupAllContracts(s.T(), sender, nibiru, ctx)
	s.nibiru = nibiru
	s.ctx = ctx
	s.keeper = TestOnlySudoKeeper{Keeper: s.nibiru.SudoKeeper}
	s.wasmKeeper = wasmkeeper.NewDefaultPermissionKeeper(nibiru.WasmKeeper)

	s.contractPerp = ContractMap[wasmbin.WasmKeyPerpBinding]
	s.contractController = ContractMap[wasmbin.WasmKeyController]
	s.contractShifter = ContractMap[wasmbin.WasmKeyShifter]
	s.contractController = ContractMap[wasmbin.WasmKeyController]
	s.T().Logf("contract bindings-perp: %s", s.contractPerp)
	s.T().Logf("contract shifter: %s", s.contractShifter)
	s.OnSetupEnd()
}

func (s *TestSuiteExecutor) OnSetupEnd() {
	SetExchangeRates(&s.Suite, s.nibiru, s.ctx)
}

func (s *TestSuiteExecutor) TestOpenAddRemoveClose() {
	pair := asset.MustNewPair(s.happyFields.Pair)
	margin := sdk.NewCoin(denoms.NUSD, sdk.NewInt(69))

	coins := sdk.NewCoins(
		margin.Add(sdk.NewCoin(denoms.NUSD, sdk.NewInt(1_000))),
	)
	s.NoError(testapp.FundAccount(s.nibiru.BankKeeper, s.ctx, s.contractPerp, coins))

	// TestMarketOrder (integration - real contract, real app)
	execMsg := bindings.NibiruMsg{
		MarketOrder: &bindings.MarketOrder{
			Pair:            s.happyFields.Pair,
			IsLong:          true,
			QuoteAmount:     sdk.NewInt(42),
			Leverage:        sdk.NewDec(5),
			BaseAmountLimit: sdk.ZeroInt(),
		},
	}

	s.T().Log("Executing with permission should succeed")
	s.keeper.SetSudoContracts(
		[]string{s.contractPerp.String()}, s.ctx,
	)

	contractRespBz, err := s.ExecuteAgainstContract(s.contractPerp, execMsg)
	s.NoErrorf(err, "contractRespBz: %s", contractRespBz)

	// TestAddMargin (integration - real contract, real app)
	execMsg = bindings.NibiruMsg{
		AddMargin: &bindings.AddMargin{
			Pair:   pair.String(),
			Margin: margin,
		},
	}
	contractRespBz, err = s.ExecuteAgainstContract(s.contractPerp, execMsg)
	s.NoErrorf(err, "contractRespBz: %s", contractRespBz)

	// TestRemoveMargin (integration - real contract, real app)
	execMsg = bindings.NibiruMsg{
		RemoveMargin: &bindings.RemoveMargin{
			Pair:   pair.String(),
			Margin: margin,
		},
	}
	contractRespBz, err = s.ExecuteAgainstContract(s.contractPerp, execMsg)
	s.NoErrorf(err, "contractRespBz: %s", contractRespBz)

	// TestClosePosition (integration - real contract, real app)
	execMsg = bindings.NibiruMsg{
		ClosePosition: &bindings.ClosePosition{
			Pair: pair.String(),
		},
	}
	contractRespBz, err = s.ExecuteAgainstContract(s.contractPerp, execMsg)
	s.NoErrorf(err, "contractRespBz: %s", contractRespBz)
}

func (s *TestSuiteExecutor) TestOracleParams() {
	defaultParams := types.DefaultParams()
	defaultParams.VotePeriod = 1_000
	theVotePeriod := sdk.NewInt(1234)
	execMsg := bindings.NibiruMsg{
		EditOracleParams: &bindings.EditOracleParams{
			VotePeriod: &theVotePeriod,
		},
	}

	params, err := s.nibiru.OracleKeeper.Params.Get(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(defaultParams, params)

	s.T().Log("Executing without permission should fail")
	s.keeper.SetSudoContracts(
		[]string{}, s.ctx,
	)
	contractRespBz, err := s.ExecuteAgainstContract(s.contractController, execMsg)
	s.Errorf(err, "contractRespBz: %s", contractRespBz)

	s.T().Log("Executing with permission should succeed")
	s.keeper.SetSudoContracts(
		[]string{s.contractController.String()}, s.ctx,
	)

	// VotePeriod should be updated
	theVotePeriod = sdk.NewInt(1234)
	execMsg = bindings.NibiruMsg{
		EditOracleParams: &bindings.EditOracleParams{
			VotePeriod: &theVotePeriod,
		},
	}

	contractRespBz, err = s.ExecuteAgainstContract(s.contractController, execMsg)
	s.NoErrorf(err, "contractRespBz: %s", contractRespBz)

	params, err = s.nibiru.OracleKeeper.Params.Get(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(uint64(1_234), params.VotePeriod)

	// VoteThreshold should be updated
	theVoteThreshold := sdk.NewDecWithPrec(1, 1)
	execMsg = bindings.NibiruMsg{
		EditOracleParams: &bindings.EditOracleParams{
			VoteThreshold: &theVoteThreshold,
		},
	}

	contractRespBz, err = s.ExecuteAgainstContract(s.contractController, execMsg)
	s.NoErrorf(err, "contractRespBz: %s", contractRespBz)

	params, err = s.nibiru.OracleKeeper.Params.Get(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(theVoteThreshold, params.VoteThreshold)

	// RewardBand should be updated
	theRewardBand := sdk.NewDecWithPrec(1, 1)
	execMsg = bindings.NibiruMsg{
		EditOracleParams: &bindings.EditOracleParams{
			RewardBand: &theRewardBand,
		},
	}

	contractRespBz, err = s.ExecuteAgainstContract(s.contractController, execMsg)
	s.NoErrorf(err, "contractRespBz: %s", contractRespBz)

	params, err = s.nibiru.OracleKeeper.Params.Get(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(theRewardBand, params.RewardBand)

	// Whitelist should be updated
	theWhitelist := []string{"BTC:USDC"}
	execMsg = bindings.NibiruMsg{
		EditOracleParams: &bindings.EditOracleParams{
			Whitelist: theWhitelist,
		},
	}

	contractRespBz, err = s.ExecuteAgainstContract(s.contractController, execMsg)
	s.NoErrorf(err, "contractRespBz: %s", contractRespBz)

	params, err = s.nibiru.OracleKeeper.Params.Get(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal([]asset.Pair{asset.NewPair("BTC", "USDC")}, params.Whitelist)

	// SlashFraction should be updated
	theSlashFraction := sdk.NewDecWithPrec(1, 4)
	execMsg = bindings.NibiruMsg{
		EditOracleParams: &bindings.EditOracleParams{
			SlashFraction: &theSlashFraction,
		},
	}

	contractRespBz, err = s.ExecuteAgainstContract(s.contractController, execMsg)
	s.NoErrorf(err, "contractRespBz: %s", contractRespBz)

	params, err = s.nibiru.OracleKeeper.Params.Get(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(theSlashFraction, params.SlashFraction)

	// SlashWindow should be updated
	theSlashWindow := sdk.NewInt(1234)
	execMsg = bindings.NibiruMsg{
		EditOracleParams: &bindings.EditOracleParams{
			SlashWindow: &theSlashWindow,
		},
	}

	contractRespBz, err = s.ExecuteAgainstContract(s.contractController, execMsg)
	s.NoErrorf(err, "contractRespBz: %s", contractRespBz)

	params, err = s.nibiru.OracleKeeper.Params.Get(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(uint64(1234), params.SlashWindow)

	// MinValidPerWindow should be updated
	theMinValidPerWindow := sdk.NewDecWithPrec(1, 4)
	execMsg = bindings.NibiruMsg{
		EditOracleParams: &bindings.EditOracleParams{
			MinValidPerWindow: &theMinValidPerWindow,
		},
	}

	contractRespBz, err = s.ExecuteAgainstContract(s.contractController, execMsg)
	s.NoErrorf(err, "contractRespBz: %s", contractRespBz)

	params, err = s.nibiru.OracleKeeper.Params.Get(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(theMinValidPerWindow, params.MinValidPerWindow)

	// TwapLookback should be updated
	theTwapLookback := sdk.NewInt(1234)
	execMsg = bindings.NibiruMsg{
		EditOracleParams: &bindings.EditOracleParams{
			TwapLookbackWindow: &theTwapLookback,
		},
	}

	contractRespBz, err = s.ExecuteAgainstContract(s.contractController, execMsg)
	s.NoErrorf(err, "contractRespBz: %s", contractRespBz)

	params, err = s.nibiru.OracleKeeper.Params.Get(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(time.Duration(1234), params.TwapLookbackWindow)

	// MinVoters should be updated
	theMinVoters := sdk.NewInt(1234)
	execMsg = bindings.NibiruMsg{
		EditOracleParams: &bindings.EditOracleParams{
			MinVoters: &theMinVoters,
		},
	}

	contractRespBz, err = s.ExecuteAgainstContract(s.contractController, execMsg)
	s.NoErrorf(err, "contractRespBz: %s", contractRespBz)

	params, err = s.nibiru.OracleKeeper.Params.Get(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(uint64(1234), params.MinVoters)

	// Validator Fee Ratio should be updated
	theValidatorFeeRatio := sdk.NewDecWithPrec(1, 4)
	execMsg = bindings.NibiruMsg{
		EditOracleParams: &bindings.EditOracleParams{
			ValidatorFeeRatio: &theValidatorFeeRatio,
		},
	}

	contractRespBz, err = s.ExecuteAgainstContract(s.contractController, execMsg)
	s.NoErrorf(err, "contractRespBz: %s", contractRespBz)

	params, err = s.nibiru.OracleKeeper.Params.Get(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(theValidatorFeeRatio, params.ValidatorFeeRatio)
}

func (s *TestSuiteExecutor) TestPegShift() {
	pair := asset.MustNewPair(s.happyFields.Pair)
	execMsg := bindings.NibiruMsg{
		PegShift: &bindings.PegShift{
			Pair:    pair.String(),
			PegMult: sdk.NewDec(420),
		},
	}

	s.T().Log("Executing with permission should succeed")
	contract := s.contractShifter
	s.keeper.SetSudoContracts(
		[]string{contract.String()}, s.ctx,
	)
	contractRespBz, err := s.ExecuteAgainstContract(contract, execMsg)
	s.NoErrorf(err, "contractRespBz: %s", contractRespBz)

	s.T().Log("Executing without permission should fail")
	s.keeper.SetSudoContracts(
		[]string{}, s.ctx,
	)
	contractRespBz, err = s.ExecuteAgainstContract(contract, execMsg)
	s.Errorf(err, "contractRespBz: %s", contractRespBz)

	s.T().Log("Executing the wrong contract should fail")
	contract = s.contractPerp
	s.keeper.SetSudoContracts(
		[]string{contract.String()}, s.ctx,
	)
	contractRespBz, err = s.ExecuteAgainstContract(contract, execMsg)
	s.Errorf(err, "contractRespBz: %s", contractRespBz)
	s.Contains(err.Error(), "Error parsing into type")
}

func (s *TestSuiteExecutor) TestNoOp() {
	contract := s.contractShifter
	execMsg := bindings.NibiruMsg{
		NoOp: &bindings.NoOp{},
	}
	contractRespBz, err := s.ExecuteAgainstContract(contract, execMsg)
	s.NoErrorf(err, "contractRespBz: %s", contractRespBz)
}

func (s *TestSuiteExecutor) TestDepthShift() {
	pair := asset.MustNewPair(s.happyFields.Pair)
	execMsg := bindings.NibiruMsg{
		DepthShift: &bindings.DepthShift{
			Pair:      pair.String(),
			DepthMult: sdk.NewDec(2),
		},
	}

	s.T().Log("Executing with permission should succeed")
	contract := s.contractShifter
	s.keeper.SetSudoContracts(
		[]string{contract.String()}, s.ctx,
	)
	contractRespBz, err := s.ExecuteAgainstContract(contract, execMsg)
	s.NoErrorf(err, "contractRespBz: %s", contractRespBz)

	s.T().Log("Executing without permission should fail")
	s.keeper.SetSudoContracts(
		[]string{}, s.ctx,
	)
	contractRespBz, err = s.ExecuteAgainstContract(contract, execMsg)
	s.Errorf(err, "contractRespBz: %s", contractRespBz)

	s.T().Log("Executing the wrong contract should fail")
	contract = s.contractPerp
	s.keeper.SetSudoContracts(
		[]string{contract.String()}, s.ctx,
	)
	contractRespBz, err = s.ExecuteAgainstContract(contract, execMsg)
	s.Errorf(err, "contractRespBz: %s", contractRespBz)
	s.Contains(err.Error(), "Error parsing into type")
}

func (s *TestSuiteExecutor) TestInsuranceFundWithdraw() {
	admin := s.contractDeployer.String()
	amtToWithdraw := sdk.NewInt(69)
	execMsg := bindings.NibiruMsg{
		InsuranceFundWithdraw: &bindings.InsuranceFundWithdraw{
			Amount: amtToWithdraw,
			To:     admin,
		},
	}

	s.T().Log("Executing should fail since the IF doesn't have funds")
	contract := s.contractController
	s.keeper.SetSudoContracts(
		[]string{contract.String()}, s.ctx,
	)
	contractRespBz, err := s.ExecuteAgainstContract(contract, execMsg)
	s.Errorf(err, "contractRespBz: %s", contractRespBz)

	s.T().Log("Executing without permission should fail")
	s.keeper.SetSudoContracts(
		[]string{}, s.ctx,
	)
	contractRespBz, err = s.ExecuteAgainstContract(contract, execMsg)
	s.Errorf(err, "contractRespBz: %s", contractRespBz)

	s.T().Log("Executing should work when the IF has funds")
	err = testapp.FundModuleAccount(
		s.nibiru.BankKeeper,
		s.ctx,
		perpv2types.PerpEFModuleAccount,
		sdk.NewCoins(sdk.NewCoin(denoms.NUSD, sdk.NewInt(420))),
	)
	s.NoError(err)
	s.keeper.SetSudoContracts(
		[]string{contract.String()}, s.ctx,
	)
	contractRespBz, err = s.ExecuteAgainstContract(contract, execMsg)
	s.NoErrorf(err, "contractRespBz: %s", contractRespBz)

	s.T().Log("Executing the wrong contract should fail")
	contract = s.contractPerp
	s.keeper.SetSudoContracts(
		[]string{contract.String()}, s.ctx,
	)
	contractRespBz, err = s.ExecuteAgainstContract(contract, execMsg)
	s.Errorf(err, "contractRespBz: %s", contractRespBz)
	s.Contains(err.Error(), "Error parsing into type")
}

func (s *TestSuiteExecutor) TestSetMarketEnabled() {
	// admin := s.contractDeployer.String()
	perpv2Genesis := genesis.PerpV2Genesis()
	contract := s.contractController
	var execMsg bindings.NibiruMsg

	for testIdx, market := range perpv2Genesis.Markets {
		execMsg = bindings.NibiruMsg{
			SetMarketEnabled: &bindings.SetMarketEnabled{
				Pair:    market.Pair.String(),
				Enabled: !market.Enabled,
			},
		}

		s.T().Logf("Execute - happy %v: market: %s", testIdx, market.Pair)
		s.keeper.SetSudoContracts(
			[]string{contract.String()}, s.ctx,
		)
		contractRespBz, err := s.ExecuteAgainstContract(contract, execMsg)
		s.NoErrorf(err, "contractRespBz: %s", contractRespBz)

		marketAfter, err := s.nibiru.PerpKeeperV2.GetMarket(s.ctx, market.Pair)
		s.NoError(err)
		s.Equal(!market.Enabled, marketAfter.Enabled)
	}

	s.T().Log("Executing without permission should fail")
	s.keeper.SetSudoContracts(
		[]string{}, s.ctx,
	)
	contractRespBz, err := s.ExecuteAgainstContract(contract, execMsg)
	s.Errorf(err, "contractRespBz: %s", contractRespBz)

	s.T().Log("Executing the wrong contract should fail")
	contract = s.contractPerp
	s.keeper.SetSudoContracts(
		[]string{contract.String()}, s.ctx,
	)
	contractRespBz, err = s.ExecuteAgainstContract(contract, execMsg)
	s.Errorf(err, "contractRespBz: %s", contractRespBz)
	s.Contains(err.Error(), "Error parsing into type")
}

func (s *TestSuiteExecutor) TestCreateMarket() {
	contract := s.contractController
	pair := asset.MustNewPair("bloop:blam")
	execMsg := bindings.NibiruMsg{
		CreateMarket: &bindings.CreateMarket{
			Pair:         pair.String(),
			PegMult:      sdk.NewDec(420),
			SqrtDepth:    sdk.NewDec(1_000),
			MarketParams: nil,
		},
	}

	s.T().Logf("Execute - happy: market: %s", pair)
	s.keeper.SetSudoContracts(
		[]string{contract.String()}, s.ctx,
	)
	contractRespBz, err := s.ExecuteAgainstContract(contract, execMsg)
	s.NoErrorf(err, "contractRespBz: %s", contractRespBz)

	market, err := s.nibiru.PerpKeeperV2.GetMarket(s.ctx, pair)
	s.NoError(err)
	s.NoError(market.Validate())
	s.True(market.Enabled)
	s.EqualValues(pair, market.Pair)

	s.T().Log("Executing without permission should fail")
	s.keeper.SetSudoContracts(
		[]string{}, s.ctx,
	)
	contractRespBz, err = s.ExecuteAgainstContract(contract, execMsg)
	s.Errorf(err, "contractRespBz: %s", contractRespBz)

	s.T().Log("Executing the wrong contract should fail")
	contract = s.contractPerp
	s.keeper.SetSudoContracts(
		[]string{contract.String()}, s.ctx,
	)
	contractRespBz, err = s.ExecuteAgainstContract(contract, execMsg)
	s.Errorf(err, "contractRespBz: %s", contractRespBz)
	s.Contains(err.Error(), "Error parsing into type")
}
