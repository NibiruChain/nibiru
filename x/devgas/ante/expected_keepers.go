package ante

// Interfaces needed for the Nibiru Chain ante handler

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/x/devgas"
)

type BankKeeper interface {
	SendCoinsFromAccountToModule(
		ctx sdk.Context, senderAddr sdk.AccAddress,
		recipientModule string, amt sdk.Coins,
	) error
	SendCoinsFromModuleToAccount(
		ctx sdk.Context, senderModule string,
		recipientAddr sdk.AccAddress, amt sdk.Coins,
	) error
}

type IDevGasKeeper interface {
	GetParams(ctx sdk.Context) devgas.ModuleParams
	GetFeeShare(ctx sdk.Context, contract sdk.Address) (devgas.FeeShare, bool)
}
