package binding_test

import (
	"encoding/json"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	perpv2types "github.com/NibiruChain/nibiru/x/perp/types/v2"
	"github.com/NibiruChain/nibiru/x/wasm/binding/cw_struct"

	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/wasm/binding/wasmbin"
)

func TestSuiteExecutor_RunAll(t *testing.T) {
	suite.Run(t, new(TestSuiteExecutor))
}

func DoCustomBindingExecute(
	ctx sdk.Context,
	nibiru *app.NibiruApp,
	contract sdk.AccAddress,
	sender sdk.AccAddress,
	cwMsg cw_struct.BindingMsg,
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
	contract sdk.AccAddress, execMsg cw_struct.BindingMsg,
) (contractRespBz []byte, err error) {
	return DoCustomBindingExecute(
		s.ctx, s.nibiru, contract, s.contractDeployer, execMsg, sdk.Coins{})
}

type TestSuiteExecutor struct {
	suite.Suite

	nibiru           *app.NibiruApp
	ctx              sdk.Context
	contractDeployer sdk.AccAddress

	contractPerp       sdk.AccAddress
	contractController sdk.AccAddress
	contractShifter    sdk.AccAddress
	contractController sdk.AccAddress
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

	s.contractPerp = ContractMap[wasmbin.WasmKeyPerpBinding]
	s.contractController = ContractMap[wasmbin.WasmKeyController]
	s.contractShifter = ContractMap[wasmbin.WasmKeyShifter]
	s.contractController = ContractMap[wasmbin.WasmKeyController]
	s.T().Logf("contract bindings-perp: %s", s.contractPerp)
	s.T().Logf("contract shifter: %s", s.contractShifter)
	s.OnSetupEnd()
}

func (s *TestSuiteExecutor) OnSetupEnd() {
	SetExchangeRates(s.Suite, s.nibiru, s.ctx)
}

func (s *TestSuiteExecutor) TestOpenAddRemoveClose() {
	pair := asset.MustNewPair(s.happyFields.Pair)
	margin := sdk.NewCoin(denoms.NUSD, sdk.NewInt(69))
	sender := s.contractDeployer.String()

	// TestOpenPosition (integration - real contract, real app)
	execMsg := cw_struct.BindingMsg{
		OpenPosition: &cw_struct.OpenPosition{
			Sender:          sender,
			Pair:            s.happyFields.Pair,
			IsLong:          true,
			QuoteAmount:     sdk.NewInt(4_200_000),
			Leverage:        sdk.NewDec(5),
			BaseAmountLimit: sdk.NewInt(0),
		},
	}
	contractRespBz, err := s.ExecuteAgainstContract(s.contractPerp, execMsg)
	s.NoErrorf(err, "contractRespBz: %s", contractRespBz)

	// TestAddMargin (integration - real contract, real app)
	execMsg = cw_struct.BindingMsg{
		AddMargin: &cw_struct.AddMargin{
			Sender: sender,
			Pair:   pair.String(),
			Margin: margin,
		},
	}
	contractRespBz, err = s.ExecuteAgainstContract(s.contractPerp, execMsg)
	s.NoErrorf(err, "contractRespBz: %s", contractRespBz)

	// TestRemoveMargin (integration - real contract, real app)
	execMsg = cw_struct.BindingMsg{
		RemoveMargin: &cw_struct.RemoveMargin{
			Sender: sender,
			Pair:   pair.String(),
			Margin: margin,
		},
	}
	contractRespBz, err = s.ExecuteAgainstContract(s.contractPerp, execMsg)
	s.NoErrorf(err, "contractRespBz: %s", contractRespBz)

	// TestClosePosition (integration - real contract, real app)
	execMsg = cw_struct.BindingMsg{
		ClosePosition: &cw_struct.ClosePosition{
			Sender: sender,
			Pair:   pair.String(),
		},
	}
	contractRespBz, err = s.ExecuteAgainstContract(s.contractPerp, execMsg)
	s.NoErrorf(err, "contractRespBz: %s", contractRespBz)
}

func (s *TestSuiteExecutor) TestOracleParams() {
	theVotePeriod := sdk.NewInt(1234)
	execMsg := cw_struct.BindingMsg{
		EditOracleParams: &cw_struct.EditOracleParams{
			VotePeriod: &theVotePeriod,
		},
	}

	params, err := s.nibiru.OracleKeeper.Params.Get(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(uint64(1_000), params.VotePeriod)

	s.T().Log("Executing with permission should succeed")
	contract := s.contractController
	s.nibiru.SudoKeeper.SetSudoContracts(
		[]string{contract.String()}, s.ctx,
	)

	contractRespBz, err := s.ExecuteAgainstContract(s.contractController, execMsg)
	s.NoErrorf(err, "contractRespBz: %s", contractRespBz)

	params, err = s.nibiru.OracleKeeper.Params.Get(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(uint64(1_234), params.VotePeriod)

	s.T().Log("Executing without permission should fail")
	s.nibiru.SudoKeeper.SetSudoContracts(
		[]string{}, s.ctx,
	)
	contractRespBz, err = s.ExecuteAgainstContract(contract, execMsg)
	s.Errorf(err, "contractRespBz: %s", contractRespBz)
}

func (s *TestSuiteExecutor) TestPegShift() {
	pair := asset.MustNewPair(s.happyFields.Pair)
	execMsg := cw_struct.BindingMsg{
		PegShift: &cw_struct.PegShift{
			Pair:    pair.String(),
			PegMult: sdk.NewDec(420),
		},
	}

	s.T().Log("Executing with permission should succeed")
	contract := s.contractShifter
	s.nibiru.SudoKeeper.SetSudoContracts(
		[]string{contract.String()}, s.ctx,
	)
	contractRespBz, err := s.ExecuteAgainstContract(contract, execMsg)
	s.NoErrorf(err, "contractRespBz: %s", contractRespBz)

	s.T().Log("Executing without permission should fail")
	s.nibiru.SudoKeeper.SetSudoContracts(
		[]string{}, s.ctx,
	)
	contractRespBz, err = s.ExecuteAgainstContract(contract, execMsg)
	s.Errorf(err, "contractRespBz: %s", contractRespBz)

	s.T().Log("Executing the wrong contract should fail")
	contract = s.contractPerp
	s.nibiru.SudoKeeper.SetSudoContracts(
		[]string{contract.String()}, s.ctx,
	)
	contractRespBz, err = s.ExecuteAgainstContract(contract, execMsg)
	s.Errorf(err, "contractRespBz: %s", contractRespBz)
	s.Contains(err.Error(), "Error parsing into type")
}

func (s *TestSuiteExecutor) TestDepthShift() {
	pair := asset.MustNewPair(s.happyFields.Pair)
	execMsg := cw_struct.BindingMsg{
		DepthShift: &cw_struct.DepthShift{
			Pair:      pair.String(),
			DepthMult: sdk.NewDec(2),
		},
	}

	s.T().Log("Executing with permission should succeed")
	contract := s.contractShifter
	s.nibiru.SudoKeeper.SetSudoContracts(
		[]string{contract.String()}, s.ctx,
	)
	contractRespBz, err := s.ExecuteAgainstContract(contract, execMsg)
	s.NoErrorf(err, "contractRespBz: %s", contractRespBz)

	s.T().Log("Executing without permission should fail")
	s.nibiru.SudoKeeper.SetSudoContracts(
		[]string{}, s.ctx,
	)
	contractRespBz, err = s.ExecuteAgainstContract(contract, execMsg)
	s.Errorf(err, "contractRespBz: %s", contractRespBz)

	s.T().Log("Executing the wrong contract should fail")
	contract = s.contractPerp
	s.nibiru.SudoKeeper.SetSudoContracts(
		[]string{contract.String()}, s.ctx,
	)
	contractRespBz, err = s.ExecuteAgainstContract(contract, execMsg)
	s.Errorf(err, "contractRespBz: %s", contractRespBz)
	s.Contains(err.Error(), "Error parsing into type")
}

func (s *TestSuiteExecutor) TestInsuranceFundWithdraw() {
	admin := s.contractDeployer.String()
	amtToWithdraw := sdk.NewInt(69)
	execMsg := cw_struct.BindingMsg{
		InsuranceFundWithdraw: &cw_struct.InsuranceFundWithdraw{
			Amount: amtToWithdraw,
			To:     admin,
		},
	}

	s.T().Log("Executing should fail since the IF doesn't have funds")
	contract := s.contractController
	s.nibiru.SudoKeeper.SetSudoContracts(
		[]string{contract.String()}, s.ctx,
	)
	contractRespBz, err := s.ExecuteAgainstContract(contract, execMsg)
	s.Errorf(err, "contractRespBz: %s", contractRespBz)

	s.T().Log("Executing without permission should fail")
	s.nibiru.SudoKeeper.SetSudoContracts(
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
	s.nibiru.SudoKeeper.SetSudoContracts(
		[]string{contract.String()}, s.ctx,
	)
	contractRespBz, err = s.ExecuteAgainstContract(contract, execMsg)
	s.NoErrorf(err, "contractRespBz: %s", contractRespBz)

	s.T().Log("Executing the wrong contract should fail")
	contract = s.contractPerp
	s.nibiru.SudoKeeper.SetSudoContracts(
		[]string{contract.String()}, s.ctx,
	)
	contractRespBz, err = s.ExecuteAgainstContract(contract, execMsg)
	s.Errorf(err, "contractRespBz: %s", contractRespBz)
	s.Contains(err.Error(), "Error parsing into type")
}
