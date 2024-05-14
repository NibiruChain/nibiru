package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/NibiruChain/nibiru/x/common"

	"github.com/NibiruChain/nibiru/x/tokenfactory/types"
)

var _ types.MsgServer = (*Keeper)(nil)

var errNilMsg error = common.ErrNilGrpcMsg()

func (k Keeper) CreateDenom(
	goCtx context.Context, txMsg *types.MsgCreateDenom,
) (resp *types.MsgCreateDenomResponse, err error) {
	if txMsg == nil {
		return resp, errNilMsg
	}
	if err := txMsg.ValidateBasic(); err != nil {
		return resp, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	denom := types.TFDenom{
		Creator:  txMsg.Sender,
		Subdenom: txMsg.Subdenom,
	}
	err = k.Store.InsertDenom(ctx, denom)
	if err != nil {
		return resp, err
	}

	return &types.MsgCreateDenomResponse{
		NewTokenDenom: denom.String(),
	}, err
}

func (k Keeper) ChangeAdmin(
	goCtx context.Context, txMsg *types.MsgChangeAdmin,
) (resp *types.MsgChangeAdminResponse, err error) {
	if txMsg == nil {
		return resp, errNilMsg
	}
	if err := txMsg.ValidateBasic(); err != nil {
		return resp, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	authData, err := k.Store.GetDenomAuthorityMetadata(ctx, txMsg.Denom)
	if err != nil {
		return resp, err
	}
	if txMsg.Sender != authData.Admin {
		return resp, types.ErrInvalidSender.Wrapf(
			"only the current admin can set a new admin: current admin (%s), sender (%s)",
			authData.Admin, txMsg.Sender,
		)
	}

	authData.Admin = txMsg.NewAdmin
	k.Store.denomAdmins.Insert(ctx, txMsg.Denom, authData)

	return &types.MsgChangeAdminResponse{}, ctx.EventManager().EmitTypedEvent(
		&types.EventChangeAdmin{
			Denom:    txMsg.Denom,
			OldAdmin: txMsg.Sender,
			NewAdmin: txMsg.NewAdmin,
		})
}

// UpdateModuleParams: Message handler for the abci.Msg: MsgUpdateModuleParams
func (k Keeper) UpdateModuleParams(
	goCtx context.Context, txMsg *types.MsgUpdateModuleParams,
) (resp *types.MsgUpdateModuleParamsResponse, err error) {
	if txMsg == nil {
		return resp, errNilMsg
	}
	if err := txMsg.ValidateBasic(); err != nil {
		return resp, err
	}

	if k.authority != txMsg.Authority {
		return nil, govtypes.ErrInvalidSigner.Wrapf("invalid authority; expected %s, got %s", k.authority, txMsg.Authority)
	}

	if err := txMsg.Params.Validate(); err != nil {
		return resp, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	k.Store.ModuleParams.Set(ctx, txMsg.Params)
	return &types.MsgUpdateModuleParamsResponse{}, err
}

// Mint: Message handler for the abci.Msg: MsgMint
func (k Keeper) Mint(
	goCtx context.Context, txMsg *types.MsgMint,
) (resp *types.MsgMintResponse, err error) {
	if txMsg == nil {
		return resp, errNilMsg
	}
	if err := txMsg.ValidateBasic(); err != nil {
		return resp, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	admin, err := k.Store.GetAdmin(ctx, txMsg.Coin.Denom)
	if err != nil {
		return nil, err
	}

	if txMsg.Sender != admin {
		return resp, types.ErrUnauthorized.Wrapf(
			"sender (%s), admin (%s)", txMsg.Sender, admin,
		)
	}

	if txMsg.MintTo == "" {
		txMsg.MintTo = txMsg.Sender
	}

	if err := k.mint(
		ctx, txMsg.Coin, txMsg.MintTo, txMsg.Sender,
	); err != nil {
		return resp, err
	}

	return &types.MsgMintResponse{
			MintTo: txMsg.MintTo,
		}, ctx.EventManager().EmitTypedEvent(
			&types.EventMint{
				Coin:   txMsg.Coin,
				ToAddr: txMsg.MintTo,
				Caller: txMsg.Sender,
			},
		)
}

func (k Keeper) mint(
	ctx sdk.Context, coin sdk.Coin, mintTo string, caller string,
) error {
	if err := types.DenomStr(coin.Denom).Validate(); err != nil {
		return err
	}

	coins := sdk.NewCoins(coin)
	err := k.bankKeeper.MintCoins(ctx, types.ModuleName, coins)
	if err != nil {
		return err
	}

	mintToAddr, err := sdk.AccAddressFromBech32(mintTo)
	if err != nil {
		return err
	}

	if k.bankKeeper.BlockedAddr(mintToAddr) {
		return types.ErrBlockedAddress.Wrapf(
			"failed to mint to %s", mintToAddr)
	}

	return k.bankKeeper.SendCoinsFromModuleToAccount(
		ctx, types.ModuleName, mintToAddr, coins,
	)
}

// Burn: Message handler for the abci.Msg: MsgBurn
func (k Keeper) Burn(
	goCtx context.Context, txMsg *types.MsgBurn,
) (resp *types.MsgBurnResponse, err error) {
	if txMsg == nil {
		return resp, errNilMsg
	}
	if err := txMsg.ValidateBasic(); err != nil {
		return resp, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	admin, err := k.Store.GetAdmin(ctx, txMsg.Coin.Denom)
	if err != nil {
		return nil, err
	}

	if txMsg.Sender != admin {
		return resp, types.ErrUnauthorized.Wrapf(
			"sender (%s), admin (%s)", txMsg.Sender, admin,
		)
	}

	if txMsg.BurnFrom == "" {
		txMsg.BurnFrom = txMsg.Sender
	}

	if err := k.burn(
		ctx, txMsg.Coin, txMsg.BurnFrom, txMsg.Sender,
	); err != nil {
		return resp, err
	}

	return &types.MsgBurnResponse{}, ctx.EventManager().EmitTypedEvent(
		&types.EventBurn{
			Coin:     txMsg.Coin,
			FromAddr: txMsg.BurnFrom,
			Caller:   txMsg.Sender,
		},
	)
}

func (k Keeper) burn(
	ctx sdk.Context, coin sdk.Coin, burnFrom string, caller string,
) error {
	if err := types.DenomStr(coin.Denom).Validate(); err != nil {
		return err
	}

	burnFromAddr, err := sdk.AccAddressFromBech32(burnFrom)
	if err != nil {
		return err
	}

	if k.bankKeeper.BlockedAddr(burnFromAddr) {
		return types.ErrBlockedAddress.Wrapf(
			"failed to burn from %s", burnFromAddr)
	}

	coins := sdk.NewCoins(coin)
	if err = k.bankKeeper.SendCoinsFromAccountToModule(
		ctx, burnFromAddr, types.ModuleName, coins,
	); err != nil {
		return err
	}

	return k.bankKeeper.BurnCoins(ctx, types.ModuleName, coins)
}

// SetDenomMetadata: Message handler for the abci.Msg: MsgSetDenomMetadata
func (k Keeper) SetDenomMetadata(
	goCtx context.Context, txMsg *types.MsgSetDenomMetadata,
) (resp *types.MsgSetDenomMetadataResponse, err error) {
	if txMsg == nil {
		return resp, errNilMsg
	}
	if err := txMsg.ValidateBasic(); err != nil {
		return resp, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	denom := txMsg.Metadata.Base
	admin, err := k.Store.GetAdmin(ctx, denom)
	if err != nil {
		return nil, err
	}

	if txMsg.Sender != admin {
		return resp, types.ErrUnauthorized.Wrapf(
			"sender (%s), admin (%s)", txMsg.Sender, admin,
		)
	}

	k.bankKeeper.SetDenomMetaData(ctx, txMsg.Metadata)

	return &types.MsgSetDenomMetadataResponse{}, ctx.EventManager().
		EmitTypedEvent(&types.EventSetDenomMetadata{
			Denom:    denom,
			Metadata: txMsg.Metadata,
			Caller:   txMsg.Sender,
		})
}

func (k Keeper) BurnNative(
	goCtx context.Context, msg *types.MsgBurnNative,
) (resp *types.MsgBurnNativeResponse, err error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	sender, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	coins := sdk.NewCoins(msg.Coin)

	if err := k.bankKeeper.SendCoinsFromAccountToModule(
		ctx, sender, types.ModuleName, coins,
	); err != nil {
		return nil, err
	}

	err = k.bankKeeper.BurnCoins(ctx, types.ModuleName, coins)
	if err != nil {
		return nil, err
	}

	return &types.MsgBurnNativeResponse{}, err
}
