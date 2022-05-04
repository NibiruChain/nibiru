package simulation

import (
	"math/rand"
	"time"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/dex/keeper"
	"github.com/NibiruChain/nibiru/x/dex/types"
	simulation "github.com/NibiruChain/nibiru/x/simulation"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
)

func SimulateMsgCreatePool2(
	ak types.AccountKeeper,
	bk types.BankKeeper,
	k keeper.Keeper,
) simtypes.Operation {
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		msg := &types.MsgCreatePool{
			Creator: simAccount.Address.String(),
		}

		// TODO: Handling the CreatePool simulation

		return simtypes.NoOpMsg(types.ModuleName, msg.Type(), "CreatePool simulation not implemented"), nil, nil
	}
}

// SimulateMsgCreateBalancerPool generates a MsgCreatePool with random values.
func SimulateMsgCreatePool(ak types.AccountKeeper, bk types.BankKeeper, k keeper.Keeper) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		simCoins := bk.SpendableCoins(ctx, simAccount.Address)

		msg := &types.MsgCreatePool{
			Creator: simAccount.Address.String(),
		}

		million := 1_000_000
		if simCoins.Len() <= 1 {
			// Fund account with tokens
			newTokens := sdk.NewCoins(
				sdk.NewCoin(common.GovDenom, sdk.NewInt(int64(10*million))),
				sdk.NewCoin(common.CollDenom, sdk.NewInt(int64(10*million))),
				sdk.NewCoin(common.StableDenom, sdk.NewInt(int64(10*million))),
			)

			err := bk.MintCoins(ctx, types.ModuleName, newTokens)
			if err != nil {
				panic(err)
			}
			err = bk.SendCoinsFromModuleToAccount(
				ctx,
				types.ModuleName,
				simAccount.Address,
				newTokens,
			)
			if err != nil {
				panic(err)
			}

			simCoins = bk.SpendableCoins(ctx, simAccount.Address)
		}

		whitelistedAssets := k.GetParams(ctx).GetWhitelistedAssetsAsMap()

		poolAssets := genPoolAssets(r, simAccount, simCoins, whitelistedAssets)
		poolParams := genBalancerPoolParams(r, ctx.BlockTime(), poolAssets)

		balances := bk.GetAllBalances(ctx, simAccount.Address)
		denoms := make([]string, len(balances))
		for i := range balances {
			denoms[i] = balances[i].Denom
		}

		// set the pool params to set the pool creation fee to dust amount of denom
		params := k.GetParams(ctx)
		params.PoolCreationFee = sdk.Coins{sdk.NewInt64Coin(denoms[0], 1)}
		k.SetParams(ctx, params)

		msg.PoolParams = &poolParams
		msg.PoolAssets = poolAssets

		spentCoins := PoolAssetsCoins(poolAssets)

		txGen := simapp.MakeTestEncodingConfig().TxConfig
		return simulation.GenAndDeliverTxWithRandFees(
			/*r*/ r,
			/*app*/ app,
			/*txGen*/ txGen,
			/*msg*/ msg,
			/*coinsSpentInMsg*/ spentCoins,
			/*ctx*/ ctx,
			/*simAccount*/ simAccount,
			/*ak*/ ak,
			/*bk*/ bk,
			/*moduleName*/ types.ModuleName)
	}
}

// PoolAssetsCoins returns all the coins corresponding to a slice of pool assets.
func PoolAssetsCoins(assets []types.PoolAsset) sdk.Coins {
	coins := sdk.Coins{}
	for _, asset := range assets {
		coins = coins.Add(asset.Token)
	}
	return sdk.NewCoins(coins...)
}

// genBalancerPoolParams creates random parameters for the swap and exit fee of the pool
func genBalancerPoolParams(r *rand.Rand, blockTime time.Time, assets []types.PoolAsset) types.PoolParams {
	// swapFeeInt := int64(r.Intn(1e5))
	// swapFee := sdk.NewDecWithPrec(swapFeeInt, 6)

	exitFeeInt := int64(r.Intn(1e5))
	exitFee := sdk.NewDecWithPrec(exitFeeInt, 6)

	// TODO: Randomly generate LBP params
	return types.PoolParams{
		// SwapFee:                  swapFee,
		SwapFee: sdk.ZeroDec(),
		ExitFee: exitFee,
	}
}

// genPoolAssets creates a pool asset object based on current balance of the account
func genPoolAssets(
	r *rand.Rand,
	acct simtypes.Account,
	coins sdk.Coins,
	whitelistedAssets map[string]bool) []types.PoolAsset {
	denomIndices := r.Perm(coins.Len())
	assets := []types.PoolAsset{}

	for _, denomIndex := range denomIndices {
		denom := coins[denomIndex].Denom

		if _, ok := whitelistedAssets[denom]; ok {
			amt, _ := simtypes.RandPositiveInt(r, coins[denomIndex].Amount.QuoRaw(100))
			reserveAmt := sdk.NewCoin(denom, amt)
			weight := sdk.NewInt(r.Int63n(9) + 1)
			assets = append(assets, types.PoolAsset{Token: reserveAmt, Weight: weight})

			if len(assets) == 2 {
				break
			}
		}
	}

	return assets
}

func Min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func Max(x, y int) int {
	if x > y {
		return x
	}
	return y
}
