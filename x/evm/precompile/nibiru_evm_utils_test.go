package precompile_test

import (
	"encoding/json"
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	abibind "github.com/ethereum/go-ethereum/accounts/abi/bind"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/x/evm/precompile"
)

type UtilsSuite struct {
	suite.Suite
}

// These fields of the gethcore.Log have defaults. Look into how we populate them
// in EthereumTx.
// TODO: UD-DEBUG: Check about block hash
// TODO: UD-DEBUG: Check about tx hash
// TODO: UD-DEBUG: Check about tx index
// TODO: UD-DEBUG: Check about index
// TODO: UD-DEBUG: Check about block number

func (s *UtilsSuite) TestAttrsToJSON() {
	testCases := []struct {
		name  string
		attrs []abci.EventAttribute
		want  string
	}{
		{
			name: "repeated key - last value wins",
			attrs: []abci.EventAttribute{
				{Key: "action", Value: "first"},
				{Key: "action", Value: "second"},
				{Key: "amount", Value: "100"},
			},
			want: `{"action":"first","amount":"100"}`,
		},
		{
			name: "three unique attributes",
			attrs: []abci.EventAttribute{
				{Key: "sender", Value: "addr1"},
				{Key: "recipient", Value: "addr2"},
				{Key: "amount", Value: "150"},
			},
			want: `{"sender":"addr1","recipient":"addr2","amount":"150"}`,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			got := string(precompile.AttrsToJSON(tc.attrs))
			s.Equal(tc.want, got)
		})
	}
}

func (s *UtilsSuite) TestEmitEventAbciEvent() {
	abiEventName := precompile.EvmEventAbciEvent
	event, some := embeds.SmartContract_FunToken.ABI.Events[abiEventName]
	s.True(some, abiEventName)
	eventId := event.ID

	deps := evmtest.NewTestDeps()
	db := deps.NewStateDB()

	startIdx := len(deps.Ctx.EventManager().Events())
	dbStartIdx := len(db.Logs())
	err := deps.App.BankKeeper.MintCoins(deps.Ctx, evm.ModuleName,
		sdk.NewCoins(sdk.NewInt64Coin(evm.EVMBankDenom, 420_000)),
	)
	s.NoError(err)

	abciEvents := deps.Ctx.EventManager().Events()[startIdx:]
	s.Lenf(abciEvents, 2, "%+s", abciEvents)

	emittingAddr := precompile.PrecompileAddr_Wasm
	precompile.EmitEventAbciEvents(deps.Ctx, db, abciEvents, emittingAddr)
	blockNumber := uint64(deps.Ctx.BlockHeight())
	evmAddrBech32 := eth.EthAddrToNibiruAddr(evm.EVM_MODULE_ADDRESS)
	type Want struct {
		EventLogDataJson      func() (raw, abiForm []byte)
		EventLog              gethcore.Log
		ContractAbiParsedData map[string]any
	}
	wants := []Want{
		{
			EventLogDataJson: func() (raw, abiForm []byte) {
				dataJson := []byte(fmt.Sprintf(
					`{"eventType":"coin_received","receiver":"%s","amount":"420000unibi"}`, evmAddrBech32),
				)
				nonIndexedArgs, _ := event.Inputs.NonIndexed().Pack(dataJson)
				return dataJson, nonIndexedArgs
			},
			// Log.Data will be filled from the Want.EventLogData fn
			EventLog: gethcore.Log{
				Address: emittingAddr,
				Topics: []gethcommon.Hash{
					eventId,
					precompile.EventTopicFromString(`coin_received`),
				},
				BlockNumber: blockNumber,
				Index:       uint(dbStartIdx),
			},
			ContractAbiParsedData: map[string]any{
				"attrs": []byte(fmt.Sprintf(
					`{"eventType":"coin_received","receiver":"%s","amount":"420000unibi"}`, evmAddrBech32),
				),
			},
		},
		{
			EventLogDataJson: func() (raw, abiForm []byte) {
				dataJson := []byte(fmt.Sprintf(
					`{"eventType":"coinbase","minter":"%s","amount":"420000unibi"}`, evmAddrBech32),
				)
				nonIndexedArgs, _ := event.Inputs.NonIndexed().Pack(dataJson)
				return dataJson, nonIndexedArgs
			},
			// Log.Data will be filled from the Want.EventLogData fn
			EventLog: gethcore.Log{
				Address: emittingAddr,
				Topics: []gethcommon.Hash{
					eventId,
					precompile.EventTopicFromString(`coinbase`),
				},
				BlockNumber: blockNumber,
				Index:       uint(dbStartIdx + 1),
			},
			ContractAbiParsedData: map[string]any{
				"attrs": []byte(fmt.Sprintf(
					`{"eventType":"coinbase","minter":"%s","amount":"420000unibi"}`, evmAddrBech32),
				),
			},
		},
	}

	s.T().Log("Define the ABI and smart contract that will unpack the event data")
	abi := embeds.SmartContract_FunToken.ABI
	boundContract := abibind.NewBoundContract(
		emittingAddr,
		*abi,
		// These interface fields are not need for this test
		(abibind.ContractCaller)(nil),
		(abibind.ContractTransactor)(nil),
		(abibind.ContractFilterer)(nil),
	)

	debugBz, _ := json.MarshalIndent(abciEvents, "", "  ")
	dbLogs := db.Logs()
	for idx, want := range wants {
		_, want.EventLog.Data = want.EventLogDataJson()
		gotEventLog := *dbLogs[dbStartIdx+idx]

		s.T().Log("event log.Data must unpack according to the ABI")
		eventlogsNonIndexed := make(map[string]any)
		err = abi.UnpackIntoMap(
			eventlogsNonIndexed,
			abiEventName,
			gotEventLog.Data,
		)
		s.NoErrorf(err, "eventlogsNonIndexed: %+s", eventlogsNonIndexed)
		s.EqualValues(want.ContractAbiParsedData, eventlogsNonIndexed)

		s.T().Log("event must be unpackable by the BoundContract that emitted it.")
		gotLogMap := make(map[string]any)
		err := boundContract.UnpackLogIntoMap(gotLogMap, abiEventName, gotEventLog)
		s.NoErrorf(err, "gotLogMap: %+s", gotLogMap)

		s.EqualValuesf(
			want.EventLog,
			gotEventLog,
			"events:\n%#s", debugBz,
		)
	}

	s.Require().NoErrorf(err, "debugBz %T want %T", debugBz, wants)
}
