package upgrades

import (
	"encoding/hex"
	"encoding/json"
	"fmt"

	sdkmath "cosmossdk.io/math"

	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/app/keepers"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	wasmkeeper "github.com/NibiruChain/nibiru/v2/x/wasm/keeper"
)

const (
	erisRecoveryContract = "nibi1udqqx30cw8nwjxtl4l28ym9hhrp933zlq8dqxfjzcdhvl8y24zcqpzmh8m"
	erisRecoveryCaller   = "nibi1j3a73n3q72mdalp7mpvn4t4lt6vhfpqm00fup8"
	erisRecoveryCodeHash = "e1c2be31ae008a015efa16b51f74ca1a014f4b1d0952da12ba3f60aaffa66321"
	erisRecoveryShares   = "32298107051806"
	erisRecoveryBatchID  = uint64(172)
	erisRecoveryGasLimit = uint64(10_000_000)
)

type erisRecoveryBatch struct {
	ID          uint64      `json:"id"`
	Reconciled  bool        `json:"reconciled"`
	TotalShares sdkmath.Int `json:"total_shares"`
	Unclaimed   sdkmath.Int `json:"utoken_unclaimed"`
	EndTime     uint64      `json:"est_unbond_end_time"`
}

type erisRecoveryRequest struct {
	ID     uint64             `json:"id"`
	Shares sdkmath.Int        `json:"shares"`
	Batch  *erisRecoveryBatch `json:"batch"`
}

// recoverErisV221 isolates this optional recovery from the runtime repair. Even
// execution panics or a failed Bank dispatch must discard the contract's writes.
func recoverErisV221(ctx sdk.Context, nibiru *keepers.PublicKeepers) {
	if ctx.ChainID() != appconst.SDK_CHAIN_ID_MAINNET {
		return
	}
	cached, commit := ctx.CacheContext()
	cached = cached.WithGasMeter(sdk.NewGasMeter(erisRecoveryGasLimit))
	amount, err := attemptErisRecoveryV221(cached, nibiru)
	status := "skipped"
	if err != nil {
		status = "failed"
		ctx.Logger().Error("v2.21 Eris recovery failed", "err", err)
	} else if amount.IsPositive() {
		commit()
		status = "success"
	}
	attrs := []sdk.Attribute{
		sdk.NewAttribute("upgrade", "v2.21.0"),
		sdk.NewAttribute("status", status),
		sdk.NewAttribute("contract", erisRecoveryContract),
		sdk.NewAttribute("caller", erisRecoveryCaller),
		sdk.NewAttribute("recipient", IncidentRecoveryCW3_v2_18),
		sdk.NewAttribute("batch_id", "172"),
		sdk.NewAttribute("denom", appconst.DENOM_UNIBI),
		sdk.NewAttribute("amount", amount.String()),
		sdk.NewAttribute("gas_used", fmt.Sprint(cached.GasMeter().GasConsumed())),
	}
	if err != nil {
		attrs = append(attrs, sdk.NewAttribute("error", err.Error()))
	}
	ctx.EventManager().EmitEvent(sdk.NewEvent("eris_recovery", attrs...))
}

func attemptErisRecoveryV221(ctx sdk.Context, nibiru *keepers.PublicKeepers) (amount sdkmath.Int, err error) {
	amount = sdkmath.ZeroInt()
	defer func() {
		if panicValue := recover(); panicValue != nil {
			amount = sdkmath.ZeroInt()
			switch value := panicValue.(type) {
			case sdk.ErrorOutOfGas:
				err = fmt.Errorf("Eris recovery out of gas: %s", value.Descriptor)
			default:
				err = fmt.Errorf("Eris recovery panic (%T): %v", panicValue, panicValue)
			}
		}
		if err != nil {
			amount = sdkmath.ZeroInt()
		}
	}()
	contract := sdk.MustAccAddressFromBech32(erisRecoveryContract)
	caller := sdk.MustAccAddressFromBech32(erisRecoveryCaller)
	recipient := sdk.MustAccAddressFromBech32(IncidentRecoveryCW3_v2_18)
	info := nibiru.WasmKeeper.GetContractInfo(ctx, contract)
	if info == nil {
		return amount, fmt.Errorf("Eris contract not found")
	}
	code := nibiru.WasmKeeper.GetCodeInfo(ctx, info.CodeID)
	if code == nil || hex.EncodeToString(code.CodeHash) != erisRecoveryCodeHash {
		return amount, fmt.Errorf("unexpected Eris bytecode for code %d", info.CodeID)
	}
	if !nibiru.WasmKeeper.HasContractInfo(ctx, recipient) {
		return amount, fmt.Errorf("Treasury CW3 contract not found")
	}
	var config struct {
		Denom string `json:"utoken"`
	}
	if err := queryErisV221(ctx, nibiru, `{"config":{}}`, &config); err != nil {
		return amount, err
	}
	if config.Denom != appconst.DENOM_UNIBI {
		return amount, fmt.Errorf("unexpected Eris denom %q", config.Denom)
	}
	requests, err := erisRequestsV221(ctx, nibiru)
	if err != nil || len(requests) == 0 {
		return amount, err
	}
	// Eris withdraws all matured requests for its caller. Reject additional
	// requests instead of unintentionally recovering unrelated claims.
	if len(requests) != 1 || requests[0].ID != erisRecoveryBatchID {
		return amount, fmt.Errorf("expected only Eris batch 172 request")
	}
	request := requests[0]
	expectedShares, _ := sdkmath.NewIntFromString(erisRecoveryShares)
	if request.Shares.IsNil() || !request.Shares.Equal(expectedShares) {
		return amount, fmt.Errorf("unexpected attacker shares")
	}
	batch := request.Batch
	if batch == nil || batch.ID != erisRecoveryBatchID || !batch.Reconciled ||
		ctx.BlockTime().Unix() <= 0 || uint64(ctx.BlockTime().Unix()) <= batch.EndTime {
		return amount, fmt.Errorf("Eris batch 172 is not reconciled and matured")
	}
	if batch.TotalShares.IsNil() || batch.TotalShares.LT(request.Shares) ||
		batch.Unclaimed.IsNil() || !batch.Unclaimed.IsPositive() {
		return amount, fmt.Errorf("invalid Eris batch accounting")
	}
	// CosmWasm Uint128.multiply_ratio truncates. sdkmath.Int also provides exact
	// integer division, including the one-unibi difference if the other user claims first.
	expected := batch.Unclaimed.Mul(request.Shares).Quo(batch.TotalShares)
	if !expected.IsPositive() {
		return amount, fmt.Errorf("zero Eris payout")
	}
	beforeContract := nibiru.BankKeeper.GetBalance(ctx, contract, config.Denom)
	beforeRecipient := nibiru.BankKeeper.GetBalance(ctx, recipient, config.Denom)
	beforeCaller := nibiru.BankKeeper.GetAllBalances(ctx, caller)
	beforeSupply := nibiru.BankKeeper.GetSupply(ctx, config.Denom)
	if beforeContract.Amount.LT(expected) {
		return amount, fmt.Errorf("insufficient Eris balance for %s unibi", expected)
	}
	// Internal keeper execution intentionally supplies the quarantined caller
	// without entering public transaction ante handlers. The contract performs
	// its own normal request removal, batch accounting, and Bank send to CW3.
	msg := []byte(`{"withdraw_unbonded":{"receiver":"` + IncidentRecoveryCW3_v2_18 + `"}}`)
	_, err = wasmkeeper.NewDefaultPermissionKeeper(nibiru.WasmKeeper).Execute(
		ctx, contract, caller, msg, sdk.Coins{},
	)
	if err != nil {
		return amount, fmt.Errorf("withdraw Eris batch 172: %w", err)
	}
	afterRequests, err := erisRequestsV221(ctx, nibiru)
	if err != nil {
		return amount, err
	}
	if len(afterRequests) != 0 {
		return amount, fmt.Errorf("attacker still has Eris requests")
	}
	if !nibiru.BankKeeper.GetBalance(ctx, contract, config.Denom).Amount.Equal(beforeContract.Amount.Sub(expected)) ||
		!nibiru.BankKeeper.GetBalance(ctx, recipient, config.Denom).Amount.Equal(beforeRecipient.Amount.Add(expected)) ||
		!nibiru.BankKeeper.GetAllBalances(ctx, caller).IsEqual(beforeCaller) ||
		!nibiru.BankKeeper.GetSupply(ctx, config.Denom).IsEqual(beforeSupply) {
		return amount, fmt.Errorf("unexpected recovery balance or supply changes")
	}
	var remaining []erisRecoveryBatch
	if err := queryErisV221(ctx, nibiru, `{"previous_batches":{"start_after":171,"limit":1}}`, &remaining); err != nil {
		return amount, err
	}
	remainingShares := batch.TotalShares.Sub(request.Shares)
	if remainingShares.IsZero() {
		if len(remaining) > 0 && remaining[0].ID == erisRecoveryBatchID {
			return amount, fmt.Errorf("empty batch 172 still exists")
		}
	} else if len(remaining) != 1 || remaining[0].ID != erisRecoveryBatchID ||
		remaining[0].TotalShares.IsNil() || remaining[0].Unclaimed.IsNil() ||
		!remaining[0].TotalShares.Equal(remainingShares) ||
		!remaining[0].Unclaimed.Equal(batch.Unclaimed.Sub(expected)) ||
		remaining[0].Reconciled != batch.Reconciled || remaining[0].EndTime != batch.EndTime {
		return amount, fmt.Errorf("unexpected remaining Eris batch accounting")
	}
	return expected, nil
}

func erisRequestsV221(ctx sdk.Context, nibiru *keepers.PublicKeepers) ([]erisRecoveryRequest, error) {
	var requests []erisRecoveryRequest
	err := queryErisV221(ctx, nibiru,
		`{"unbond_requests_by_user_details":{"user":"`+erisRecoveryCaller+`","limit":2}}`, &requests)
	return requests, err
}

func queryErisV221(ctx sdk.Context, nibiru *keepers.PublicKeepers, msg string, result any) error {
	bz, err := nibiru.WasmKeeper.QuerySmart(ctx, sdk.MustAccAddressFromBech32(erisRecoveryContract), []byte(msg))
	if err != nil {
		return fmt.Errorf("query Eris: %w", err)
	}
	if err := json.Unmarshal(bz, result); err != nil {
		return fmt.Errorf("decode Eris query: %w", err)
	}
	return nil
}
