package simulation

import (
	"math/rand"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"

	"github.com/NibiruChain/nibiru/x/common/denoms"
	"github.com/NibiruChain/nibiru/x/stablecoin/keeper"
	"github.com/NibiruChain/nibiru/x/stablecoin/types"
)

func SimulateMsgMintStable(
	k keeper.Keeper,
	ak types.AccountKeeper,
	bk types.BankKeeper,
) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context,
		accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAcc, _ := simtypes.RandomAcc(r, accs)
		// How much stable should get minted?
		simStable := sdk.NewCoin(denoms.DenomNUSD, sdk.NewInt(100))
		msg := &types.MsgMintStable{
			Creator: simAcc.Address.String(),
			Stable:  simStable,
		}

		// TODO: Implement the actual MintStable simulation.
		var simOpMsg = simtypes.NoOpMsg(
			types.ModuleName, msg.Type(), "SimulateMintStable not implemented",
		)
		var futureOp []simtypes.FutureOperation = nil
		return simOpMsg, futureOp, nil
	}
}
