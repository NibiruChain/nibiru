package simulation

import (
	"math/rand"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/simapp/helpers"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/auth/legacy/legacytx"
	"github.com/cosmos/cosmos-sdk/x/simulation"
	stakingTypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

type SimulationContext struct {
	R          *rand.Rand
	SdkCtx     sdk.Context
	App        *baseapp.BaseApp
	Accs       []simtypes.Account
	simAccount *simtypes.Account
}

func NewSimulationContext(r *rand.Rand, ctx sdk.Context, app *baseapp.BaseApp, accs []simtypes.Account) SimulationContext {
	return SimulationContext{r, ctx, app, accs, nil}
}

func (ctx *SimulationContext) GetMsgSender() simtypes.Account {
	if ctx.simAccount == nil {
		sel := ctx.R.Intn(len(ctx.Accs))
		ctx.simAccount = &ctx.Accs[sel]
	}
	return *ctx.simAccount
}

func GenAndDeliverTxWithRandFees(
	r *rand.Rand,
	app *baseapp.BaseApp,
	txGen client.TxConfig,
	msg legacytx.LegacyMsg,
	coinsSpentInMsg sdk.Coins,
	ctx sdk.Context,
	simAccount simtypes.Account,
	ak stakingTypes.AccountKeeper,
	bk stakingTypes.BankKeeper,
	moduleName string,
) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
	account := ak.GetAccount(ctx, simAccount.Address)
	spendable := bk.SpendableCoins(ctx, account.GetAddress())

	var fees sdk.Coins
	var err error

	coins, hasNeg := spendable.SafeSub(coinsSpentInMsg)
	if hasNeg {
		return simtypes.NoOpMsg(moduleName, msg.Type(), "message doesn't leave room for fees"), nil, err
	}

	// Only allow fees in common.DenomGov
	// coins = sdk.NewCoins(sdk.NewCoin(common.DenomGov, coins.AmountOf(common.DenomGov)))

	fees, err = Fees(r, coins)
	if err != nil {
		return simtypes.NoOpMsg(moduleName, msg.Type(), "unable to generate fees"), nil, err
	}

	txCtx := simulation.OperationInput{
		R:               r,
		App:             app,
		TxGen:           txGen,
		Cdc:             nil,
		Msg:             msg,
		MsgType:         msg.Type(),
		CoinsSpentInMsg: coinsSpentInMsg,
		Context:         ctx,
		SimAccount:      simAccount,
		AccountKeeper:   ak,
		Bankkeeper:      bk,
		ModuleName:      moduleName,
	}
	return GenAndDeliverTx(txCtx, fees)
}

// GenAndDeliverTx generates a transactions and delivers it.
func GenAndDeliverTx(txCtx simulation.OperationInput, fees sdk.Coins) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
	account := txCtx.AccountKeeper.GetAccount(txCtx.Context, txCtx.SimAccount.Address)
	tx, err := helpers.GenTx(
		txCtx.TxGen,
		[]sdk.Msg{txCtx.Msg},
		fees,
		Gas,
		txCtx.Context.ChainID(),
		[]uint64{account.GetAccountNumber()},
		[]uint64{account.GetSequence()},
		txCtx.SimAccount.PrivKey,
	)

	if err != nil {
		return simtypes.NoOpMsg(txCtx.ModuleName, txCtx.MsgType, "unable to generate mock tx"), nil, err
	}

	_, _, err = txCtx.App.Deliver(txCtx.TxGen.TxEncoder(), tx)
	if err != nil {
		return simtypes.NoOpMsg(txCtx.ModuleName, txCtx.MsgType, "unable to deliver tx"), nil, err
	}

	return simtypes.NewOperationMsg(txCtx.Msg, true, "", txCtx.Cdc), nil, nil
}
