package simulation

import (
	sdkmath "cosmossdk.io/math"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/spot/keeper"
	"github.com/NibiruChain/nibiru/x/spot/types"
)

const defaultWeight = 100

// WeightedOperations returns all the operations from the module with their respective weights
func WeightedOperations(
	ak types.AccountKeeper,
	bk types.BankKeeper,
	k keeper.Keeper) simulation.WeightedOperations {
	return simulation.WeightedOperations{
		simulation.NewWeightedOperation(
			defaultWeight,
			SimulateMsgCreatePool(ak, bk, k),
		),
		simulation.NewWeightedOperation(
			defaultWeight,
			SimulateMsgSwap(ak, bk, k),
		),
		simulation.NewWeightedOperation(
			defaultWeight,
			SimulateJoinPool(ak, bk, k),
		),
		simulation.NewWeightedOperation(
			defaultWeight,
			SimulateExitPool(ak, bk, k),
		),
	}
}

// SimulateMsgCreatePool generates a MsgCreatePool with random values.
func SimulateMsgCreatePool(ak types.AccountKeeper, bk types.BankKeeper, k keeper.Keeper) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		params := k.GetParams(ctx)

		fundAccountWithTokens(ctx, simAccount.Address, bk)
		spendableCoins := bk.SpendableCoins(ctx, simAccount.Address)

		whitelistedAssets := params.GetWhitelistedAssetsAsMap()

		poolAssets := genPoolAssets(r, spendableCoins, whitelistedAssets)
		poolParams := genBalancerPoolParams(r, ctx.BlockTime(), poolAssets)

		// set the pool params to set the pool creation fee to dust amount of denom
		params.PoolCreationFee = sdk.Coins{sdk.NewInt64Coin(spendableCoins[0].Denom, 1)}
		k.SetParams(ctx, params)

		msg := &types.MsgCreatePool{
			Creator:    simAccount.Address.String(),
			PoolParams: &poolParams,
			PoolAssets: poolAssets,
		}
		_, err := k.FetchPoolFromPair(ctx, poolAssets[0].Token.Denom, poolAssets[1].Token.Denom)
		if err == nil {
			// types.ErrPoolWithSameAssetsExists
			return simtypes.NoOpMsg(types.ModuleName, msg.Type(), "pool already exists for these tokens"), nil, nil
		}

		return simulation.GenAndDeliverTxWithRandFees(
			simulation.OperationInput{
				R:               r,
				App:             app,
				TxGen:           simapp.MakeTestEncodingConfig().TxConfig,
				Cdc:             nil,
				Msg:             msg,
				MsgType:         msg.Type(),
				Context:         ctx,
				SimAccount:      simAccount,
				AccountKeeper:   ak,
				Bankkeeper:      bk,
				ModuleName:      types.ModuleName,
				CoinsSpentInMsg: PoolAssetsCoins(poolAssets),
			},
		)
	}
}

/*
SimulateMsgSwap generates a MsgSwap with random values
*/
func SimulateMsgSwap(ak types.AccountKeeper, bk types.BankKeeper, k keeper.Keeper) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		msg := &types.MsgSwapAssets{}

		simAccount, _ := simtypes.RandomAcc(r, accs)
		fundAccountWithTokens(ctx, simAccount.Address, bk)
		spendableCoins := bk.SpendableCoins(ctx, simAccount.Address)

		denomIn, denomOut, poolId, balanceIn := findRandomPoolWithDenom(ctx, r, spendableCoins, k)

		if denomIn == "" {
			return simtypes.NoOpMsg(types.ModuleName, msg.Type(), "No pool existing yet for account tokens"), nil, nil
		}
		if balanceIn.IsZero() {
			return simtypes.NoOpMsg(types.ModuleName, msg.Type(), "No tokens to swap in"), nil, nil
		}

		tokenIn := sdk.NewCoin(denomIn, balanceIn)
		msg = &types.MsgSwapAssets{
			Sender:        simAccount.Address.String(),
			PoolId:        poolId,
			TokenIn:       tokenIn,
			TokenOutDenom: denomOut,
		}
		pool, _ := k.FetchPool(ctx, poolId)
		_, _, err := pool.CalcOutAmtGivenIn(tokenIn, denomOut, false)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, msg.Type(), "pool imbalanced and not enough swap amount"), nil, nil
		}

		return simulation.GenAndDeliverTxWithRandFees(
			simulation.OperationInput{
				R:               r,
				App:             app,
				TxGen:           simapp.MakeTestEncodingConfig().TxConfig,
				Cdc:             nil,
				Msg:             msg,
				MsgType:         msg.Type(),
				Context:         ctx,
				SimAccount:      simAccount,
				AccountKeeper:   ak,
				Bankkeeper:      bk,
				ModuleName:      types.ModuleName,
				CoinsSpentInMsg: sdk.NewCoins(tokenIn),
			},
		)
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
		msg := &types.MsgJoinPool{}
		// run only 1/3 of the time
		if simtypes.RandomDecAmount(r, sdk.MustNewDecFromStr("1")).GTE(sdk.MustNewDecFromStr("0.33")) {
			return simtypes.NoOpMsg(types.ModuleName, msg.Type(), "No join pool done"), nil, nil
		}

		simAccount, _ := simtypes.RandomAcc(r, accs)
		fundAccountWithTokens(ctx, simAccount.Address, bk)
		spendableCoins := bk.SpendableCoins(ctx, simAccount.Address)

		pool, err, index1, index2 := findRandomPoolWithDenomPair(ctx, r, spendableCoins, k)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, msg.Type(), "No pool existing yet for tokens in account"), nil, nil
		}

		intensityFactorToken0 := simtypes.RandomDecAmount(r, sdk.MustNewDecFromStr("0.499")).Add(sdk.MustNewDecFromStr("0.5"))
		intensityFactorToken1 := simtypes.RandomDecAmount(r, sdk.MustNewDecFromStr("0.499")).Add(sdk.MustNewDecFromStr("0.5"))

		tokensIn := sdk.NewCoins(
			sdk.NewCoin(
				pool.PoolAssets[0].Token.Denom,
				intensityFactorToken0.Mul(sdk.NewDecFromInt(spendableCoins[index1].Amount)).TruncateInt()),
			sdk.NewCoin(
				pool.PoolAssets[1].Token.Denom,
				intensityFactorToken1.Mul(sdk.NewDecFromInt(spendableCoins[index2].Amount)).TruncateInt()),
		)

		msg = &types.MsgJoinPool{
			Sender:   simAccount.Address.String(),
			PoolId:   pool.Id,
			TokensIn: tokensIn,
		}

		_, err = pool.GetD(pool.PoolAssets)
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, msg.Type(), "borked pool"), nil, nil
		}

		return simulation.GenAndDeliverTxWithRandFees(
			simulation.OperationInput{
				R:               r,
				App:             app,
				TxGen:           simapp.MakeTestEncodingConfig().TxConfig,
				Cdc:             nil,
				Msg:             msg,
				MsgType:         msg.Type(),
				Context:         ctx,
				SimAccount:      simAccount,
				AccountKeeper:   ak,
				Bankkeeper:      bk,
				ModuleName:      types.ModuleName,
				CoinsSpentInMsg: tokensIn,
			},
		)
	}
}

/*
SimulateExitPool generates a MsgExitPool with random values
This function has a 33% chance of swapping a random fraction of the balance of a random token
*/
func SimulateExitPool(ak types.AccountKeeper, bk types.BankKeeper, k keeper.Keeper) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (opMsg simtypes.OperationMsg, futureOp []simtypes.FutureOperation, err error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		spendableCoins := bk.SpendableCoins(ctx, simAccount.Address)

		// Search for LP tokens in sim coins
		randomIndices := r.Perm(spendableCoins.Len())
		var shareTokens sdk.Coin

		for _, index := range randomIndices {
			coin := spendableCoins[index]
			if strings.Contains(coin.Denom, "nibiru/pool/") {
				shareTokens = coin
				break
			}
		}
		msg := &types.MsgExitPool{}

		if shareTokens.Denom == "" {
			return simtypes.NoOpMsg(types.ModuleName, msg.Type(), "No pool share token found in wallet"), nil, nil
		}

		intensityFactor := simtypes.RandomDecAmount(r, sdk.MustNewDecFromStr("0.499")).Add(sdk.MustNewDecFromStr("0.5"))
		shareTokensIn := sdk.NewCoin(
			shareTokens.Denom,
			intensityFactor.MulInt(shareTokens.Amount).TruncateInt(),
		)

		// Ugly but does the job
		poolId := sdk.MustNewDecFromStr(strings.Replace(shareTokensIn.Denom, "nibiru/pool/", "", 1)).TruncateInt().Uint64()

		// check if there are enough tokens to withdraw
		pool, err := k.FetchPool(ctx, poolId)
		if err != nil {
			return opMsg, futureOp, err
		}
		tokensOut, _, err := pool.TokensOutFromPoolSharesIn(shareTokensIn.Amount)
		if err != nil {
			return opMsg, futureOp, err
		}

		// this is necessary, as invalid tokens will be considered as wrong inputs in simulations
		if !tokensOut.IsValid() {
			return simtypes.NoOpMsg(types.ModuleName, msg.Type(), "not enough pool tokens to exit pool"), nil, nil
		}

		msg = &types.MsgExitPool{
			Sender:     simAccount.Address.String(),
			PoolId:     poolId,
			PoolShares: shareTokensIn,
		}

		return simulation.GenAndDeliverTxWithRandFees(
			simulation.OperationInput{
				R:               r,
				App:             app,
				TxGen:           simapp.MakeTestEncodingConfig().TxConfig,
				Cdc:             nil,
				Msg:             msg,
				MsgType:         msg.Type(),
				Context:         ctx,
				SimAccount:      simAccount,
				AccountKeeper:   ak,
				Bankkeeper:      bk,
				ModuleName:      types.ModuleName,
				CoinsSpentInMsg: sdk.NewCoins(shareTokensIn),
			},
		)
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
// The pool has 50% chance of being a stableswap pool.
func genBalancerPoolParams(r *rand.Rand, blockTime time.Time, assets []types.PoolAsset) types.PoolParams {
	// swapFeeInt := int64(r.Intn(1e5))
	// swapFee := sdk.NewDecWithPrec(swapFeeInt, 6)

	exitFeeInt := int64(r.Intn(1e5))
	exitFee := sdk.NewDecWithPrec(exitFeeInt, 6)
	isBalancer := r.Intn(2)

	var poolType types.PoolType
	if isBalancer == 0 {
		poolType = types.PoolType_BALANCER
	} else {
		poolType = types.PoolType_STABLESWAP
	}

	A := sdk.NewInt(int64(r.Intn(4_000) + 1))

	// Create swap fee between 0% and 5%
	swapFeeFloat := r.Float64() * .05
	swapFee := sdk.MustNewDecFromStr(fmt.Sprintf("%.5f", swapFeeFloat))

	return types.PoolParams{
		SwapFee:  swapFee,
		ExitFee:  exitFee,
		PoolType: poolType,
		A:        A,
	}
}

// genPoolAssets creates a pool asset object based on current balance of the account
func genPoolAssets(
	r *rand.Rand,
	coins sdk.Coins,
	whitelistedAssets map[string]bool,
) []types.PoolAsset {
	denomIndices := r.Perm(coins.Len())
	var assets []types.PoolAsset

	for _, denomIndex := range denomIndices {
		denom := coins[denomIndex].Denom

		if _, ok := whitelistedAssets[denom]; ok {
			amt, _ := simtypes.RandPositiveInt(r, coins[denomIndex].Amount.QuoRaw(10))
			reserveAmt := sdk.NewCoin(denom, amt)

			// Weight is useless for stableswap pools.
			weight := sdk.NewInt(r.Int63n(9) + 1)
			assets = append(assets, types.PoolAsset{Token: reserveAmt, Weight: weight})

			if len(assets) == 2 {
				return assets
			}
		}
	}

	panic("amm pool must have 2 assets")
}

// fundAccountWithTokens fund the account with some gov, coll and stable denom.
// when simulation for stablecoin is done, we should consider only funding with stable.
func fundAccountWithTokens(ctx sdk.Context, address sdk.AccAddress, bk types.BankKeeper) {
	million := 1 * common.TO_MICRO
	newTokens := sdk.NewCoins(
		sdk.NewCoin(denoms.NIBI, sdk.NewInt(int64(10*million))),
		sdk.NewCoin(denoms.USDC, sdk.NewInt(int64(10*million))),
		sdk.NewCoin(denoms.NUSD, sdk.NewInt(int64(10*million))),
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
func findRandomPoolWithDenom(ctx sdk.Context, r *rand.Rand, spendableCoins sdk.Coins, k keeper.Keeper) (
	denomIn string, denomOut string, poolId uint64, balanceIn sdkmath.Int) {
	randomIndices := r.Perm(spendableCoins.Len())
	whitelistedAssets := k.GetParams(ctx).GetWhitelistedAssetsAsMap()

	pools := k.FetchAllPools(ctx)
	for _, index := range randomIndices {
		coin := spendableCoins[index]
		if _, ok := whitelistedAssets[coin.Denom]; ok {
			for _, pool := range pools {
				if pool.PoolAssets[0].Token.Denom == coin.Denom {
					return coin.Denom, pool.PoolAssets[1].Token.Denom, pool.Id, spendableCoins[index].Amount
				} else if pool.PoolAssets[1].Token.Denom == coin.Denom {
					return coin.Denom, pool.PoolAssets[0].Token.Denom, pool.Id, spendableCoins[index].Amount
				}
			}
		}
	}

	return "", "", 0, sdk.ZeroInt()
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
