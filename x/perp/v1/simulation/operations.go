package simulation

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/NibiruChain/collections"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	perpammtypes "github.com/NibiruChain/nibiru/x/perp/amm/types"
	pooltypes "github.com/NibiruChain/nibiru/x/perp/amm/types"
	"github.com/NibiruChain/nibiru/x/perp/v1/keeper"
	types "github.com/NibiruChain/nibiru/x/perp/v1/types"
)

const defaultWeight = 100

// WeightedOperations returns all the operations from the module with their respective weights
func WeightedOperations(ak types.AccountKeeper, bk types.BankKeeper, k keeper.Keeper) simulation.WeightedOperations {
	return simulation.WeightedOperations{
		simulation.NewWeightedOperation(
			defaultWeight,
			SimulateMsgOpenPosition(ak, bk, k),
		),
		simulation.NewWeightedOperation(
			33,
			SimulateMsgClosePosition(ak, bk, k),
		),
		simulation.NewWeightedOperation(
			50,
			SimulateMsgAddMargin(ak, bk, k),
		),
		simulation.NewWeightedOperation(
			50,
			SimulateMsgRemoveMargin(ak, bk, k),
		),
	}
}

// SimulateMsgOpenPosition generates a MsgOpenPosition with random values.
func SimulateMsgOpenPosition(ak types.AccountKeeper, bk types.BankKeeper, k keeper.Keeper) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		errFundAccount := fundAccountWithTokens(ctx, simAccount.Address, bk)
		spendableCoins := bk.SpendableCoins(ctx, simAccount.Address)

		pools := k.PerpAmmKeeper.GetAllPools(ctx)
		pool := pools[rand.Intn(len(pools))]

		maxQuote := getMaxQuoteForPool(pool)
		quoteAmt, _ := simtypes.RandPositiveInt(r, sdk.MinInt(sdk.Int(maxQuote), spendableCoins.AmountOf(denoms.NUSD)))

		leverage := simtypes.RandomDecAmount(r, pool.Config.MaxLeverage.Sub(sdk.OneDec())).Add(sdk.OneDec()) // between [1, MaxLeverage]
		openNotional := leverage.MulInt(quoteAmt)

		var side perpammtypes.Direction
		var direction pooltypes.Direction
		if r.Float32() < .5 {
			side = perpammtypes.Direction_LONG
			direction = pooltypes.Direction_LONG
		} else {
			side = perpammtypes.Direction_SHORT
			direction = pooltypes.Direction_SHORT
		}

		feesAmt := openNotional.Mul(sdk.MustNewDecFromStr("0.002")).Ceil().TruncateInt()
		spentCoins := sdk.NewCoins(sdk.NewCoin(denoms.NUSD, quoteAmt.Add(feesAmt)))

		msg := &types.MsgOpenPosition{
			Sender:               simAccount.Address.String(),
			Pair:                 asset.Registry.Pair(denoms.BTC, denoms.NUSD),
			Side:                 side,
			QuoteAssetAmount:     quoteAmt,
			Leverage:             leverage,
			BaseAssetAmountLimit: sdk.ZeroInt(),
		}

		isOverFluctation := checkIsOverFluctation(ctx, k, pool, openNotional, direction)
		if isOverFluctation {
			return simtypes.NoOpMsg(types.ModuleName, msg.Type(), "over fluctuation limit"), nil, nil
		}

		opMsg, futureOps, err := simulation.GenAndDeliverTxWithRandFees(
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
				CoinsSpentInMsg: spentCoins,
			},
		)

		return opMsg, futureOps, common.CombineErrors(err, errFundAccount)
	}
}

// Ensure wether the position we open won't trigger the fluctuation limit.
func checkIsOverFluctation(
	ctx sdk.Context, k keeper.Keeper, pool pooltypes.Market, openNotional sdk.Dec, direction pooltypes.Direction) bool {
	quoteDelta := openNotional
	baseDelta, _ := pool.GetBaseAmountByQuoteAmount(quoteDelta.Abs().MulInt64(direction.ToMultiplier()))
	snapshot, _ := k.PerpAmmKeeper.GetLastSnapshot(ctx, pool)
	currentPrice := snapshot.QuoteReserve.Quo(snapshot.BaseReserve)
	newPrice := pool.QuoteReserve.Add(quoteDelta).Quo(pool.BaseReserve.Sub(baseDelta))

	fluctuationLimitRatio := pool.Config.FluctuationLimitRatio
	snapshotUpperLimit := currentPrice.Mul(sdk.OneDec().Add(fluctuationLimitRatio))
	snapshotLowerLimit := currentPrice.Mul(sdk.OneDec().Sub(fluctuationLimitRatio))
	isOverFluctation := newPrice.GT(snapshotUpperLimit) || newPrice.LT(snapshotLowerLimit)
	return isOverFluctation
}

/*
getMaxQuoteForPool computes the maximum quote the user can swap considering the max fluctuation ratio and  trade limit
ratio.

Fluctuation limit ratio:
------------------------

	Considering a xy=k pool, the price evolution for a swap of quote=q can be written as:

		price_evolution = (1 + q/quoteReserve) ** 2

	which means that the trade will be under the fluctuation limit l if:

			abs(price_evolution - 1) <= l
	<=>		sqrt(1-l) * quoteReserve < q < sqrt(l+1) * quoteReserve

	In our case we only care about the right part since q is always positive (short/long would be the sign).

Trade limit ratio:
------------------

	The maximum quote amount considering the trade limit ratio is set at:

	 	q <= QuoteReserve * tl

		with tl the trade limit ratio.
*/
func getMaxQuoteForPool(pool pooltypes.Market) sdk.Dec {
	ratioFloat := math.Sqrt(pool.Config.FluctuationLimitRatio.Add(sdk.OneDec()).MustFloat64())
	maxQuoteFluctationLimit := sdk.MustNewDecFromStr(fmt.Sprintf("%f", ratioFloat)).Mul(pool.QuoteReserve)

	maxQuoteTradeLimit := pool.QuoteReserve.Mul(pool.Config.TradeLimitRatio)

	return sdk.MinDec(maxQuoteTradeLimit, maxQuoteFluctationLimit)
}

// SimulateMsgClosePosition generates a MsgClosePosition with random values.
func SimulateMsgClosePosition(ak types.AccountKeeper, bk types.BankKeeper, k keeper.Keeper) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		trader := simAccount.Address.String()
		pair := asset.Registry.Pair(denoms.BTC, denoms.NUSD)

		msg := &types.MsgClosePosition{
			Sender: trader,
			Pair:   pair,
		}

		_, err := k.Positions.Get(ctx, collections.Join(asset.Registry.Pair(denoms.BTC, denoms.NUSD), simAccount.Address))
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, msg.Type(), "no position opened yet"), nil, nil
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
				CoinsSpentInMsg: sdk.NewCoins(),
			},
		)
	}
}

// SimulateMsgAddMargin generates a MsgAddMargin with random values.
func SimulateMsgAddMargin(ak types.AccountKeeper, bk types.BankKeeper, k keeper.Keeper) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		trader := simAccount.Address.String()
		pair := asset.Registry.Pair(denoms.BTC, denoms.NUSD)

		msg := &types.MsgAddMargin{}
		_, err := k.Positions.Get(ctx, collections.Join(asset.Registry.Pair(denoms.BTC, denoms.NUSD), simAccount.Address))
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, msg.Type(), "no position opened yet"), nil, nil
		}

		spendableCoins := bk.SpendableCoins(ctx, simAccount.Address)

		if spendableCoins.AmountOf(denoms.NUSD).IsZero() {
			return simtypes.NoOpMsg(types.ModuleName, msg.Type(), "no nusd left"), nil, nil
		}
		quoteAmt, _ := simtypes.RandPositiveInt(r, spendableCoins.AmountOf(denoms.NUSD))

		spentCoin := sdk.NewCoin(denoms.NUSD, quoteAmt)

		msg = &types.MsgAddMargin{
			Sender: trader,
			Pair:   pair,
			Margin: spentCoin,
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
				CoinsSpentInMsg: sdk.NewCoins(spentCoin),
			},
		)
	}
}

// SimulateMsgRemoveMargin generates a MsgRemoveMargin with random values.
func SimulateMsgRemoveMargin(ak types.AccountKeeper, bk types.BankKeeper, k keeper.Keeper) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (opMsg simtypes.OperationMsg, futureOps []simtypes.FutureOperation, err error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		trader := simAccount.Address.String()
		pair := asset.Registry.Pair(denoms.BTC, denoms.NUSD)

		msg := &types.MsgRemoveMargin{}

		position, err := k.Positions.Get(ctx, collections.Join(asset.Registry.Pair(denoms.BTC, denoms.NUSD), simAccount.Address))
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, msg.Type(), "no position opened yet"), nil, nil
		}

		//simple calculation, might still fail due to funding rate or unrealizedPnL
		maintenanceMarginRatio, err := k.PerpAmmKeeper.GetMaintenanceMarginRatio(ctx, position.Pair)
		if err != nil {
			return
		}
		maintenanceMarginRequirement := position.OpenNotional.Mul(maintenanceMarginRatio)
		maxMarginToRemove := position.Margin.Sub(maintenanceMarginRequirement).Quo(sdk.NewDec(2))

		if maxMarginToRemove.TruncateInt().LT(sdk.OneInt()) {
			return simtypes.NoOpMsg(types.ModuleName, msg.Type(), "margin too tight"), nil, nil
		}

		marginToRemove, _ := simtypes.RandPositiveInt(r, maxMarginToRemove.TruncateInt())

		expectedCoin := sdk.NewCoin(denoms.NUSD, marginToRemove)

		msg = &types.MsgRemoveMargin{
			Sender: trader,
			Pair:   pair,
			Margin: expectedCoin,
		}

		opMsg, futureOps, err = simulation.GenAndDeliverTxWithRandFees(
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
				CoinsSpentInMsg: sdk.NewCoins(),
			},
		)
		if err != nil {
			errDebugHelper := fmt.Errorf("expectedCoin: %s, maxMarginToRemove: %s", expectedCoin, maxMarginToRemove)
			err = common.CombineErrors(err, errDebugHelper)
		}

		return opMsg, futureOps, err
	}
}

func fundAccountWithTokens(ctx sdk.Context, receiver sdk.AccAddress, bk types.BankKeeper) (err error) {
	newCoins := sdk.NewCoins(
		sdk.NewCoin(denoms.NUSD, sdk.NewInt(1e6)),
	)

	if err := bk.MintCoins(ctx, types.ModuleName, newCoins); err != nil {
		return err
	}

	if err := bk.SendCoinsFromModuleToAccount(
		ctx,
		types.ModuleName,
		receiver,
		newCoins,
	); err != nil {
		return err
	}

	return nil
}
