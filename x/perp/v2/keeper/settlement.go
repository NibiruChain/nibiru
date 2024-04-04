package keeper

import (
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/perp/v2/types"
)

// SettlePosition settles a position and transfer the margin and funding payments to the trader.
func (k Keeper) SettlePosition(ctx sdk.Context, pair asset.Pair, version uint64, traderAddr sdk.AccAddress) (resp *types.PositionResp, err error) {
	market, err := k.GetMarketByPairAndVersion(ctx, pair, version)
	if err != nil {
		return
	}
	if market.Enabled {
		return nil, types.ErrSettlementPositionMarketEnabled
	}

	amm, err := k.GetAMMByPairAndVersion(ctx, pair, version)
	if err != nil {
		return
	}

	position, err := k.GetPosition(ctx, pair, version, traderAddr)
	if err != nil {
		return
	}

	_, positionResp, err := k.settlePosition(ctx, market, amm, position)
	if err != nil {
		return
	}

	if positionResp.BadDebt.IsPositive() {
		if err = k.realizeBadDebt(
			ctx,
			market,
			positionResp.BadDebt.RoundInt(),
		); err != nil {
			return nil, err
		}
	}

	if err = k.afterPositionUpdate(
		ctx,
		market,
		traderAddr,
		*positionResp,
		types.ChangeReason_Settlement,
		sdkmath.ZeroInt(),
		position,
	); err != nil {
		return nil, err
	}

	return positionResp, nil
}

// Settles a position and realizes PnL and funding payments.
// Returns the updated AMM and the realized PnL and funding payments.
func (k Keeper) settlePosition(ctx sdk.Context, market types.Market, amm types.AMM, position types.Position) (updatedAMM *types.AMM, resp *types.PositionResp, err error) {
	positionNotional := position.Size_.Abs().Mul(amm.SettlementPrice)

	resp = &types.PositionResp{
		ExchangedPositionSize: position.Size_.Neg(),
		PositionNotional:      sdkmath.LegacyZeroDec(),
		FundingPayment:        FundingPayment(position, market.LatestCumulativePremiumFraction),
		RealizedPnl:           UnrealizedPnl(position, positionNotional),
		UnrealizedPnlAfter:    sdkmath.LegacyZeroDec(),
	}

	remainingMargin := position.Margin.Add(resp.RealizedPnl).Sub(resp.FundingPayment)

	if remainingMargin.IsPositive() {
		resp.BadDebt = sdkmath.LegacyZeroDec()
		resp.MarginToVault = remainingMargin.Neg()
	} else {
		resp.BadDebt = remainingMargin.Abs()
		resp.MarginToVault = sdkmath.LegacyZeroDec()
	}

	var dir types.Direction
	// flipped since we are going against the current position
	if position.Size_.IsPositive() {
		dir = types.Direction_SHORT
	} else {
		dir = types.Direction_LONG
	}
	updatedAMM, exchangedNotionalValue, err := k.SwapBaseAsset(
		ctx,
		amm,
		dir,
		position.Size_.Abs(),
		sdkmath.LegacyZeroDec(),
	)
	if err != nil {
		return nil, nil, err
	}

	resp.ExchangedNotionalValue = exchangedNotionalValue
	resp.Position = types.Position{
		TraderAddress:                   position.TraderAddress,
		Pair:                            position.Pair,
		Size_:                           sdkmath.LegacyZeroDec(),
		Margin:                          sdkmath.LegacyZeroDec(),
		OpenNotional:                    sdkmath.LegacyZeroDec(),
		LatestCumulativePremiumFraction: market.LatestCumulativePremiumFraction,
		LastUpdatedBlockNumber:          ctx.BlockHeight(),
	}

	err = k.DeletePosition(ctx, position.Pair, market.Version, sdk.MustAccAddressFromBech32(position.TraderAddress))
	if err != nil {
		return nil, nil, err
	}

	return updatedAMM, resp, nil
}
