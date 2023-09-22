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
// function is being used when it's called on the Admin() struct.
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
	market, err := k.GetMarket(ctx, pair)
	if err == nil && market.Enabled {
		return fmt.Errorf("market %s already exists and it is enabled", pair)
	}

	// init market
	sqrtDepth := args.SqrtDepth
	quoteReserve := sqrtDepth
	baseReserve := sqrtDepth
	if args.Market == nil {
		market = types.DefaultMarket(pair)
	} else {
		market = *args.Market
	}
	if err := market.Validate(); err != nil {
		return err
	}

	lastVersion := k.MarketLastVersion.GetOr(ctx, pair, types.MarketLastVersion{Version: 0})
	lastVersion.Version += 1
	market.Version = lastVersion.Version

	// init amm
	amm := types.AMM{
		Pair:            pair,
		Version:         lastVersion.Version,
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

	k.SaveMarket(ctx, market)
	k.SaveAMM(ctx, amm)
	k.MarketLastVersion.Insert(ctx, pair, lastVersion)

	return nil
}
