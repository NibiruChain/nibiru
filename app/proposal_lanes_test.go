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

// classifyFixture holds accounts and a tx builder for [classifyProposalTx]
// tests: two allowlisted feeders, a payer off the allowlist, and the oracle
// and perp contracts on it.
type classifyFixture struct {
	t                          testing.TB
	feeder, otherFeeder, payer sdk.AccAddress
	oracle, perp               string
	zeroGasActors              sudo.ZeroGasActors
}

func newClassifyFixture(t testing.TB) classifyFixture {
	newAddr := func(seed byte) sdk.AccAddress {
		return sdk.AccAddress(bytes.Repeat([]byte{seed}, 20))
	}
	f := classifyFixture{
		t:           t,
		feeder:      newAddr(1),
		otherFeeder: newAddr(2),
		payer:       newAddr(3),
		oracle:      newAddr(4).String(),
		perp:        newAddr(5).String(),
	}
	f.zeroGasActors = sudo.ZeroGasActors{
		Senders:   []string{f.feeder.String(), f.otherFeeder.String()},
		Contracts: []string{f.oracle, f.perp},
	}
	return f
}

func (f classifyFixture) exec(
	sender sdk.AccAddress, contract, msg string,
) sdk.Msg {
	return &wasm.MsgExecuteContract{
		Sender:   sender.String(),
		Contract: contract,
		Msg:      wasm.RawContractMessage(msg),
	}
}

func (f classifyFixture) build(feePayer sdk.AccAddress, msgs ...sdk.Msg) sdk.Tx {
	builder := MakeEncodingConfig().TxConfig.NewTxBuilder()
	require.NoError(f.t, builder.SetMsgs(msgs...))
	if feePayer != nil {
		builder.SetFeePayer(feePayer)
	}
	return builder.GetTx()
}

const (
	testSetPricesMsg = `{"set_prices":{"prices":[]}}`
	testExpiryMsg    = `{"update_expiration_time":{"expiration_time":60}}`
	testAdminMsg     = `{"update_permissions":{"address":"x","permission":"y"}}`
)

// TestClassifyProposalTxOracleSoleSigner proves a tx is in the oracle lane
// only when its sender is its only tx-level signer, under both the resolved
// Sai oracle address and the message-key fallback. An oracle call with a
// distinct fee payer, or a batch of oracle calls from two senders, is a
// zero-gas actor tx but not an oracle-lane tx, so it lands in the zero-gas
// lane. A fee payer equal to the sender adds no signer and keeps the oracle
// lane.
func TestClassifyProposalTxOracleSoleSigner(t *testing.T) {
	f := newClassifyFixture(t)
	call := func(sender sdk.AccAddress) sdk.Msg {
		return f.exec(sender, f.oracle, testSetPricesMsg)
	}
	for _, rule := range []struct {
		name   string
		oracle oracleLaneRule
	}{
		{"resolved", oracleLaneRule{saiOracle: f.oracle}},
		{"fallback", oracleLaneRule{}},
	} {
		for _, tc := range []struct {
			name string
			tx   sdk.Tx
			want proposalLane
		}{
			{"sole signer", f.build(nil, call(f.feeder)), proposalLaneOracle},
			{"batch from one sender",
				f.build(nil, call(f.feeder), call(f.feeder)), proposalLaneOracle},
			{"fee payer is the sender",
				f.build(f.feeder, call(f.feeder)), proposalLaneOracle},
			{"distinct fee payer",
				f.build(f.payer, call(f.feeder)), proposalLaneZeroGas},
			{"second sender",
				f.build(nil, call(f.feeder), call(f.otherFeeder)),
				proposalLaneZeroGas},
			{"sender off the allowlist",
				f.build(nil, call(f.payer)), proposalLaneDefault},
		} {
			t.Run(rule.name+"/"+tc.name, func(t *testing.T) {
				require.Equal(t, tc.want,
					classifyProposalTx(tc.tx, f.zeroGasActors, rule.oracle))
			})
		}
	}
}

// TestClassifyProposalTxOracleContract proves that with a resolved Sai oracle
// address the oracle lane matches the contract, not the message: every
// execute of the oracle qualifies, including expiration and permission
// calls, while a price write to another allowlisted contract, or a batch
// that also calls one, is only zero-gas. Under the fallback the same txs are
// matched by message key instead.
func TestClassifyProposalTxOracleContract(t *testing.T) {
	f := newClassifyFixture(t)
	resolved := oracleLaneRule{saiOracle: f.oracle}
	fallback := oracleLaneRule{}
	for _, tc := range []struct {
		name               string
		tx                 sdk.Tx
		resolved, fallback proposalLane
	}{
		{"price write to oracle",
			f.build(nil, f.exec(f.feeder, f.oracle, testSetPricesMsg)),
			proposalLaneOracle, proposalLaneOracle},
		{"expiration update to oracle",
			f.build(nil, f.exec(f.feeder, f.oracle, testExpiryMsg)),
			proposalLaneOracle, proposalLaneZeroGas},
		{"permission call to oracle",
			f.build(nil, f.exec(f.feeder, f.oracle, testAdminMsg)),
			proposalLaneOracle, proposalLaneZeroGas},
		{"malformed JSON to oracle",
			f.build(nil, f.exec(f.feeder, f.oracle, `{"set_prices":`)),
			proposalLaneOracle, proposalLaneZeroGas},
		{"price write to perp",
			f.build(nil, f.exec(f.feeder, f.perp, testSetPricesMsg)),
			proposalLaneZeroGas, proposalLaneOracle},
		{"oracle and perp batch",
			f.build(nil,
				f.exec(f.feeder, f.oracle, testExpiryMsg),
				f.exec(f.feeder, f.perp, testExpiryMsg)),
			proposalLaneZeroGas, proposalLaneZeroGas},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.resolved,
				classifyProposalTx(tc.tx, f.zeroGasActors, resolved), "resolved")
			require.Equal(t, tc.fallback,
				classifyProposalTx(tc.tx, f.zeroGasActors, fallback), "fallback")
		})
	}
}
