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
		s.ctx, s.nibiru, s.contractPerp, s.contractDeployer, execMsg, sdk.Coins{})
}

type TestSuiteExecutor struct {
	suite.Suite

	nibiru           *app.NibiruApp
	ctx              sdk.Context
	contractDeployer sdk.AccAddress

	contractPerp sdk.AccAddress
	happyFields  ExampleFields
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
		sdk.NewCoin(denoms.NIBI, sdk.NewInt(1_000)),
		sdk.NewCoin(denoms.NUSD, sdk.NewInt(420*69)),
	)

	s.NoError(testapp.FundAccount(nibiru.BankKeeper, ctx, sender, coins))

	nibiru, ctx = SetupAllContracts(s.T(), sender, nibiru, ctx)
	s.nibiru = nibiru
	s.ctx = ctx

	s.contractPerp = ContractMap[wasmbin.WasmKeyPerpBinding]
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
			QuoteAmount:     sdk.NewInt(420),
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

func (s *TestSuiteExecutor) TestPegShift() {
	pair := asset.MustNewPair(s.happyFields.Pair)
	execMsg := cw_struct.BindingMsg{
		PegShift: &cw_struct.PegShift{
			Pair:    pair.String(),
			PegMult: sdk.NewDec(420),
		},
	}
	contractRespBz, err := s.ExecuteAgainstContract(s.contractPerp, execMsg)
	s.NoErrorf(err, "contractRespBz: %s", contractRespBz)
}

func (s *TestSuiteExecutor) TestSwapInvariantMultiply() {
	pair := asset.MustNewPair(s.happyFields.Pair)
	execMsg := cw_struct.BindingMsg{
		SwapInvariantMultiply: &cw_struct.SwapInvariantMultiply{
			Pair:                    pair.String(),
			SwapInvariantMultiplier: sdk.NewDec(2),
		},
	}
	contractRespBz, err := s.ExecuteAgainstContract(s.contractPerp, execMsg)
	s.NoErrorf(err, "contractRespBz: %s", contractRespBz)
}
