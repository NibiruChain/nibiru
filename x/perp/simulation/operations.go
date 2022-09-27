package simulation

import (
	"fmt"
	"math/rand"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/keeper"
	"github.com/NibiruChain/nibiru/x/perp/types"
)

const defaultWeight = 100

// WeightedOperations returns all the operations from the module with their respective weights
func WeightedOperations(
	appParams simtypes.AppParams,
	cdc codec.JSONCodec,
	ak types.AccountKeeper,
	bk types.BankKeeper,
	k keeper.Keeper) simulation.WeightedOperations {
	return simulation.WeightedOperations{
		simulation.NewWeightedOperation(
			defaultWeight,
			SimulateMsgOpenPosition(ak, bk, k),
		),
	}
}

// SimulateMsgCreateBalancerPool generates a MsgCreatePool with random values.
func SimulateMsgOpenPosition(ak types.AccountKeeper, bk types.BankKeeper, k keeper.Keeper) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		fundAccountWithTokens(ctx, simAccount.Address, bk)
		spendableCoins := bk.SpendableCoins(ctx, simAccount.Address)

		quoteAmt, _ := simtypes.RandPositiveInt(r, spendableCoins.AmountOf(common.DenomNUSD))
		leverage := simtypes.RandomDecAmount(r, sdk.NewDec(9)).Add(sdk.OneDec()) // between [1, 10]
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
