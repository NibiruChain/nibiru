package binding_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/common/testutil"
	"github.com/NibiruChain/nibiru/x/common/testutil/genesis"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	"github.com/NibiruChain/nibiru/x/wasm/binding/cw_struct"
	"github.com/NibiruChain/nibiru/x/wasm/binding/wasmbin"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/suite"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
)

func TestSuite(t *testing.T) {
	suite.Run(t, new(QuerierTestSuite))
}

type WasmRequest struct {
	Request wasmvmtypes.QueryRequest `json:"request"`
}

type WasmResponse struct {
	Data []byte `json:"data"`
}

func DoCustomBindingQuery(
	ctx sdk.Context,
	nibiru *app.NibiruApp,
	contract sdk.AccAddress,
	bindingRequest cw_struct.BindingQuery,
	responsePointer interface{},
) error {
	// Parse query type compatible with wasm vm
	reqJsonBz, err := json.Marshal(bindingRequest)
	if err != nil {
		return err
	}

	// Query the smart contract
	var contractRespBz []byte

	var originalError error
	if err := common.TryCatch(func() {
		// The WasmVM tends to panic pretty easily with "Wasmer runtimer error".
		// TryCatch here makes it more safe and easy to debug.
		bz, err := nibiru.WasmKeeper.QuerySmart(
			ctx, contract, reqJsonBz,
		)
		if err != nil {
			originalError = err
		} else {
			contractRespBz = bz
		}
	})(); err != nil {
		return errors.Wrapf(err, "contractRespBz: %s", contractRespBz)
	}

	if originalError != nil {
		return originalError // the error raised if the WasmVM doesn't panic
	}

	// Parse the response data into the response pointer
	err = json.Unmarshal(contractRespBz, responsePointer)
	if err != nil {
		return errors.Wrapf(err, "responsePointer: %s", responsePointer)
	}

	return nil
}

type QuerierTestSuite struct {
	suite.Suite

	nibiru           *app.NibiruApp
	ctx              sdk.Context
	contractDeployer sdk.AccAddress

	contractPerp sdk.AccAddress
	fields       ExampleFields
}

type ExampleFields struct {
	Pair   string
	Trader sdk.AccAddress
	Dec    sdk.Dec
	Int    sdk.Int
	Market cw_struct.Market
}

func (s *QuerierTestSuite) GetHappyFields() ExampleFields {

	fields := ExampleFields{
		Pair:   asset.Registry.Pair(denoms.ETH, denoms.NUSD).String(),
		Trader: sdk.AccAddress([]byte("trader")),
		Dec:    sdk.NewDec(50),
		Int:    sdk.NewInt(420),
	}

	fields.Market = cw_struct.Market{
		Pair:         fields.Pair,
		BaseReserve:  fields.Dec,
		QuoteReserve: fields.Dec,
		SqrtDepth:    fields.Dec,
		Depth:        fields.Int,
		Bias:         fields.Dec,
		PegMult:      fields.Dec,
		Config: cw_struct.MarketConfig{
			TradeLimitRatio:        fields.Dec,
			FluctLimitRatio:        fields.Dec,
			MaxOracleSpreadRatio:   fields.Dec,
			MaintenanceMarginRatio: fields.Dec,
			MaxLeverage:            fields.Dec,
		},
		MarkPrice:   fields.Dec,
		IndexPrice:  fields.Dec.String(),
		TwapMark:    fields.Dec.String(),
		BlockNumber: 100,
	}
	return fields
}

func (s *QuerierTestSuite) SetupPerpGenesis() app.GenesisState {
	genesisState := genesis.NewTestGenesisState()
	genesisState = genesis.AddPerpGenesis(genesisState)
	genesisState = genesis.AddOracleGenesis(genesisState)
	return genesisState
}

func (s *QuerierTestSuite) SetupSuite() {
	s.fields = s.GetHappyFields()
	sender := testutil.AccAddress()
	s.contractDeployer = sender

	genesisState := s.SetupPerpGenesis()
	nibiru := testapp.NewNibiruTestApp(genesisState)
	ctx := nibiru.NewContext(false, tmproto.Header{
		Height:  1,
		ChainID: "nibiru-wasmnet-1",
		Time:    time.Now().UTC(),
	})
	coins := sdk.NewCoins(
		sdk.NewCoin(denoms.NIBI, sdk.NewInt(1_000)),
		sdk.NewCoin(denoms.NUSD, sdk.NewInt(420)),
	)
	testapp.FundAccount(
		nibiru.BankKeeper, ctx, sender, coins,
	)

	nibiru, ctx = SetupAllContracts(s.T(), sender, nibiru, ctx)
	s.nibiru = nibiru
	s.ctx = ctx

	s.contractPerp = ContractMap[wasmbin.WasmKeyPerpBinding]
}

func (s *QuerierTestSuite) TestQueryReserves() {
	testCases := map[string]struct {
		pairStr   string
		wasmError bool
	}{
		"happy":                   {pairStr: s.fields.Pair, wasmError: false},
		"sad - non existent pair": {pairStr: "fxs:ust", wasmError: true},
	}

	for name, testCase := range testCases {
		s.T().Run(name, func(t *testing.T) {
			pairStr := testCase.pairStr
			bindingQuery := cw_struct.BindingQuery{
				Reserves: &cw_struct.ReservesRequest{Pair: pairStr},
			}
			bindingResp := new(cw_struct.ReservesResponse)

			if testCase.wasmError {
				err := DoCustomBindingQuery(
					s.ctx, s.nibiru, s.contractPerp,
					bindingQuery, bindingResp,
				)
				s.Assert().Contains(err.Error(), "Error calling the VM")
				s.Assert().Contains(err.Error(), "Wasmer runtime error")
				return
			}

			s.Require().NoError(
				DoCustomBindingQuery(
					s.ctx, s.nibiru, s.contractPerp, bindingQuery, bindingResp,
				))

			wantPair := asset.MustNewPair(pairStr)
			s.Assert().EqualValues(bindingResp.Pair, wantPair)
			s.Assert().EqualValues(
				bindingResp.BaseReserve.String(),
				genesis.START_MARKETS[wantPair].BaseAssetReserve.String())
			s.Assert().EqualValues(
				bindingResp.QuoteReserve.String(),
				genesis.START_MARKETS[wantPair].QuoteAssetReserve.String())
		})
	}
}
