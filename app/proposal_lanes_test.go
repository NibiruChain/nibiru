package app

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	"github.com/NibiruChain/nibiru/v2/x/sudo"
	wasm "github.com/NibiruChain/nibiru/v2/x/wasm/types"
)

// TestIsOraclePriceWriteMsg pins the JSON shapes accepted as oracle price
// writes: exactly one top-level key, `set_prices` or `set_price`, and nothing
// else in the message.
func TestIsOraclePriceWriteMsg(t *testing.T) {
	for _, tc := range []struct {
		name string
		msg  string
		want bool
	}{
		{"set_prices", `{"set_prices":{"prices":[[0,"1.5"]]}}`, true},
		{"set_price", `{"set_price":{"token_id":0,"price_usd":"1.5"}}`, true},
		{"whitespace", " { \"set_prices\" : { } } \n", true},
		{"non-price variant", `{"update_expiration_time":{"expiration_time":1}}`, false},
		{"extra key", `{"set_prices":{},"set_price":{}}`, false},
		{"price key second", `{"create_token":{},"set_prices":{}}`, false},
		{"trailing data", `{"set_prices":{}}{}`, false},
		{"truncated", `{"set_prices":{"prices":[]}`, false},
		{"camel case", `{"setPrices":{}}`, false},
		{"nested", `{"wrapper":{"set_prices":{}}}`, false},
		{"array", `["set_prices"]`, false},
		{"string", `"set_prices"`, false},
		{"empty object", `{}`, false},
		{"empty", ``, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, isOraclePriceWriteMsg([]byte(tc.msg)))
		})
	}
}

// TestClassifyProposalTxPriceWriteSigners proves a tx is an oracle price write
// only when its sender is its only tx-level signer. A price write with a
// distinct fee payer, or a batch of price writes from two senders, is a
// zero-gas actor tx but not a price write, so it lands in the zero-gas lane.
// A fee payer equal to the sender adds no signer and keeps the oracle lane.
func TestClassifyProposalTxPriceWriteSigners(t *testing.T) {
	newAddr := func(seed byte) sdk.AccAddress {
		return sdk.AccAddress(bytes.Repeat([]byte{seed}, 20))
	}
	feeder, otherFeeder, payer := newAddr(1), newAddr(2), newAddr(3)
	oracle := newAddr(4).String()
	zeroGasActors := sudo.ZeroGasActors{
		Senders:   []string{feeder.String(), otherFeeder.String()},
		Contracts: []string{oracle},
	}
	priceWrite := func(sender sdk.AccAddress) sdk.Msg {
		return &wasm.MsgExecuteContract{
			Sender:   sender.String(),
			Contract: oracle,
			Msg:      wasm.RawContractMessage(`{"set_prices":{"prices":[]}}`),
		}
	}
	build := func(feePayer sdk.AccAddress, msgs ...sdk.Msg) sdk.Tx {
		builder := MakeEncodingConfig().TxConfig.NewTxBuilder()
		require.NoError(t, builder.SetMsgs(msgs...))
		if feePayer != nil {
			builder.SetFeePayer(feePayer)
		}
		return builder.GetTx()
	}

	for _, tc := range []struct {
		name string
		tx   sdk.Tx
		want proposalLane
	}{
		{"sole signer", build(nil, priceWrite(feeder)), proposalLaneOracle},
		{"batch from one sender",
			build(nil, priceWrite(feeder), priceWrite(feeder)), proposalLaneOracle},
		{"fee payer is the sender",
			build(feeder, priceWrite(feeder)), proposalLaneOracle},
		{"distinct fee payer",
			build(payer, priceWrite(feeder)), proposalLaneZeroGas},
		{"second sender",
			build(nil, priceWrite(feeder), priceWrite(otherFeeder)),
			proposalLaneZeroGas},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, classifyProposalTx(tc.tx, zeroGasActors))
		})
	}
}
