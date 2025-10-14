package keeper

import (
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bank "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/holiman/uint256"

	"github.com/NibiruChain/nibiru/v2/x/bank/collections"
)

var _ NibiruExtKeeper = (*BaseSendKeeper)(nil)

type NibiruExtKeeper interface {
	AddWei(ctx sdk.Context, addr sdk.AccAddress, amtWei *uint256.Int)
	GetWeiBalance(ctx sdk.Context, addr sdk.AccAddress) (bal *uint256.Int)
	SubWei(ctx sdk.Context, addr sdk.AccAddress, amtWei *uint256.Int) error

	// WeiBlockDelta is the net sum of all calls of [AddWei] and [SubWei] in
	// the current block. There is no guarantee that the EVM State DB will
	// add the same amount it subtracts. The EVM module manages the supply
	// invariant using [WeiBlockDelta] every block.
	WeiBlockDelta(ctx sdk.Context) sdkmath.Int
}

const (
	DENOM_UNIBI = "unibi"

	// NAMEPSACE_BALANCE_WEI is the store prefix for the wei balance store.
	NAMEPSACE_BALANCE_WEI     collections.Namespace = 15
	NAMESPACE_WEI_BLOCK_DELTA collections.Namespace = 16
	// NAMESPACE_WEI_SUPPLY is the store prefix for the total supply of the wei
	// balance store.
	NAMESPACE_WEI_SUPPLY collections.Namespace = 17
)

var (
	// WeiPerUnibi is a big.Int for 10^{12}. Each "unibi" (micronibi) is 10^{12}
	// wei because 1 NIBI = 10^{18} wei.
	WeiPerUnibi = sdkmath.NewIntFromBigInt(
		new(big.Int).Exp(big.NewInt(10), big.NewInt(12), nil),
	)

	// WeiPerUnibi is a uint256.Int for 10^{12}. Each "unibi" (micronibi) is 10^{12}
	// wei because 1 NIBI = 10^{18} wei.
	WeiPerUnibiU256 = uint256.MustFromBig(WeiPerUnibi.BigInt())
)

func (k BaseSendKeeper) getWeiStoreBalance(
	ctx sdk.Context,
	addr sdk.AccAddress,
) (storeBal *uint256.Int) {
	balInt := k.weiStore.GetOr(ctx, addr, sdkmath.ZeroInt())
	return uint256.MustFromBig(balInt.BigInt())
}

// TODO: UD-DEBUG: Test
func (k BaseSendKeeper) setWeiStoreBalance(
	ctx sdk.Context,
	addr sdk.AccAddress,
	newStoreBal *uint256.Int,
) {
	weiStoreBalPre := k.getWeiStoreBalance(ctx, addr)
	if weiStoreBalPre.Eq(newStoreBal) || newStoreBal == nil {
		return
	}

	if newStoreBal.Eq(uint256.NewInt(0)) {
		k.weiStore.Delete(ctx, addr)
	}

	k.weiStore.Insert(ctx, addr, sdkmath.NewIntFromBigInt(newStoreBal.ToBig()))
}

func BigToU256Safe(x *big.Int) (*uint256.Int, error) {
	if x.Sign() < 0 || x.BitLen() > 256 {
		return nil, fmt.Errorf("TODO") // out of range for uint256
	} else if x.BitLen() > 256 {
		return nil, fmt.Errorf("TODO") // out of range for uint256
	}
	// Most libs have a helper; otherwise zero-pad to 32 bytes then load.
	num, isOverflow := uint256.FromBig(x) // typically returns (*Int, ok) or (*Int, overflow)
	if isOverflow {
		return nil, fmt.Errorf("TODO") // out of range for uint256
	}
	return num, nil
}

// GetWeiBalance -> aggregate both
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
		storeBalUnibi = uint256.MustFromBig(balSdkInt.BigInt())
	}

	return uint256.MustFromBig(
		new(big.Int).Add(
			new(big.Int).Mul(storeBalUnibi.ToBig(), WeiPerUnibi.BigInt()),
			storeBalWei.ToBig(),
		),
	)
}

// TODO: UD-DEBUG: test
func ParseNibiBalance(wei sdkmath.Int) (amtUnibi, amtWei sdkmath.Int) {
	return wei.Quo(WeiPerUnibi), wei.Mod(WeiPerUnibi)
}

// TODO: UD-DEBUG: test
func ParseNibiBalanceFromParts(unibi, wei sdkmath.Int) (amtUnibi, amtWei sdkmath.Int) {
	unibiPartInWei := unibi.Mul(WeiPerUnibi)
	totalWei := unibiPartInWei.Add(wei)
	return ParseNibiBalance(totalWei)
}

// TODO: UD-DEBUG: test
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
	if newWeiStoreBal.Cmp(WeiPerUnibiU256) < 0 {
		k.setWeiStoreBalance(ctx, addr, newWeiStoreBal)
		return
	}

	balPre := k.GetWeiBalance(ctx, addr)
	k.setNibiBalanceFromWei(ctx, addr,
		new(uint256.Int).Add(balPre, amtWei),
	)
	weiBlockDelta := k.weiBlockDelta.GetOr(ctx, sdkmath.ZeroInt())
	k.weiBlockDelta.Set(
		ctx,
		weiBlockDelta.Add(sdkmath.NewIntFromBigInt(amtWei.ToBig())),
	)

	newWeiStoreSupply := k.weiStoreSupply.GetOr(ctx, sdkmath.ZeroInt()).
		Add(sdkmath.NewIntFromBigInt(amtWei.ToBig()))
	k.weiStoreSupply.Set(
		ctx, newWeiStoreSupply,
	)

	ctx.EventManager().EmitEvent(
		bank.NewEventWeiChange(
			bank.WeiChangeReason_AddWei,
			addr.String(),
		),
	)
}

// TODO: gasless Wei balance changes?
// TODO: event emission for balance changes?

func (k BaseSendKeeper) SubWei(
	ctx sdk.Context,
	addr sdk.AccAddress,
	amtWei *uint256.Int,
) error {
	if amtWei == nil || amtWei.IsZero() {
		return nil
	}

	// case: amtWei < 10^{12} and weiBalPre >= amtWei
	// weiBalPre is bounded, so we only need to check Cmp
	weiStoreBalPre := k.getWeiStoreBalance(ctx, addr)
	if weiStoreBalPre.Cmp(amtWei) >= 0 {
		k.setWeiStoreBalance(ctx, addr,
			new(uint256.Int).Sub(weiStoreBalPre, amtWei),
		)
		return nil
	}

	// case: aggBal < amtWei -> error
	balPre := k.GetWeiBalance(ctx, addr)
	if balPre.Cmp(amtWei) < 0 {
		return fmt.Errorf("TODO") // TODO: UD-DEBUG: err msg
	}

	// case (happy): aggBal >= amtWei
	k.setNibiBalanceFromWei(ctx, addr,
		new(uint256.Int).Sub(balPre, amtWei),
	)
	weiBlockDelta := k.weiBlockDelta.GetOr(ctx, sdkmath.ZeroInt())
	k.weiBlockDelta.Set(
		ctx,
		weiBlockDelta.Sub(sdkmath.NewIntFromBigInt(amtWei.ToBig())),
	)
	newWeiStoreSupply := k.weiStoreSupply.GetOr(ctx, sdkmath.ZeroInt()).
		Sub(sdkmath.NewIntFromBigInt(amtWei.ToBig()))
	k.weiStoreSupply.Set(
		ctx, newWeiStoreSupply,
	)

	ctx.EventManager().EmitEvent(
		bank.NewEventWeiChange(
			bank.WeiChangeReason_SubWei,
			addr.String(),
		),
	)
	return nil
}

// TODO: UD-DEBUG: test
func (k BaseSendKeeper) setNibiBalanceFromWei(
	ctx sdk.Context, addr sdk.AccAddress, wei *uint256.Int,
) {
	// fmt.Errorf("invalid wei amount: cannot set negative balance %s", wei)
	if wei == nil || wei.Cmp(WeiPerUnibiU256) < 0 {
		k.setBalance(ctx, addr, sdk.NewInt64Coin(DENOM_UNIBI, 0))
		k.setWeiStoreBalance(ctx, addr, wei)
	}

	amtUnibi, amtWei := ParseNibiBalance(sdkmath.NewIntFromBigInt(wei.ToBig()))
	k.setBalance(ctx, addr, sdk.NewCoin(DENOM_UNIBI, amtUnibi))
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

// WeiBlockDelta is the net sum of all calls of [AddWei] and [SubWei] in the
// current block. There is no guarantee that the EVM State DB will add the same
// amount it subtracts. The EVM module manages the supply invariant using
// [WeiBlockDelta] every block.
func (k BaseSendKeeper) WeiBlockDelta(
	ctx sdk.Context,
) sdkmath.Int {
	return k.weiBlockDelta.GetOr(ctx, sdkmath.ZeroInt())
}
