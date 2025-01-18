package precompile_test

import (
	"encoding/json"
	"fmt"
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	abibind "github.com/ethereum/go-ethereum/accounts/abi/bind"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
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

func TestUtilsSuite(t *testing.T) {
	suite.Run(t, new(UtilsSuite))
}

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
	abi := embeds.SmartContract_FunToken.ABI
	event, some := abi.Events[abiEventName]
	s.True(some, abiEventName)
	eventId := event.ID

	deps := evmtest.NewTestDeps()
	db := deps.NewStateDB()

	s.T().Log("Mint coins to generate ABCI events")
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
		EventLog gethcore.Log
	}
	wants := []Want{
		{
			EventLog: gethcore.Log{
				Address: emittingAddr,
				Topics: []gethcommon.Hash{
					eventId,
					precompile.EventTopicFromString(`coin_received`),
				},
				Data: []byte(fmt.Sprintf(
					`{"eventType":"coin_received","receiver":"%s","amount":"420000unibi"}`, evmAddrBech32),
				),
				BlockNumber: blockNumber,
				Index:       uint(dbStartIdx),
			},
		},
		{
			EventLog: gethcore.Log{
				Address: emittingAddr,
				Topics: []gethcommon.Hash{
					eventId,
					precompile.EventTopicFromString(`coinbase`),
				},
				Data: []byte(fmt.Sprintf(
					`{"eventType":"coinbase","minter":"%s","amount":"420000unibi"}`, evmAddrBech32),
				),
				BlockNumber: blockNumber,
				Index:       uint(dbStartIdx + 1),
			},
		},
	}

	s.T().Log("Define the ABI and smart contract that will unpack the event data")

	boundContract := abibind.NewBoundContract(
		emittingAddr,
		*abi,
		// verbose but descriptive to write out the interface implementations that are unused
		(abibind.ContractCaller)(nil),
		(abibind.ContractTransactor)(nil),
		(abibind.ContractFilterer)(nil),
	)

	debugBz, _ := json.MarshalIndent(abciEvents, "", "  ")
	dbLogs := db.Logs()
	for idx, want := range wants {
		gotEventLog := *dbLogs[dbStartIdx+idx]

		s.T().Log("Check event log fields")
		// logDataHex: Geth stores the bytes as a hex string (hexutil.Bytes)
		logDataHex := hexutil.Bytes(gotEventLog.Data).String()
		logDataHexDecoded, err := hexutil.Decode(logDataHex)
		s.NoErrorf(err, "logDataHex: %s")
		s.Contains(string(logDataHexDecoded), string(want.EventLog.Data))
		{
			w, g := want.EventLog.Topics, gotEventLog.Topics
			s.Require().EqualValuesf(w, g, "events:\n%#s", debugBz)
		}
		{
			w, g := want.EventLog.Address.Hex(), gotEventLog.Address.Hex()
			s.Require().EqualValuesf(w, g, "events:\n%#s", debugBz)
		}
		{
			w, g := string(want.EventLog.Data), string(gotEventLog.Data)
			s.Require().Containsf(g, w, "events:\n%#s", debugBz)
		}

		s.T().Log("Use geth/.../abi/bind Go bindings to test ABI event decoding")
		eventMap := make(map[string]any)
		err = boundContract.UnpackLogIntoMap(eventMap, abiEventName, gotEventLog)
		s.Require().NoError(err)

		abciEventValUntyped, isSome := eventMap["abciEvent"]
		s.Truef(isSome, "%+s", eventMap)
		abciEventVal, ok := abciEventValUntyped.(string)
		s.True(ok, "%+s\nttype of abciEventVal: %T", eventMap, abciEventValUntyped)
		s.Equal(string(want.EventLog.Data), string(abciEventVal), "%+s", eventMap)

		_, isSome = eventMap["eventType"]
		s.Truef(isSome, "%+s", eventMap)
	}
}
