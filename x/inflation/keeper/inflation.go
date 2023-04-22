package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/inflation/types"
)

// MintAndAllocateInflation performs inflation minting and allocation
func (k Keeper) MintAndAllocateInflation(
	ctx sdk.Context,
	coins sdk.Coin,
	params types.Params,
) (
	staking, strategic, community sdk.Coin,
	err error,
) {
	// skip as no coins need to be minted
	if coins.Amount.IsNil() || !coins.Amount.IsPositive() {
		return sdk.Coin{}, sdk.Coin{}, sdk.Coin{}, nil
	}

	// Mint coins for distribution
	if err := k.MintCoins(ctx, coins); err != nil {
		return sdk.Coin{}, sdk.Coin{}, sdk.Coin{}, err
	}

	// Allocate minted coins according to allocation proportions (staking, usage
	// incentives, community pool)
	return k.AllocateExponentialInflation(ctx, coins, params)
}

// MintCoins implements an alias call to the underlying supply keeper's
// MintCoins to be used in BeginBlocker.
func (k Keeper) MintCoins(ctx sdk.Context, coin sdk.Coin) error {
	coins := sdk.Coins{coin}
	return k.bankKeeper.MintCoins(ctx, types.ModuleName, coins)
}

// AllocateExponentialInflation allocates coins from the inflation to external
// modules according to allocation proportions:
//   - staking rewards -> sdk `auth` module fee collector
//   - usage incentives -> `x/incentives` module
//   - community pool -> `sdk `distr` module community pool
func (k Keeper) AllocateExponentialInflation(
	ctx sdk.Context,
	mintedCoin sdk.Coin,
	params types.Params,
) (
	staking, strategic, community sdk.Coin,
	err error,
) {
	inflationDistribution := params.InflationDistribution
	moduleAddr := k.accountKeeper.GetModuleAddress(types.ModuleName)
	// Allocate staking rewards into fee collector account
	staking = k.GetProportions(ctx, mintedCoin, inflationDistribution.StakingRewards)

	if err := k.bankKeeper.SendCoinsFromModuleToModule(
		ctx,
		types.ModuleName,
		k.feeCollectorName,
		sdk.NewCoins(staking),
	); err != nil {
		return sdk.Coin{}, sdk.Coin{}, sdk.Coin{}, err
	}

	// Allocate community pool rewards into community pool
	community = k.GetProportions(ctx, mintedCoin, inflationDistribution.CommunityPool)

	if err = k.distrKeeper.FundCommunityPool(
		ctx,
		sdk.NewCoins(community),
		moduleAddr,
	); err != nil {
		return sdk.Coin{}, sdk.Coin{}, sdk.Coin{}, err
	}

	// Remaining balance is strategic reserve allocation
	strategic = k.bankKeeper.GetBalance(ctx, moduleAddr, denoms.NIBI)
	return staking, strategic, community, nil
}

// GetAllocationProportion calculates the proportion of coins that is to be
// allocated during inflation for a given distribution.
func (k Keeper) GetProportions(
	_ sdk.Context,
	coin sdk.Coin,
	proportion sdk.Dec,
) sdk.Coin {
	return sdk.Coin{
		Denom:  coin.Denom,
		Amount: sdk.NewDecFromInt(coin.Amount).Mul(proportion).TruncateInt(),
	}
}

// GetCirculatingSupply returns the bank supply of the mintDenom excluding the
// team allocation in the first year
func (k Keeper) GetCirculatingSupply(ctx sdk.Context, mintDenom string) sdk.Int {
	return k.bankKeeper.GetSupply(ctx, mintDenom).Amount
}

// GetInflationRate returns the inflation rate for the current period.
func (k Keeper) GetInflationRate(ctx sdk.Context, mintDenom string) sdk.Dec {
	epochMintProvision := k.GetEpochMintProvision(ctx)
	if epochMintProvision.IsZero() {
		return sdk.ZeroDec()
	}

	circulatingSupply := k.GetCirculatingSupply(ctx, mintDenom)
	if circulatingSupply.IsZero() {
		return sdk.ZeroDec()
	}

	// EpochMintProvision * 365 / circulatingSupply * 100
	return epochMintProvision.MulInt64(int64(k.EpochsPerPeriod(ctx))).Quo(circulatingSupply.ToDec()).Mul(sdk.NewDec(100))
}

// GetEpochMintProvision retrieves necessary params KV storage
// and calculate EpochMintProvision
func (k Keeper) GetEpochMintProvision(ctx sdk.Context) sdk.Dec {
	return types.CalculateEpochMintProvision(
		k.GetParams(ctx),
		k.CurrentPeriod.Peek(ctx),
	)
}
