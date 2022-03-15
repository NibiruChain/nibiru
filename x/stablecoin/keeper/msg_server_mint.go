package keeper

import (
	"context"

	"github.com/MatrixDao/matrix/x/stablecoin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) Mint(goCtx context.Context, msg *types.MsgMint) (*types.MsgMintResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	fromAddr, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}

	hasEnoughBalance, err := k.GetMinterBalance(ctx, msg.Collateral, fromAddr)
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

	// TODO(heisenberg): Get the actual price to multiply by
	newCoin := sdk.NewCoin("usdm", msg.Collateral.Amount.Mul(sdk.NewInt(10)))
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
