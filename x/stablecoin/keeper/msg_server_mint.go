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

	for _, coin := range []sdk.Coin{msg.Collateral, msg.Gov} {
		hasEnoughBalance, err := k.CheckEnoughBalance(ctx, coin, fromAddr)
		if err != nil {
			return nil, err
		}

		if !hasEnoughBalance {
			return nil, types.NotEnoughBalance.Wrap(coin.Amount.String())
		}
	}

	// TODO: This should be called after we compute 'maxMintableStable'
	err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, fromAddr, types.ModuleName, sdk.NewCoins(msg.Collateral))
	if err != nil {
		return nil, err
	}

	/*  Minting USDM
	TODO(heisenberg): Get the actual price to multiply by
	See Example B of https://docs.frax.finance/minting-and-redeeming

	collateralDeposited: (sdk.Coin)
	collateralRatio:
	priceGOV: Price of the governance token in USD.

	govDeposited: Units of GOV burned
	govDeposited = (1 - collateralRatio) * (collateralDeposited * 1) / (collateralRatio * priceGOV)

	*/
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

	return &types.MsgMintResponse{Stable: newCoin}, nil
}
