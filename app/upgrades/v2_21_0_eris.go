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

// The checksum pins contract behavior even if a different code ID contains the
// same bytecode. Incident shares identify the reviewed request. Eris computes
// the payout from its own batch state, including changes from another user claiming.
const (
	erisRecoveryContract = "nibi1udqqx30cw8nwjxtl4l28ym9hhrp933zlq8dqxfjzcdhvl8y24zcqpzmh8m"
	erisRecoveryCaller   = "nibi1j3a73n3q72mdalp7mpvn4t4lt6vhfpqm00fup8"
	erisRecoveryCodeHash = "e1c2be31ae008a015efa16b51f74ca1a014f4b1d0952da12ba3f60aaffa66321"
	erisRecoveryShares   = "32298107051806"
	erisRecoveryBatchID  = uint64(172)
)

// erisRecoveryBatch projects only the fields that decide whether the reviewed
// claim is eligible. Payout calculation and batch accounting belong to Eris.
type erisRecoveryBatch struct {
	ID         uint64 `json:"id"`
	Reconciled bool   `json:"reconciled"`
	EndTime    uint64 `json:"est_unbond_end_time"`
}

// erisRecoveryRequest includes the batch returned by the user-details query,
// which lets preflight inspect the claim and its maturity in the same response.
type erisRecoveryRequest struct {
	ID     uint64             `json:"id"`
	Shares sdkmath.Int        `json:"shares"`
	Batch  *erisRecoveryBatch `json:"batch"`
}

// recoverErisV221 isolates incident recovery from deployment grants and schema
// migration. The parent-context event survives discarded contract writes and
// reports zero recovered funds on failure. A missing request makes repeat
// execution a no-op; other users' claims remain under Eris's normal accounting.
func (h Handler_v2_21) recoverErisV221(ctx sdk.Context, nibiru *keepers.PublicKeepers) error {
	if ctx.ChainID() != appconst.SDK_CHAIN_ID_MAINNET {
		return nil
	}
	amount := "0"
	executed := false
	var gasUsed uint64
	err := h.runCachedUpgradeStep(ctx, func(cached sdk.Context) error {
		defer func() { gasUsed = cached.GasMeter().GasConsumed() }()
		var err error
		executed, err = h.attemptErisRecoveryV221(cached, nibiru)
		if err == nil && executed {
			// Report Eris's own payout event without another query or assertion.
			// Missing event metadata must not reject a successful withdrawal.
			amount = ""
			for _, event := range cached.EventManager().Events() {
				if event.Type == "wasm-erishub/unbonded_withdrawn" {
					if attr, ok := event.GetAttribute("utoken_refunded"); ok {
						amount = attr.Value
					}
				}
			}
		}
		return err
	})
	status := "skipped"
	if err != nil {
		amount = "0"
		status = "failed"
	} else if executed {
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
		sdk.NewAttribute("gas_used", fmt.Sprint(gasUsed)),
	}
	if amount != "" {
		attrs = append(attrs, sdk.NewAttribute("amount", amount))
	}
	if err != nil {
		attrs = append(attrs, sdk.NewAttribute("error", err.Error()))
	}
	ctx.EventManager().EmitEvent(sdk.NewEvent("eris_recovery", attrs...))
	return err
}

// attemptErisRecoveryV221 uses the quarantined account only as Eris's claim
// owner. The receiver directs the native payout to Treasury without granting
// public transaction authority back to the attacker. Its caller must provide a
// cached context because contract writes can precede a failed Bank dispatch.
// After eligibility checks, successful keeper execution is sufficient to commit:
// Eris owns payout arithmetic, request removal, and the Bank send. Assertions
// about resulting balances, supply, and storage stay in the deployed-code replay.
func (h Handler_v2_21) attemptErisRecoveryV221(ctx sdk.Context, nibiru *keepers.PublicKeepers) (bool, error) {
	contract := sdk.MustAccAddressFromBech32(erisRecoveryContract)
	caller := sdk.MustAccAddressFromBech32(erisRecoveryCaller)
	recipient := sdk.MustAccAddressFromBech32(IncidentRecoveryCW3_v2_18)
	info := nibiru.WasmKeeper.GetContractInfo(ctx, contract)
	if info == nil {
		return false, fmt.Errorf("Eris contract not found")
	}
	code := nibiru.WasmKeeper.GetCodeInfo(ctx, info.CodeID)
	if code == nil || hex.EncodeToString(code.CodeHash) != erisRecoveryCodeHash {
		return false, fmt.Errorf("unexpected Eris bytecode for code %d", info.CodeID)
	}
	if !nibiru.WasmKeeper.HasContractInfo(ctx, recipient) {
		return false, fmt.Errorf("Treasury CW3 contract not found")
	}
	var config struct {
		Denom string `json:"utoken"`
	}
	if err := h.queryErisV221(ctx, nibiru, `{"config":{}}`, &config); err != nil {
		return false, err
	}
	if config.Denom != appconst.DENOM_UNIBI {
		return false, fmt.Errorf("unexpected Eris denom %q", config.Denom)
	}
	requests, err := h.erisRequestsV221(ctx, nibiru)
	if err != nil || len(requests) == 0 {
		return false, err
	}
	// Eris withdraws all matured requests for its caller. Reject additional
	// requests instead of unintentionally recovering unrelated claims.
	if len(requests) != 1 || requests[0].ID != erisRecoveryBatchID {
		return false, fmt.Errorf("expected only Eris batch 172 request")
	}
	request := requests[0]
	expectedShares, _ := sdkmath.NewIntFromString(erisRecoveryShares)
	if request.Shares.IsNil() || !request.Shares.Equal(expectedShares) {
		return false, fmt.Errorf("unexpected attacker shares")
	}
	batch := request.Batch
	if batch == nil || batch.ID != erisRecoveryBatchID || !batch.Reconciled ||
		ctx.BlockTime().Unix() <= 0 || uint64(ctx.BlockTime().Unix()) <= batch.EndTime {
		return false, fmt.Errorf("Eris batch 172 is not reconciled and matured")
	}
	// Internal keeper execution intentionally supplies the quarantined caller
	// without entering public transaction ante handlers. The contract performs
	// its own normal request removal, batch accounting, and Bank send to CW3.
	msg := []byte(`{"withdraw_unbonded":{"receiver":"` + IncidentRecoveryCW3_v2_18 + `"}}`)
	_, err = wasmkeeper.NewDefaultPermissionKeeper(nibiru.WasmKeeper).Execute(
		ctx, contract, caller, msg, sdk.Coins{},
	)
	if err != nil {
		return false, fmt.Errorf("withdraw Eris batch 172: %w", err)
	}
	return true, nil
}

// erisRequestsV221 requests two entries to detect an additional claim. Eris
// withdraws all eligible requests for a caller, so accepting only the reviewed
// request prevents this upgrade from sweeping a different incident-account claim.
func (h Handler_v2_21) erisRequestsV221(ctx sdk.Context, nibiru *keepers.PublicKeepers) ([]erisRecoveryRequest, error) {
	var requests []erisRecoveryRequest
	err := h.queryErisV221(ctx, nibiru,
		`{"unbond_requests_by_user_details":{"user":"`+erisRecoveryCaller+`","limit":2}}`, &requests)
	return requests, err
}

// queryErisV221 queries the deployed contract against the upgrade context.
// These reads use activation-time state and consume the recovery step's gas;
// the JSON response is contract output without the CLI's outer data wrapper.
func (h Handler_v2_21) queryErisV221(ctx sdk.Context, nibiru *keepers.PublicKeepers, msg string, result any) error {
	bz, err := nibiru.WasmKeeper.QuerySmart(ctx, sdk.MustAccAddressFromBech32(erisRecoveryContract), []byte(msg))
	if err != nil {
		return fmt.Errorf("query Eris: %w", err)
	}
	if err := json.Unmarshal(bz, result); err != nil {
		return fmt.Errorf("decode Eris query: %w", err)
	}
	return nil
}
