package keeper

import (
	"fmt"
	"time"

	"github.com/NibiruChain/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/asset"
	v2types "github.com/NibiruChain/nibiru/x/perp/types/v2"
)

var (
	// Temporary - avoids static check error for unused functions
	_ = Keeper.transferFee
	_ = checkOpenPositionRequirements
	_ = Keeper.checkPriceFluctuationLimitRatio
)

// checkOpenPositionRequirements checks the minimum requirements to open a position.
//
// - Checks that quote asset is not zero.
// - Checks that leverage is not zero.
// - Checks that leverage is below requirement.
//
// args:
// - market: the market where the position will be opened
// - quoteAssetAmt: the amount of quote asset
// - leverage: the amount of leverage to take, as sdk.Dec
//
// returns:
// - error: if any of the requirements is not met
func checkOpenPositionRequirements(market v2types.Market, quoteAssetAmt sdk.Int, leverage sdk.Dec) error {
	if quoteAssetAmt.IsZero() {
		return v2types.ErrQuoteAmountIsZero
	}

	if leverage.IsZero() {
		return v2types.ErrLeverageIsZero
	}

	if leverage.GT(market.MaxLeverage) {
		return v2types.ErrLeverageIsTooHigh
	}

	return nil
}

// transfers the fee to the exchange fee pool and ecosystem fund
//
// args:
// - ctx: the cosmos-sdk context
// - pair: the trading pair
// - trader: the trader's address
// - positionNotional: the position's notional value
//
// returns:
// - fees: the fees to be transferred
// - err: error if any
func (k Keeper) transferFee(
	ctx sdk.Context,
	pair asset.Pair,
	trader sdk.AccAddress,
	positionNotional sdk.Dec,
) (fees sdk.Int, err error) {
	m, err := k.Markets.Get(ctx, pair)
	if err != nil {
		return sdk.Int{}, err
	}

	feeToExchangeFeePool := m.ExchangeFeeRatio.Mul(positionNotional).RoundInt()
	if feeToExchangeFeePool.IsPositive() {
		if err = k.BankKeeper.SendCoinsFromAccountToModule(
			ctx,
			/* from */ trader,
			/* to */ v2types.FeePoolModuleAccount,
			/* coins */ sdk.NewCoins(
				sdk.NewCoin(
					pair.QuoteDenom(),
					feeToExchangeFeePool,
				),
			),
		); err != nil {
			return sdk.Int{}, err
		}
	}

	feeToEcosystemFund := m.EcosystemFundFeeRatio.Mul(positionNotional).RoundInt()
	if feeToEcosystemFund.IsPositive() {
		if err = k.BankKeeper.SendCoinsFromAccountToModule(
			ctx,
			/* from */ trader,
			/* to */ v2types.PerpEFModuleAccount,
			/* coins */ sdk.NewCoins(
				sdk.NewCoin(
					pair.QuoteDenom(),
					feeToEcosystemFund,
				),
			),
		); err != nil {
			return sdk.Int{}, err
		}
	}

	return feeToExchangeFeePool.Add(feeToEcosystemFund), nil
}

// checks that the mark price of the pool does not violate the fluctuation limit
//
// args:
//   - ctx: the cosmos-sdk context
//   - market: the perp market
//   - amm: the amm reserves
//
// returns:
//   - err: error if any
func (k Keeper) checkPriceFluctuationLimitRatio(ctx sdk.Context, market v2types.Market, amm v2types.AMM) error {
	if market.PriceFluctuationLimitRatio.IsZero() {
		// early return to avoid expensive state operations
		return nil
	}

	it := k.ReserveSnapshots.Iterate(ctx, collections.PairRange[asset.Pair, time.Time]{}.Prefix(amm.Pair).Descending())
	defer it.Close()

	if !it.Valid() {
		return fmt.Errorf("error getting last snapshot number for pair %s", amm.Pair)
	}

	snapshotMarkPrice := it.Value().Amm.MarkPrice()
	snapshotUpperLimit := snapshotMarkPrice.Mul(sdk.OneDec().Add(market.PriceFluctuationLimitRatio))
	snapshotLowerLimit := snapshotMarkPrice.Mul(sdk.OneDec().Sub(market.PriceFluctuationLimitRatio))

	if amm.MarkPrice().GT(snapshotUpperLimit) || snapshotMarkPrice.LT(snapshotLowerLimit) {
		return v2types.ErrOverFluctuationLimit
	}

	return nil
}
