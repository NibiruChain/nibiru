package precompile_test

import (
	"encoding/json"
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
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

// func abci.Event -> abi.Event

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
	want := []*gethcore.Log{
		{
			Address: emittingAddr,
			Topics: []gethcommon.Hash{
				eventId,
				precompile.EventTopicFromString(`coin_received`),
			},
			Data: []byte(fmt.Sprintf(
				`{"receiver":"%s","amount":"420000unibi"}`, evmAddrBech32),
			),
			BlockNumber: blockNumber,
			Index:       uint(dbStartIdx),
		},
		{
			Address: emittingAddr,
			Topics: []gethcommon.Hash{
				eventId,
				precompile.EventTopicFromString(`coinbase`),
			},
			Data: []byte(fmt.Sprintf(
				`{"minter":"%s","amount":"420000unibi"}`, evmAddrBech32),
			),
			BlockNumber: blockNumber,
			Index:       uint(dbStartIdx + 1),
		},
	}

	debugBz, _ := json.MarshalIndent(abciEvents, "", "  ")
	for idx, wantLog := range want {
		s.EqualValuesf(
			*wantLog,
			*db.Logs()[dbStartIdx+idx],
			"events:\n%#s", debugBz,
		)
	}
}
