package stablecoin

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/stablecoin/keeper"
)

// EndBlocker updates the current oracle
func EndBlocker(ctx sdk.Context, k keeper.Keeper) {
	if !k.GetParams(ctx).IsCollateralRatioValid {
		// Try to re-start the collateral ratio updates
		err := k.EvaluateCollRatio(ctx)

		params := k.GetParams(ctx)
		params.IsCollateralRatioValid = (err == nil)

		k.SetParams(ctx, params)
	}

	_, err := k.OracleKeeper.GetExchangeRateTwap(ctx, common.Pair_USDC_NUSD.String())

	fmt.Println("Price")
	fmt.Println(k.OracleKeeper.GetExchangeRateTwap(ctx, common.Pair_USDC_NUSD.String()))
	fmt.Println(k.OracleKeeper.GetExchangeRate(ctx, common.Pair_USDC_NUSD.String()))

	if err != nil {
		params := k.GetParams(ctx)
		params.IsCollateralRatioValid = false

		k.SetParams(ctx, params)
	}
}
