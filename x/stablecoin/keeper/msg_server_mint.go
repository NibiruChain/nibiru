package keeper

import (
	"context"

	"github.com/MatrixDao/matrix/x/stablecoin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (k msgServer) Mint(goCtx context.Context, msg *types.MsgMint) (*types.MsgMintResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	fromAddr, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}

	hasEnoughBalance, err := k.checkEnoughBalance(ctx, msg.Collateral, fromAddr)
	if err != nil {
		return nil, err
	}

	if !hasEnoughBalance {
		return nil, types.NotEnoughBalance.Wrap(msg.Collateral.Amount.String())
	}

	err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, fromAddr, types.ModuleName, sdk.NewCoins(msg.Collateral))
	if err != nil {
		return nil, err
	}

	/*  Minting USDM
	TODO: Get the actual price to multiply by
	See Example B of https://docs.frax.finance/minting-and-redeeming
	*/

	assetPriceInfo, err := k.priceKeeper.GetCurrentPrice(ctx, msg.Collateral.Denom)
	if err != nil {
		return nil, sdkerrors.Wrapf(types.ErrPriceNotFound, "no price found for market %s", msg.Collateral.Denom)
	}
	newCoin := sdk.NewCoin("usdm", sdk.NewDecFromInt(msg.Collateral.Amount).Mul(assetPriceInfo.Price).TruncateInt())

	newCoins := sdk.NewCoins(newCoin)
	err = k.bankKeeper.MintCoins(ctx, types.ModuleName, newCoins)
	if err != nil {
		panic(err)
	}

	err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, fromAddr, newCoins)
	if err != nil {
		panic(err)
	}

	return &types.MsgMintResponse{Amount: newCoin}, nil
}
