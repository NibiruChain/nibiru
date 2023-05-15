package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common/denoms"
	v2types "github.com/NibiruChain/nibiru/x/perp/types/v2"
)

func (k Keeper) Admin() admin {
	return admin{&k}
}

type admin struct{ *Keeper }

/*
AdminWithdrawFromInsuranceFund // TODO docs
*/
func (k admin) WithdrawFromInsuranceFund(
	ctx sdk.Context, amount sdk.Int, to sdk.AccAddress,
) (err error) {
	coinToSend := sdk.NewCoin(denoms.NUSD, amount)
	if err = k.BankKeeper.SendCoinsFromModuleToAccount(
		ctx,
		/* from */ v2types.PerpEFModuleAccount,
		/* to */ to,
		/* amount */ sdk.NewCoins(coinToSend),
	); err != nil {
		return err
	}
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"withdraw_from_if",
		sdk.NewAttribute("to", to.String()),
		sdk.NewAttribute("funds", coinToSend.String()),
	))
	return nil
}
