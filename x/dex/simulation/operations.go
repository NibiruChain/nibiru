package simulation

import (
	"math/rand"
	"strings"
	"time"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/dex/keeper"
	"github.com/NibiruChain/nibiru/x/dex/types"
	simulation "github.com/NibiruChain/nibiru/x/simulation"
)

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

		if simCoins.Len() <= 1 {
			fundAccountWithTokens(ctx, simAccount.Address, bk)
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

/*
	SimulateMsgSwap generates a MsgSwap with random values
	This function has a 33% chance of swapping a random fraction of the balance of a random token
*/
func SimulateMsgSwap(ak types.AccountKeeper, bk types.BankKeeper, k keeper.Keeper) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		simCoins := bk.SpendableCoins(ctx, simAccount.Address)

		if simCoins.Len() <= 1 {
			fundAccountWithTokens(ctx, simAccount.Address, bk)
			simCoins = bk.SpendableCoins(ctx, simAccount.Address)
		}
		msg := &types.MsgSwapAssets{}

		denomIn, denomOut, poolIdSwap, balanceIn := findRandomPoolWithDenom(ctx, r, simCoins, k)

		if denomIn == "" {
			return simtypes.NoOpMsg(
				types.ModuleName, msg.Type(), "No pool existing yet for account tokens"), nil, nil
		}

		intensityFactor := simtypes.RandomDecAmount(r, sdk.MustNewDecFromStr("0.01")).Add(sdk.MustNewDecFromStr("0.05"))
		frequencyFactor := simtypes.RandomDecAmount(r, sdk.MustNewDecFromStr("1"))

		intensity := intensityFactor.Mul(sdk.NewDecFromInt(balanceIn)).TruncateInt()

		if frequencyFactor.GTE(sdk.MustNewDecFromStr("0.33")) {
			return simtypes.NoOpMsg(
				types.ModuleName, msg.Type(), "No swap done"), nil, nil
		}

		msg = &types.MsgSwapAssets{
			Sender:        simAccount.Address.String(),
			PoolId:        poolIdSwap,
			TokenIn:       sdk.NewCoin(denomIn, intensity),
			TokenOutDenom: denomOut,
		}

		txGen := simapp.MakeTestEncodingConfig().TxConfig
		return simulation.GenAndDeliverTxWithRandFees(
			/*r*/ r,
			/*app*/ app,
			/*txGen*/ txGen,
			/*msg*/ msg,
			/*coinsSpentInMsg*/ sdk.NewCoins(sdk.NewCoin(denomIn, intensity)),
			/*ctx*/ ctx,
			/*simAccount*/ simAccount,
			/*ak*/ ak,
			/*bk*/ bk,
			/*moduleName*/ types.ModuleName)
	}
}

/*
	SimulateJoinPool generates a MsgJoinPool with random values
	This function has a 33% chance of swapping a random fraction of the balance of a random token
*/
func SimulateJoinPool(ak types.AccountKeeper, bk types.BankKeeper, k keeper.Keeper) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		simCoins := bk.SpendableCoins(ctx, simAccount.Address)

		if simCoins.Len() <= 1 {
			fundAccountWithTokens(ctx, simAccount.Address, bk)
			simCoins = bk.SpendableCoins(ctx, simAccount.Address)
		}

		msg := &types.MsgJoinPool{
			Sender: simAccount.Address.String(),
		}
		pool, err, index1, index2 := findRandomPoolWithDenomPair(ctx, r, simCoins, k)
		if err != nil {
			return simtypes.NoOpMsg(
				types.ModuleName, msg.Type(), "No pool existing yet for tokens in account"), nil, nil
		}

		frequencyFactor := simtypes.RandomDecAmount(r, sdk.MustNewDecFromStr("1"))

		intensityFactorToken0 := simtypes.RandomDecAmount(r, sdk.MustNewDecFromStr("0.499")).Add(sdk.MustNewDecFromStr("0.5"))
		intensityFactorToken1 := simtypes.RandomDecAmount(r, sdk.MustNewDecFromStr("0.499")).Add(sdk.MustNewDecFromStr("0.5"))

		tokensIn := sdk.NewCoins(
			sdk.NewCoin(
				pool.PoolAssets[0].Token.Denom,
				intensityFactorToken0.Mul(sdk.NewDecFromInt(simCoins[index1].Amount)).TruncateInt()),

			sdk.NewCoin(
				pool.PoolAssets[1].Token.Denom,
				intensityFactorToken1.Mul(sdk.NewDecFromInt(simCoins[index2].Amount)).TruncateInt()),
		)

		if frequencyFactor.GTE(sdk.MustNewDecFromStr("0.33")) {
			return simtypes.NoOpMsg(
				types.ModuleName, msg.Type(), "No join pool done"), nil, nil
		}

		msg = &types.MsgJoinPool{
			Sender:   simAccount.Address.String(),
			PoolId:   pool.Id,
			TokensIn: tokensIn}

		txGen := simapp.MakeTestEncodingConfig().TxConfig

		return simulation.GenAndDeliverTxWithRandFees(
			/*r*/ r,
			/*app*/ app,
			/*txGen*/ txGen,
			/*msg*/ msg,
			/*coinsSpentInMsg*/ tokensIn,
			/*ctx*/ ctx,
			/*simAccount*/ simAccount,
			/*ak*/ ak,
			/*bk*/ bk,
			/*moduleName*/ types.ModuleName)
	}
}

/*
	SimulateExitPool generates a MsgExitPool with random values
	This function has a 33% chance of swapping a random fraction of the balance of a random token
*/
func SimulateExitPool(ak types.AccountKeeper, bk types.BankKeeper, k keeper.Keeper) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		simCoins := bk.SpendableCoins(ctx, simAccount.Address)

		// Search for pool in sim coins
		randomIndices := r.Perm(simCoins.Len())
		var shareTokenOut sdk.Coin

		for _, index := range randomIndices {
			coin := simCoins[index]
			if strings.Contains(coin.Denom, "nibiru/pool/") {
				shareTokenOut = coin
				break
			}
		}
		msg := &types.MsgExitPool{}

		if shareTokenOut.Denom == "" {
			return simtypes.NoOpMsg(
				types.ModuleName, msg.Type(), "No pool share token found in wallet"), nil, nil
		}

		intensityFactor := simtypes.RandomDecAmount(r, sdk.MustNewDecFromStr("0.499")).Add(sdk.MustNewDecFromStr("0.5"))

		tokenOut := sdk.NewCoin(
			shareTokenOut.Denom,
			intensityFactor.Mul(sdk.NewDecFromInt(shareTokenOut.Amount)).TruncateInt(),
		)

		// Ugly but does the job
		poolId := uint64(sdk.MustNewDecFromStr(strings.Replace(tokenOut.Denom, "nibiru/pool/", "", 1)).TruncateInt().Int64())

		msg = &types.MsgExitPool{
			Sender:     simAccount.Address.String(),
			PoolId:     poolId,
			PoolShares: tokenOut}

		txGen := simapp.MakeTestEncodingConfig().TxConfig

		return simulation.GenAndDeliverTxWithRandFees(
			/*r*/ r,
			/*app*/ app,
			/*txGen*/ txGen,
			/*msg*/ msg,
			/*coinsSpentInMsg*/ sdk.NewCoins(tokenOut),
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
			amt, _ := simtypes.RandPositiveInt(r, coins[denomIndex].Amount.QuoRaw(10))
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

// fundAccountWithTokens fund the account with some gov, coll and stable denom.
// When simulation for stablecoin is done, we should consider only funding with stable.
func fundAccountWithTokens(ctx sdk.Context, address sdk.AccAddress, bk types.BankKeeper) {
	million := 1_000_000
	newTokens := sdk.NewCoins(
		sdk.NewCoin(common.DenomGov, sdk.NewInt(int64(10*million))),
		sdk.NewCoin(common.DenomColl, sdk.NewInt(int64(10*million))),
		sdk.NewCoin(common.DenomStable, sdk.NewInt(int64(10*million))),
	)

	err := bk.MintCoins(ctx, types.ModuleName, newTokens)
	if err != nil {
		panic(err)
	}
	err = bk.SendCoinsFromModuleToAccount(
		ctx,
		types.ModuleName,
		address,
		newTokens,
	)
	if err != nil {
		panic(err)
	}
}

// findRandomPoolWithDenom search possible pool available to swap from a set of coins
func findRandomPoolWithDenom(ctx sdk.Context, r *rand.Rand, simCoins sdk.Coins, k keeper.Keeper) (
	denomIn string, denomOut string, poolId uint64, balanceIn sdk.Int) {
	randomIndices := r.Perm(simCoins.Len())
	whitelistedAssets := k.GetParams(ctx).GetWhitelistedAssetsAsMap()

	pools := k.FetchAllPools(ctx)
	for _, index := range randomIndices {
		coin := simCoins[index]
		if _, ok := whitelistedAssets[coin.Denom]; ok {
			for _, pool := range pools {
				if pool.PoolAssets[0].Token.Denom == coin.Denom {
					return coin.Denom, pool.PoolAssets[1].Token.Denom, pool.Id, simCoins[index].Amount
				} else if pool.PoolAssets[1].Token.Denom == coin.Denom {
					return coin.Denom, pool.PoolAssets[0].Token.Denom, pool.Id, simCoins[index].Amount
				}
			}
		}
	}
	return "", "", 0, sdk.NewInt(0)
}

// findRandomPoolWithDenomPair search one pool available from a pair of coins of simCoins
func findRandomPoolWithDenomPair(ctx sdk.Context, r *rand.Rand, simCoins sdk.Coins, k keeper.Keeper) (
	pool types.Pool, err error, index1 int, index2 int) {
	whitelistedAssets := k.GetParams(ctx).GetWhitelistedAssetsAsMap()
	randomIndices1 := r.Perm(simCoins.Len())
	randomIndices2 := r.Perm(simCoins.Len())

	for _, index1 := range randomIndices1 {
		coin1 := simCoins[index1]
		if _, ok := whitelistedAssets[coin1.Denom]; ok {
			for _, index2 := range randomIndices2 {
				if index1 != index2 {
					coin2 := simCoins[index2]
					if _, ok := whitelistedAssets[coin2.Denom]; ok {
						pool, err := k.FetchPoolFromPair(ctx, coin1.Denom, coin2.Denom)
						if err == nil {
							return pool, nil, index1, index2
						}
					}
				}
			}
		}
	}
	return types.Pool{}, types.ErrPoolNotFound.Wrapf("could not find pool compatible with any pair of assets"), 0, 0
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
