package keeper

import (
	"fmt"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	types "github.com/NibiruChain/nibiru/x/perp/v2/types"
)

// Admin is syntactic sugar to separate admin calls off from the other Keeper
// methods.
//
// These Admin functions should:
// 1. Not be wired into the MsgServer or
// 2. Not be called in other methods in the x/perp module.
// 3. Only be callable from x/wasm/binding via sudo contracts.
//
// The intention here is to make it more obvious to the developer that an unsafe
// is being used when it's called as a function on the Admin() struct.
func (k Keeper) Admin() admin {
	return admin{&k}
}

// Extends the Keeper with admin functions.
type admin struct{ *Keeper }

/*
WithdrawFromInsuranceFund sends funds from the Insurance Fund to the given "to"
address.

Args:
- ctx: Blockchain context holding the current state
- amount: Amount of micro-NUSD to withdraw.
- to: Recipient address
*/
func (k admin) WithdrawFromInsuranceFund(
	ctx sdk.Context, amount sdkmath.Int, to sdk.AccAddress,
) (err error) {
	coinToSend := sdk.NewCoin(denoms.NUSD, amount)
	if err = k.BankKeeper.SendCoinsFromModuleToAccount(
		ctx,
		/* from */ types.PerpEFModuleAccount,
		/* to */ to,
		/* amount */ sdk.NewCoins(coinToSend),
	); err != nil {
		return err
	}
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"withdraw_from_if",
		sdk.NewAttribute("to", to.String()),
		sdk.NewAttribute("funds", coinToSend.String()),
	))
	return nil
}

func (k admin) SetMarketEnabled(
	ctx sdk.Context, pair asset.Pair, enabled bool,
) (err error) {
	market, err := k.Markets.Get(ctx, pair)
	if err != nil {
		return
	}
	market.Enabled = enabled
	k.Markets.Insert(ctx, pair, market)
	return
}

type ArgsCreateMarket struct {
	Pair            asset.Pair
	PriceMultiplier sdk.Dec
	SqrtDepth       sdk.Dec
	Market          *types.Market // pointer makes it optional
}

// CreateMarket creates a pool for a specific pair.
func (k admin) CreateMarket(
	ctx sdk.Context,
	args ArgsCreateMarket,
) error {
	pair := args.Pair
	_, err := k.Markets.Get(ctx, pair)
	if err == nil {
		return fmt.Errorf("market %s already exists", pair)
	}

	// init market
	sqrtDepth := args.SqrtDepth
	quoteReserve := sqrtDepth
	baseReserve := sqrtDepth
	var market *types.Market
	if args.Market == nil {
		*market = types.DefaultMarket(pair)
	} else {
		market = args.Market
	}
	if err := market.Validate(); err != nil {
		return err
	}

	// init amm
	amm := types.AMM{
		Pair:            pair,
		BaseReserve:     baseReserve,
		QuoteReserve:    quoteReserve,
		SqrtDepth:       sqrtDepth,
		PriceMultiplier: args.PriceMultiplier,
		TotalLong:       sdk.ZeroDec(),
		TotalShort:      sdk.ZeroDec(),
	}
	if err := amm.Validate(); err != nil {
		return err
	}

	k.Markets.Insert(ctx, pair, *market)
	k.AMMs.Insert(ctx, pair, amm)

	return nil
}
