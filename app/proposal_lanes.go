package app

import (
	"encoding/json"
	"fmt"

	abci "github.com/NibiruChain/nibiru/v2/bft/abci/types"

	"github.com/NibiruChain/nibiru/v2/app/ante"
	"github.com/NibiruChain/nibiru/v2/evm"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	"github.com/NibiruChain/nibiru/v2/x/sudo"
	wasm "github.com/NibiruChain/nibiru/v2/x/wasm/types"
)

// ProposalLane is the block lane of one proposal transaction. Lanes are
// ordered: a lower value must appear earlier in the block.
type ProposalLane uint8

const (
	// LaneOracle holds zero-gas actor transactions in which every message is a
	// Wasm execute whose JSON has the single top-level key "set_prices" or
	// "set_price".
	LaneOracle ProposalLane = iota
	// LaneZeroGas holds every other zero-gas actor transaction.
	LaneZeroGas
	// LaneEVM holds Ethereum transactions ([evm.IsEthTx]).
	LaneEVM
	// LaneCosmos holds every remaining decodable transaction.
	LaneCosmos
)

func (lane ProposalLane) String() string {
	switch lane {
	case LaneOracle:
		return "oracle"
	case LaneZeroGas:
		return "zero-gas"
	case LaneEVM:
		return "evm"
	case LaneCosmos:
		return "cosmos"
	default:
		return fmt.Sprintf("lane(%d)", uint8(lane))
	}
}

// oraclePriceExecuteKeys are the Wasm execute variants that write prices.
var oraclePriceExecuteKeys = map[string]bool{
	"set_prices": true,
	"set_price":  true,
}

// ClassifyProposalTx assigns tx to a lane. It is a pure function of the
// decoded tx and zeroGasActors, which callers read from committed state, so
// every validator computes the same lane for the same proposal bytes.
func ClassifyProposalTx(tx sdk.Tx, zeroGasActors sudo.ZeroGasActors) ProposalLane {
	if evm.IsEthTx(tx) {
		return LaneEVM
	}
	if !ante.IsZeroGasActorTx(tx, zeroGasActors) {
		return LaneCosmos
	}
	for _, msg := range tx.GetMsgs() {
		// IsZeroGasActorTx guarantees every message is a Wasm execute.
		exec := msg.(*wasm.MsgExecuteContract)
		if !isOraclePriceExecute(exec.Msg) {
			return LaneZeroGas
		}
	}
	return LaneOracle
}

// isOraclePriceExecute reports whether raw is a JSON object with exactly one
// top-level key naming a price write. encoding/json is deterministic for a
// given input, so the result is consensus-safe.
func isOraclePriceExecute(raw []byte) bool {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil || len(obj) != 1 {
		return false
	}
	for key := range obj {
		return oraclePriceExecuteKeys[key]
	}
	return false
}

// LaneOrderRule selects which ordering ProcessProposal enforces.
type LaneOrderRule uint8

const (
	// LaneOrderMonotonic requires lanes to be non-decreasing across the block:
	// oracle, zero-gas, EVM, Cosmos. Order inside a lane is not checked.
	LaneOrderMonotonic LaneOrderRule = iota
	// LaneOrderOracleFirst requires only that no oracle-lane tx follows a
	// tx from another lane (Block SDK VerifyNoMatches semantics for one lane).
	LaneOrderOracleFirst
)

// LaneOrderConfig configures [VerifyProposalLaneOrder].
type LaneOrderConfig struct {
	Rule LaneOrderRule
	// MaxOracleLaneTxs caps oracle-lane transactions per block. Zero means no
	// cap. Zero-gas txs bypass the block gas limit, so this is the only bound
	// besides block bytes. PrepareProposal must apply the same cap.
	MaxOracleLaneTxs int
}

// DefaultLaneOrderConfig is the prototype's consensus configuration.
var DefaultLaneOrderConfig = LaneOrderConfig{Rule: LaneOrderMonotonic}

// VerifyProposalLaneOrder decodes each proposal tx, classifies it, and returns
// an error when the lane sequence breaks cfg. An undecodable tx is an error,
// matching the default ProcessProposal handler, which rejects such proposals.
func VerifyProposalLaneOrder(
	decode sdk.TxDecoder,
	txs [][]byte,
	zeroGasActors sudo.ZeroGasActors,
	cfg LaneOrderConfig,
) error {
	prev := LaneOracle
	oracleTxs := 0
	for idx, txBytes := range txs {
		tx, err := decode(txBytes)
		if err != nil {
			return fmt.Errorf("tx %d: undecodable: %w", idx, err)
		}
		lane := ClassifyProposalTx(tx, zeroGasActors)
		if lane == LaneOracle {
			oracleTxs++
			if cfg.MaxOracleLaneTxs > 0 && oracleTxs > cfg.MaxOracleLaneTxs {
				return fmt.Errorf(
					"tx %d: oracle lane exceeds %d txs", idx, cfg.MaxOracleLaneTxs,
				)
			}
		}
		switch cfg.Rule {
		case LaneOrderMonotonic:
			if lane < prev {
				return fmt.Errorf("tx %d: %s lane after %s lane", idx, lane, prev)
			}
		case LaneOrderOracleFirst:
			if lane == LaneOracle && prev != LaneOracle {
				return fmt.Errorf("tx %d: oracle lane after %s lane", idx, prev)
			}
		default:
			return fmt.Errorf("unknown lane order rule %d", cfg.Rule)
		}
		if lane > prev {
			prev = lane
		}
	}
	return nil
}

// LaneOrderUpgradeName is the software upgrade that activates lane-order
// enforcement. Placeholder: the production version registers it in
// app/upgrades.
const LaneOrderUpgradeName = "v2.23.0"

// LaneOrderActiveAt reports whether lane order is enforced for a proposal at
// height, given the height the activating upgrade ran at (0 when it has not
// run). ProcessProposal at the upgrade height itself sees pre-upgrade state,
// so enforcement starts at doneHeight+1.
func LaneOrderActiveAt(doneHeight, height int64) bool {
	return doneHeight > 0 && height > doneHeight
}

// NewLaneOrderProcessProposalHandler wraps next with the lane-order check.
// isActive gates enforcement; zeroGasActors reads the x/sudo set from the
// ProcessProposal context, which CometBFT builds from the last committed
// state. A failing check rejects the proposal before next runs.
func NewLaneOrderProcessProposalHandler(
	decode sdk.TxDecoder,
	zeroGasActors func(sdk.Context) sudo.ZeroGasActors,
	isActive func(sdk.Context) bool,
	cfg LaneOrderConfig,
	next sdk.ProcessProposalHandler,
) sdk.ProcessProposalHandler {
	return func(
		ctx sdk.Context, req abci.RequestProcessProposal,
	) abci.ResponseProcessProposal {
		if isActive(ctx) {
			err := VerifyProposalLaneOrder(decode, req.Txs, zeroGasActors(ctx), cfg)
			if err != nil {
				ctx.Logger().Info(
					"rejecting proposal: lane order", "height", req.Height, "err", err,
				)
				return abci.ResponseProcessProposal{
					Status: abci.ResponseProcessProposal_REJECT,
				}
			}
		}
		return next(ctx, req)
	}
}
