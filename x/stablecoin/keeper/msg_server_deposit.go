package keeper

import (
	"context"

	tokens "github.com/MatrixDao/matrix/x/pricefeed/tokens"
	"github.com/MatrixDao/matrix/x/stablecoin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) Deposit(goCtx context.Context, msg *types.MsgDeposit) (*types.MsgDepositResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	var fromAddr sdk.AccAddress
	fromAddr, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}

	hasEnoughBalance, err := k.CheckEnoughBalance(ctx, msg.Stable, fromAddr)
	// Is there USDM in the account?
	if err != nil {
		return nil, err
	}
	if !hasEnoughBalance {
		return nil, types.NotEnoughBalance.Wrap(msg.Stable.Amount.String())
	}

	// User deposits USDM into the protocol.
	err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, fromAddr, types.ModuleName, sdk.NewCoins(msg.Stable))
	if err != nil {
		return nil, err
	}

	// User receives some mixture of collateral and GOV tokens based on the collateral ratio..
	// TODO Initialize based on the collateral ratio of the protocol
	collateralRatio, _ := sdk.NewDecFromStr("0.9")
	govRatio := sdk.NewDec(1).Sub(collateralRatio)

	// TODO Read the governance token price (MTRX per USD) from an oracle.
	priceGov, _ := sdk.NewDecFromStr("10")
	// TODO Read the collateral token price (per USD) from an oracle.
	priceCollateral, _ := sdk.NewDecFromStr("1")

	amtGov := sdk.NewDecFromInt(msg.Stable.Amount).Mul(govRatio).Quo(priceGov)
	tenDec := sdk.MustNewDecFromStr("10")
	govExponent := uint64(tokens.NATIVE_MAP["mtrx"].BaseExponent)
	baseAmtGov := sdk.NewIntFromBigInt(
		amtGov.Mul(tenDec.Power(govExponent)).BigInt())

	amtCollateral := sdk.NewDecFromInt(
		msg.Stable.Amount).Mul(collateralRatio).Quo(priceCollateral)
	collateralExponent := uint64(tokens.IBC_MAP[msg.CollateralDenom].BaseExponent)
	baseAmtCollateral := sdk.NewIntFromBigInt(
		amtCollateral.Mul(tenDec.Power(collateralExponent)).BigInt())

	gov := sdk.NewCoin(tokens.NATIVE_MAP["mtrx"].BaseDisplay, baseAmtGov)
	collateral := sdk.NewCoin(tokens.IBC_MAP["mtrx"].BaseDisplay, baseAmtCollateral)
	msgDepositResponse := new(types.MsgDepositResponse)
	msgDepositResponse.AmtGOV = gov
	msgDepositResponse.AmtCollateral = collateral

	// msg.CollateralDenom
	// coinWithdraw := sdk.NewCoin(msg.Collateral.Denom, msg.Collateral.Amount)
	// err = k.bankKeeper.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(coinDeposit))

	_ = ctx

	return msgDepositResponse, nil
}

// Redeem logic
// TODO collateralRatio:
// TODO collateralRedeemedY: Y, the collateral going to the user.
// TODO priceY: USDM price of Y
// TODO mintedGOV
// TODO priceGOV
