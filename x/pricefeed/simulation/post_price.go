package simulation

import (
	"math/rand"
	"time"

	"github.com/cosmos/cosmos-sdk/baseapp"
	simappparams "github.com/cosmos/cosmos-sdk/simapp/params"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"

	"github.com/NibiruChain/nibiru/x/pricefeed/keeper"
	"github.com/NibiruChain/nibiru/x/pricefeed/types"
)

var maxPrice = sdk.NewDec(100_000)

const minutesTenYears = 5_256_000

func GenPrice(r *rand.Rand) sdk.Dec {
	return sdk.OneDec().Add(simtypes.RandomDecAmount(r, maxPrice)) // must be > 0
}

func GenFutureDate(r *rand.Rand, now time.Time) time.Time {
	expiryTs := time.Duration(r.Intn(minutesTenYears)) * time.Minute
	return now.Add(1 * time.Minute).Add(expiryTs)
}

func SimulateMsgPostPrice(
	ak types.AccountKeeper,
	bk types.BankKeeper,
	k keeper.Keeper,
) simtypes.Operation {
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		var msg *types.MsgPostPrice
		pairs := k.GetPairs(ctx)
		oraclesByPair := k.GetOraclesForPairs(ctx, pairs)

		if len(pairs) == 0 || len(oraclesByPair) == 0 {
			return simtypes.NoOpMsg(types.ModuleName, msg.Type(), "No pairs or oracles found for post price simulation"), nil, nil
		}

		pairCount := len(pairs)
		pair := pairs[r.Intn(pairCount-1)]
		oracles := oraclesByPair[pair]
		oraclesCount := len(oracles)
		oracle := oracles[r.Intn(oraclesCount-1)]

		simAccount, found := simtypes.FindAccount(accs, oracle)
		if !found {
			return simtypes.NoOpMsg(types.ModuleName, msg.Type(), "oracle account not found"), nil, nil
		}

		msg = &types.MsgPostPrice{
			Token0: pair.Token0,
			Token1: pair.Token1,
			Oracle: oracle.String(),
			Price:  GenPrice(r),
			Expiry: GenFutureDate(r, ctx.BlockTime()),
		}

		txCtx := simulation.OperationInput{
			R:               r,
			App:             app,
			TxGen:           simappparams.MakeTestEncodingConfig().TxConfig,
			Cdc:             nil,
			Msg:             msg,
			MsgType:         msg.Type(),
			Context:         ctx,
			SimAccount:      simAccount,
			ModuleName:      types.ModuleName,
			CoinsSpentInMsg: sdk.NewCoins(),
			AccountKeeper:   ak,
			Bankkeeper:      bk,
		}
		return simulation.GenAndDeliverTxWithRandFees(txCtx)
	}
}
