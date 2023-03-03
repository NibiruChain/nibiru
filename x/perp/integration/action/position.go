package action

import (
	"github.com/NibiruChain/nibiru/app"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type OpenPositionAction struct {
	Account sdk.AccAddress
	Amount  sdk.Coins
}

func (o OpenPositionAction) Do(app *app.NibiruApp, ctx sdk.Context) error {
	return nil
}
