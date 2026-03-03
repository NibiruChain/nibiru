package keeper

import (
	"fmt"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bank "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/holiman/uint256"

	"github.com/NibiruChain/nibiru/v2/x/collections"

	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/x/nutil"
)

// StoreKeyTransient defines the transient store key
const StoreKeyTransient = "transient_" + bank.ModuleName

var _ NibiruExtKeeper = (*BaseSendKeeper)(nil)

// NibiruExtKeeper exposes Nibiru-specific balance operations and accounting.
//
// Dual-balance model
//   - Bank balance (micro-denom): The canonical x/bank balance is stored in the
//     chain's micro-denomination "unibi" (micro-NIBI; 10^-6 NIBI).
//   - Sub-unit store (wei remainder): A separate KV store tracks the remainder
//     of each account's NIBI at higher precision ("wei"), where
//     WeiPerUnibi = 10^12. Together they represent a unified balance with
//     18 decimals (10^6 * 10^12 = 10^18).
//
// Aggregation
//   - GetWeiBalance(ctx, addr) returns the aggregate balance in wei:
//     agg_wei = (unibi * 10^12) + wei_store
//     The return type is uint256 to match EVM semantics.
//
// Normalization (carry/borrow at the 10^12 boundary)
//   - When adding or subtracting wei, we normalize at the 10^12 boundary:
//     if wei_store >= 10^12 → carry 1 unibi to x/bank and keep remainder
//     if wei_store < 0      → borrow 1+ unibi from x/bank to make wei_store ≥ 0
//     Callers never manipulate x/bank units directly; AddWei/SubWei perform the
//     carry/borrow and keep both stores consistent.
//
// Supply accounting
//   - WeiBlockDelta(ctx) tracks the net wei added via AddWei/SubWei within the
//     current block (transient). The EVM module uses this value each block to
//     maintain supply invariants (e.g., reconcile protocol mints/burns/fees).
//   - SumWeiStoreBals(ctx) iterates the entire wei remainder store and sums it,
//     intended only for the crisis invariant that validates total supply.
//
// Events
//   - AddWei/SubWei emit bank.EventWeiChange with a reason code.
//   - x/bank coin transfers that affect "unibi" also emit a WeiChange event
//     so downstream consumers can detect changes to the aggregated wei balance.
type NibiruExtKeeper interface {
	// AddWei increases an account’s balance by amtWei (in wei), performing
	// normalization at the 10^12 boundary as needed. No-op for nil/zero input.
	AddWei(ctx sdk.Context, addr sdk.AccAddress, amtWei *uint256.Int)

	// GetWeiBalance returns the full account balance in wei:
	// (bank_unibi * 10^12) + wei_store.
	GetWeiBalance(ctx sdk.Context, addr sdk.AccAddress) (bal *uint256.Int)

	// SubWei decreases an account’s balance by amtWei (in wei). If the wei
	// remainder store is insufficient, it borrows from the x/bank “unibi”
	// balance. Returns an error if the aggregate wei balance is insufficient.
	SubWei(ctx sdk.Context, addr sdk.AccAddress, amtWei *uint256.Int) error

	// WeiBlockDelta is the net sum of all calls of [AddWei] and [SubWei] in
	// the current block. There is no guarantee that the EVM State DB will
	// add the same amount it subtracts. The EVM module manages the supply
	// invariant using [WeiBlockDelta] every block.
	WeiBlockDelta(ctx sdk.Context) sdkmath.Int

	// SumWeiStoreBals iterates across the entire wei store, summing all of the
	// balances to evaluate the [TotalSupply] [sdk.Invariant] for the crisis
	// module. This function is can be heavy to run and should not be used
	// outside of that invariant check.
	SumWeiStoreBals(ctx sdk.Context) sdkmath.Int
}

// Constants and namespaces for the dual-balance model.
const (
	DENOM_UNIBI = appconst.DENOM_UNIBI

	// NAMESPACE_BALANCE_WEI is the store prefix for the wei balance store.
	// For each address:
	//   agg_wei(addr) = bank_balance_unibi(addr) * 10^12 + wei_store(addr)
	// The wei remainder is always kept in [0, 10^12) after normalization.
	NAMESPACE_BALANCE_WEI collections.Namespace = 15

	// NAMESPACE_WEI_BLOCK_DELTA is the transient prefix tracking the net wei
	// delta (AddWei − SubWei) within the current block. Used by the EVM module
	// to reconcile protocol-level supply changes during EndBlock.
	NAMESPACE_WEI_BLOCK_DELTA collections.Namespace = 16

	// NAMESPACE_WEI_COMMITTED_DELTA holds historical committed deltas when
	// persisted. It should not be used in hot paths and exists for debugging
	// or invariant verification workflows.
	NAMESPACE_WEI_COMMITTED_DELTA collections.Namespace = 17
)

// GetWeiBalance returns the full account balance in units wei:
// (bank_unibi * 10^12) + wei_store.
// It is an official reflection of the account's total balance of NIBI.
// This "wei balance" is an aggregation of the "unibi" bank coins
// and wei store of NIBI, which is bounded within the range [0, 10^{12}).
func (k BaseSendKeeper) GetWeiBalance(
	ctx sdk.Context,
	addr sdk.AccAddress,
) (bal *uint256.Int) {
	storeBalWei := k.getWeiStoreBalance(ctx, addr)

	var storeBalUnibi *uint256.Int
	{
		balSdkInt := k.GetBalance(ctx, addr, DENOM_UNIBI).Amount
		if balSdkInt.LT(sdkmath.ZeroInt()) {
			balSdkInt = sdkmath.ZeroInt()
		}
		// MustFromBig is safehere because the supply of NIBI (10^{18} wei) is on
		// the order of billions, way beneath the threshold for safe u256 math.
		storeBalUnibi = uint256.MustFromBig(balSdkInt.BigInt())
	}

	return new(uint256.Int).Add(
		new(uint256.Int).Mul(storeBalUnibi, nutil.WeiPerUnibiU256()),
		storeBalWei,
	)
}

// AddWei increases an account’s balance by amtWei (in wei), performing
// normalization at the 10^12 boundary as needed. No-op for nil/zero input.
func (k BaseSendKeeper) AddWei(
	ctx sdk.Context,
	addr sdk.AccAddress,
	amtWei *uint256.Int,
) {
	if amtWei == nil || amtWei.IsZero() {
		return
	}

	weiStoreBalPre := k.getWeiStoreBalance(ctx, addr)
	newWeiStoreBal := new(uint256.Int).Add(weiStoreBalPre, amtWei)
	if newWeiStoreBal.Cmp(nutil.WeiPerUnibiU256()) < 0 {
		k.setWeiStoreBalance(ctx, addr, newWeiStoreBal)
	} else {
		balPre := k.GetWeiBalance(ctx, addr)
		k.setNibiBalanceFromWei(ctx, addr,
			new(uint256.Int).Add(balPre, amtWei),
		)
	}

	weiBlockDelta := k.weiBlockDelta.GetOr(ctx, sdkmath.ZeroInt())
	k.weiBlockDelta.Set(
		ctx,
		weiBlockDelta.Add(sdkmath.NewIntFromBigInt(amtWei.ToBig())),
	)

	ctx.EventManager().EmitEvent(
		bank.NewEventWeiChange(
			bank.WeiChangeReason_AddWei,
			addr.String(),
		),
	)
}

// SubWei decreases an account’s balance by amtWei (in wei). If the wei
// remainder store is insufficient, it borrows from the x/bank “unibi”
// balance. Returns an error if the aggregate wei balance is insufficient.
func (k BaseSendKeeper) SubWei(
	ctx sdk.Context,
	addr sdk.AccAddress,
	amtWei *uint256.Int,
) error {
	if amtWei == nil || amtWei.IsZero() {
		return nil
	}

	// weiBalPre is bounded, so we only need to check Cmp
	weiStoreBalPre := k.getWeiStoreBalance(ctx, addr)
	if weiStoreBalPre.Cmp(amtWei) >= 0 {
		// case(happy): amtWei < 10^{12} and weiBalPre >= amtWei
		k.setWeiStoreBalance(ctx, addr,
			new(uint256.Int).Sub(weiStoreBalPre, amtWei),
		)
	} else {
		// case(error): aggBal < amtWei -> error
		balPre := k.GetWeiBalance(ctx, addr)
		if balPre.Cmp(amtWei) < 0 {
			return fmt.Errorf(
				"SubWeiError: insufficient funds { balance: %s, amtWei: %s }",
				balPre, amtWei,
			)
		}

		// case(happy): aggBal >= amtWei
		k.setNibiBalanceFromWei(ctx, addr,
			new(uint256.Int).Sub(balPre, amtWei),
		)
	}

	weiBlockDelta := k.weiBlockDelta.GetOr(ctx, sdkmath.ZeroInt())
	k.weiBlockDelta.Set(
		ctx,
		weiBlockDelta.Sub(sdkmath.NewIntFromBigInt(amtWei.ToBig())),
	)

	ctx.EventManager().EmitEvent(
		bank.NewEventWeiChange(
			bank.WeiChangeReason_SubWei,
			addr.String(),
		),
	)
	return nil
}

// setNibiBalanceFromWei decomposes a wei amount into unibi (bank) and wei
// (store) components. This function maintains the dual-balance model for NIBI:
//
//	```
//	total_wei (NIBI) = (bank_unibi * 10^{12}) + wei_store.
//	```
//
// Normalizes so wei_store ∈ [0, 10^12) and bank stores the remainder as unibi
// coins.
//
// Args:
//   - ctx: SDK context for state operations
//   - addr: Account address to update
//   - wei: Total wei amount (nil/zero → zero balance)
//
// Note: Internal function. Use AddWei/SubWei for proper event emission and delta tracking.
func (k BaseSendKeeper) setNibiBalanceFromWei(
	ctx sdk.Context, addr sdk.AccAddress, wei *uint256.Int,
) {
	if wei == nil || wei.Cmp(nutil.WeiPerUnibiU256()) < 0 {
		// Error on setBalance with a zero coin is impossible
		_ = k.setBalance(ctx, addr, sdk.NewInt64Coin(DENOM_UNIBI, 0))
		k.setWeiStoreBalance(ctx, addr, wei)
		return
	}

	amtUnibi, amtWei := nutil.ParseNibiBalance(sdkmath.NewIntFromBigInt(wei.ToBig()))
	// The bank coin `sdk.NewCoin(DENOM_UNIBI, amtUnibi)`  is guaranteed to be
	// valid since it's amount is a u256 and denom is "unibi".
	_ = k.setBalance(ctx, addr, sdk.NewCoin(DENOM_UNIBI, amtUnibi))
	k.setWeiStoreBalance(ctx, addr, uint256.MustFromBig(amtWei.BigInt()))
}

func eventsForSendCoins(
	fromAddr sdk.AccAddress,
	toAddr sdk.AccAddress,
	coins sdk.Coins,
) sdk.Events {
	fromAddrStr := fromAddr.String() // bech32 encoding is expensive! Only do it once for fromAddr
	toAddrStr := toAddr.String()
	isSome, _ := coins.Find(DENOM_UNIBI)

	events := sdk.Events{
		sdk.NewEvent(
			bank.EventTypeTransfer,
			sdk.NewAttribute(bank.AttributeKeyRecipient, toAddrStr),
			sdk.NewAttribute(bank.AttributeKeySender, fromAddrStr),
			sdk.NewAttribute(sdk.AttributeKeyAmount, coins.String()),
		),
	}
	if isSome {
		events = append(events, bank.NewEventWeiChange(
			bank.WeiChangeReason_SendCoins,
			fromAddrStr, toAddrStr,
		))
	}
	events = append(events, sdk.NewEvent(
		sdk.EventTypeMessage,
		sdk.NewAttribute(bank.AttributeKeySender, fromAddrStr),
	))
	return events
}

func (k BaseSendKeeper) getWeiStoreBalance(
	ctx sdk.Context,
	addr sdk.AccAddress,
) (storeBal *uint256.Int) {
	balInt := k.weiStore.GetOr(ctx, addr, sdkmath.ZeroInt())
	return uint256.MustFromBig(balInt.BigInt())
}

func (k BaseSendKeeper) setWeiStoreBalance(
	ctx sdk.Context,
	addr sdk.AccAddress,
	newStoreBal *uint256.Int,
) {
	weiStoreBalPre := k.getWeiStoreBalance(ctx, addr)
	if weiStoreBalPre.Eq(newStoreBal) || newStoreBal == nil {
		// Early return with safety from `nil`. The store can only take
		// values that can be marshaled to and from bytes safely.
		return
	}

	if newStoreBal.Eq(uint256.NewInt(0)) {
		// Error only occurs if we "delete" a key that was not present, which is
		// not an error here.
		_ = k.weiStore.Delete(ctx, addr)
		return
	}

	k.weiStore.Insert(ctx, addr, sdkmath.NewIntFromBigInt(newStoreBal.ToBig()))
}

// WeiBlockDelta is the net sum of all calls of [AddWei] and [SubWei] in the
// current block. There is no guarantee in the functional sense that the EVM
// State DB will add the same amount it subtracts. It is possible for the total
// amount of wei (NIBI) across all accounts to diverge from the initial supply.
// [WeiBlockDelta] is a mechanism for recording that if it happens.
func (k BaseSendKeeper) WeiBlockDelta(
	ctx sdk.Context,
) sdkmath.Int {
	return k.weiBlockDelta.GetOr(ctx, sdkmath.ZeroInt())
}

// Iterates across the entire wei store, summing all of the balances to
// evaluate the [TotalSupply] [sdk.Invariant] for the crisis module. This
// function is can be heavy to run and should not be used outside of that
// invariant check.
func (k BaseSendKeeper) SumWeiStoreBals(ctx sdk.Context) sdkmath.Int {
	iter := k.weiStore.Iterate(ctx, collections.Range[sdk.AccAddress]{})
	totalStoreBalWei := sdkmath.ZeroInt()
	for _, storeBalWei := range iter.Values() {
		totalStoreBalWei = totalStoreBalWei.Add(storeBalWei)
	}
	return totalStoreBalWei
}
