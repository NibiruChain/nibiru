package lockup

import (
	"github.com/MatrixDao/matrix/x/lockup/keeper"
	abci "github.com/tendermint/tendermint/abci/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// BeginBlocker is called on every block.
func BeginBlocker(ctx sdk.Context, req abci.RequestBeginBlock, k keeper.LockupKeeper) {
}

// Called every block to automatically unlock matured locks.
func EndBlocker(ctx sdk.Context, k keeper.LockupKeeper) []abci.ValidatorUpdate {
	return []abci.ValidatorUpdate{}
}
