package keeper

import (
	sdkmath "cosmossdk.io/math"
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

	// Allocate minted coins according to allocation proportions (staking, strategic, community pool)
	return k.AllocatePolynomialInflation(ctx, coins, params)
}

// MintCoins implements an alias call to the underlying supply keeper's
// MintCoins to be used in BeginBlocker.
func (k Keeper) MintCoins(ctx sdk.Context, coin sdk.Coin) error {
	coins := sdk.Coins{coin}
	return k.bankKeeper.MintCoins(ctx, types.ModuleName, coins)
}

// AllocatePolynomialInflation allocates coins from the inflation to external
// modules according to allocation proportions:
//   - staking rewards -> sdk `auth` module fee collector
//   - strategic reserves -> root account of x/sudo module
//   - community pool -> `sdk `distr` module community pool
func (k Keeper) AllocatePolynomialInflation(
	ctx sdk.Context,
	mintedCoin sdk.Coin,
	params types.Params,
) (
	staking, strategic, community sdk.Coin,
	err error,
) {
	inflationDistribution := params.InflationDistribution
	inflationModuleAddr := k.accountKeeper.GetModuleAddress(types.ModuleName)
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
		inflationModuleAddr,
	); err != nil {
		return sdk.Coin{}, sdk.Coin{}, sdk.Coin{}, err
	}

	// Remaining balance is strategic reserve allocation to the root account of the x/sudo module
	strategic = k.bankKeeper.GetBalance(ctx, inflationModuleAddr, denoms.NIBI)
	strategicAccountAddr, err := k.sudoKeeper.GetRoot(ctx)
	if err != nil {
		k.Logger(ctx).Error("get root account error", "error", err)
		return staking, strategic, community, nil
	}

	if err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, strategicAccountAddr, sdk.NewCoins(strategic)); err != nil {
		k.Logger(ctx).Error("send coins to root account error", "error", err)
		return sdk.Coin{}, sdk.Coin{}, sdk.Coin{}, nil
	}

	_ = ctx.EventManager().EmitTypedEvents(
		&types.InflationDistributionEvent{
			StakingRewards:   staking,
			StrategicReserve: strategic,
			CommunityPool:    community,
		},
	)

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
func (k Keeper) GetCirculatingSupply(ctx sdk.Context, mintDenom string) sdkmath.Int {
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
	circulatingSupplyToDec := sdk.NewDecFromInt(circulatingSupply)
	return epochMintProvision.
		MulInt64(int64(k.EpochsPerPeriod(ctx))).
		MulInt64(int64(k.PeriodsPerYear(ctx))).
		Quo(circulatingSupplyToDec).
		Mul(sdk.NewDec(100))
}

// GetEpochMintProvision retrieves necessary params KV storage
// and calculate EpochMintProvision
func (k Keeper) GetEpochMintProvision(ctx sdk.Context) sdk.Dec {
	peek := k.CurrentPeriod.Peek(ctx)

	return types.CalculateEpochMintProvision(
		k.GetParams(ctx),
		peek,
	)
}
