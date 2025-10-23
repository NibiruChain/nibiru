package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/NibiruChain/nibiru/v2/x/nutil"
)

// RegisterInvariants registers the bank module invariants
func RegisterInvariants(ir sdk.InvariantRegistry, k Keeper) {
	ir.RegisterRoute(types.ModuleName, "nonnegative-outstanding", NonnegativeBalanceInvariant(k))
	ir.RegisterRoute(types.ModuleName, "total-supply", TotalSupply(k))
}

// AllInvariants runs all invariants of the X/bank module.
func AllInvariants(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		return TotalSupply(k)(ctx)
	}
}

// NonnegativeBalanceInvariant checks that all accounts in the application have non-negative balances
func NonnegativeBalanceInvariant(k ViewKeeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		var (
			msg   string
			count int
		)

		k.IterateAllBalances(ctx, func(addr sdk.AccAddress, balance sdk.Coin) bool {
			if balance.IsNegative() {
				count++
				msg += fmt.Sprintf("\t%s has a negative balance of %s\n", addr, balance)
			}

			return false
		})

		broken := count != 0

		return sdk.FormatInvariant(
			types.ModuleName, "nonnegative-outstanding",
			fmt.Sprintf("amount of negative balances found %d\n%s", count, msg),
		), broken
	}
}

// TotalSupply returns an [sdk.Invariant] that checks supply accounting.
//
// ### Summary
//   - Verifies that the sum of all account balances (excluding UNIBI)
//     equals the bank module’s TotalSupply for those coins.
//   - For UNIBI, the check is relaxed since some value exists in the wei store.
//     The invariant reports UNIBI context but does not fail on mismatches.
//
// ### Invariant rule
//  1. Subtract UNIBI from both totals.
//  2. Check equality on the remaining coins.
//  3. If non-UNIBI totals differ, the invariant is broken.
//
// ### Rationale
//   - Non-UNIBI coins must be strictly conserved.
//   - UNIBI can convert to wei for EVM math, so minor discrepancies are valid.
//   - The invariant surfaces UNIBI context without penalizing conversions.
//
// ### Computed Values for Invariant Rule
//   - totalBalOfCoins:  sum of all account balances.
//   - supplyOfCoins:    total supply reported by the bank module.
//
// ### Computed Values for NIBI:
//   - totalBalOfUnibi: total UNIBI held in accounts.
//   - supplyOfUnibiCoins: UNIBI TotalSupply from the bank.
//   - supplyOfUnibiCoinsInWei: UNIBI supply scaled by WeiPerUnibi.
//
// TODO: test TotalSupply
func TotalSupply(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		var (
			totalBalOfCoins = sdk.Coins{}
			supplyOfCoins   sdk.Coins
			// A status message added to the invariant debug string
			brokenStatusMsg string
		)
		supplyOfCoins, _, err := k.GetPaginatedTotalSupply(ctx, &query.PageRequest{Limit: query.MaxLimit})
		if err != nil {
			return sdk.FormatInvariant(types.ModuleName, "query supply",
				fmt.Sprintf("error querying total supply %v", err)), false
		}

		k.IterateAllBalances(ctx, func(addr sdk.AccAddress, balance sdk.Coin) bool {
			totalBalOfCoins = totalBalOfCoins.Add(balance)
			return false
		})

		// NIBI supply := all micronibi + all of wei store + NetWeiBlockDelta
		// 1. The bank "TotalSupply" of supply of unibi is modified only by mints
		// and burns, not by sending operations that are zero sum.
		// == netWeiDelta + totalWeiStoreBal + weiHeldInUnibi
		// 2. NetWeiBlockDelta comes from the EVM module, so we lighten the
		// invariant restriction here.
		totalBalOfUnibi := totalBalOfCoins.AmountOf(DENOM_UNIBI)
		supplyOfUnibiCoins := supplyOfCoins.AmountOf(DENOM_UNIBI)

		supplyOfUnibiCoinsInWei := nutil.WeiPerUnibi.Mul(supplyOfUnibiCoins)

		// All other supplies are in coins only
		// Compare all other coins aside from NIBI by subtracting it from
		// the totalssubtract UNIBI from totals before comparison
		totalBalOfCoins = totalBalOfCoins.
			Sub(sdk.NewCoin(DENOM_UNIBI, totalBalOfUnibi))
		supplyOfCoins = supplyOfCoins.
			Sub(sdk.NewCoin(DENOM_UNIBI, supplyOfUnibiCoins))

		brokenOthers := !totalBalOfCoins.IsEqual(supplyOfCoins)

		broken := brokenOthers
		if !brokenOthers {
			brokenStatusMsg = "invariant PASSED"
		} else {
			brokenStatusMsg = "sum of non-NIBI coins is broken. Note that we don't require NIBI to match for this invariant because some of it becomes wei (attonibi), enabling 18 decimal math in the EVM."
		}

		return sdk.FormatInvariant(types.ModuleName, "total supply",
			fmt.Sprintf(
				"\tbroken_status: \"%s\"\n"+
					"\tsum of accounts coins:   %s\n"+
					"\tsupply.Total:            %s\n"+
					"\tsupply.Total of unibi in wei units: %s\n"+
					brokenStatusMsg,
				totalBalOfCoins,
				supplyOfCoins,
				supplyOfUnibiCoinsInWei,
			)), broken
	}
}
