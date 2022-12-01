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
	"github.com/NibiruChain/nibiru/x/perp/keeper"
	"github.com/NibiruChain/nibiru/x/perp/types"
	pooltypes "github.com/NibiruChain/nibiru/x/vpool/types"
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
		fundAccountWithTokens(ctx, simAccount.Address, bk)
		spendableCoins := bk.SpendableCoins(ctx, simAccount.Address)

		pools := k.VpoolKeeper.GetAllPools(ctx)
		pool := pools[rand.Intn(len(pools))]

		maxQuote := getMaxQuoteForPool(pool)
		quoteAmt, _ := simtypes.RandPositiveInt(r, sdk.MinInt(sdk.Int(maxQuote), spendableCoins.AmountOf(common.DenomNUSD)))

		leverage := simtypes.RandomDecAmount(r, pool.Config.MaxLeverage.Sub(sdk.OneDec())).Add(sdk.OneDec()) // between [1, MaxLeverage]
		openNotional := leverage.MulInt(quoteAmt)
		feesAmt := openNotional.Mul(sdk.MustNewDecFromStr("0.002")).Ceil().TruncateInt()
		spentCoins := sdk.NewCoins(sdk.NewCoin(common.DenomNUSD, quoteAmt.Add(feesAmt)))

		msg := &types.MsgOpenPosition{
			Sender:               simAccount.Address.String(),
			TokenPair:            common.Pair_BTC_NUSD.String(),
			Side:                 types.Side_BUY,
			QuoteAssetAmount:     quoteAmt,
			Leverage:             leverage,
			BaseAssetAmountLimit: sdk.ZeroInt(),
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
		if err != nil {
			fmt.Println(spendableCoins)
			fmt.Println(quoteAmt)
		}

		return opMsg, futureOps, err
	}
}

/*
getMaxQuoteForPool computes the maximum quote the user can swap considering the max fluctuation ratio and  trade limit
ratio.

Fluctuation limit ratio:
------------------------

	Considering a xy=k pool, the price evolution for a swap of quote=q can be written as:

		price_evolution = (1 + q/quoteAssetReserve) ** 2

	which means that the trade will be under the fluctuation limit l if:

			abs(price_evolution - 1) <= l
	<=>		sqrt(1-l) * quoteAssetReserve < q < sqrt(l+1) * quoteAssetReserve

	In our case we only care about the right part since q is always positive (short/long would be the sign).

Trade limit ratio:
------------------

	The maximum quote amount considering the trade limit ratio is set at:

	 	q <= QuoteAssetReserve * tl

		with tl the trade limit ratio.
*/
func getMaxQuoteForPool(pool pooltypes.Vpool) sdk.Dec {
	ratioFloat := math.Sqrt(pool.Config.FluctuationLimitRatio.Add(sdk.OneDec()).MustFloat64())
	maxQuoteFluctationLimit := sdk.MustNewDecFromStr(fmt.Sprintf("%f", ratioFloat)).Mul(pool.QuoteAssetReserve)

	maxQuoteTradeLimit := pool.QuoteAssetReserve.Mul(pool.Config.TradeLimitRatio)

	return sdk.MinDec(maxQuoteTradeLimit, maxQuoteFluctationLimit)
}

// SimulateMsgClosePosition generates a MsgClosePosition with random values.
func SimulateMsgClosePosition(ak types.AccountKeeper, bk types.BankKeeper, k keeper.Keeper) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		trader := simAccount.Address.String()
		pair := common.Pair_BTC_NUSD.String()

		msg := &types.MsgClosePosition{
			Sender:    trader,
			TokenPair: pair,
		}

		_, err := k.Positions.Get(ctx, collections.Join(common.Pair_BTC_NUSD, simAccount.Address))
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
		pair := common.Pair_BTC_NUSD.String()

		msg := &types.MsgAddMargin{}
		_, err := k.Positions.Get(ctx, collections.Join(common.Pair_BTC_NUSD, simAccount.Address))
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, msg.Type(), "no position opened yet"), nil, nil
		}

		spendableCoins := bk.SpendableCoins(ctx, simAccount.Address)

		if spendableCoins.AmountOf(common.DenomNUSD).IsZero() {
			return simtypes.NoOpMsg(types.ModuleName, msg.Type(), "no nusd left"), nil, nil
		}
		quoteAmt, _ := simtypes.RandPositiveInt(r, spendableCoins.AmountOf(common.DenomNUSD))

		spentCoin := sdk.NewCoin(common.DenomNUSD, quoteAmt)

		msg = &types.MsgAddMargin{
			Sender:    trader,
			TokenPair: pair,
			Margin:    spentCoin,
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
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		trader := simAccount.Address.String()
		pair := common.Pair_BTC_NUSD.String()

		msg := &types.MsgRemoveMargin{}

		position, err := k.Positions.Get(ctx, collections.Join(common.Pair_BTC_NUSD, simAccount.Address))
		if err != nil {
			return simtypes.NoOpMsg(types.ModuleName, msg.Type(), "no position opened yet"), nil, nil
		}

		//simple calculation, might still fail due to funding rate or unrealizedPnL
		maintenanceMarginRatio := k.VpoolKeeper.GetMaintenanceMarginRatio(ctx, position.GetPair())
		maintenanceMarginRequirement := position.OpenNotional.Mul(maintenanceMarginRatio)
		maxMarginToRemove := position.Margin.Sub(maintenanceMarginRequirement).Quo(sdk.NewDec(2))

		if maxMarginToRemove.TruncateInt().LT(sdk.OneInt()) {
			return simtypes.NoOpMsg(types.ModuleName, msg.Type(), "margin too tight"), nil, nil
		}

		marginToRemove, _ := simtypes.RandPositiveInt(r, maxMarginToRemove.TruncateInt())

		expectedCoin := sdk.NewCoin(common.DenomNUSD, marginToRemove)

		msg = &types.MsgRemoveMargin{
			Sender:    trader,
			TokenPair: pair,
			Margin:    expectedCoin,
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
				CoinsSpentInMsg: sdk.NewCoins(),
			},
		)
		if err != nil {
			fmt.Println(expectedCoin)
			fmt.Println(maxMarginToRemove)
		}

		return opMsg, futureOps, err
	}
}

func fundAccountWithTokens(ctx sdk.Context, receiver sdk.AccAddress, bk types.BankKeeper) {
	newCoins := sdk.NewCoins(
		sdk.NewCoin(common.DenomNUSD, sdk.NewInt(1e6)),
	)

	if err := bk.MintCoins(ctx, types.ModuleName, newCoins); err != nil {
		panic(err)
	}

	if err := bk.SendCoinsFromModuleToAccount(
		ctx,
		types.ModuleName,
		receiver,
		newCoins,
	); err != nil {
		panic(err)
	}
}
