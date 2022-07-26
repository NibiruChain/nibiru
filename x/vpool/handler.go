package vpool

import (
	"fmt"

	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/NibiruChain/nibiru/x/common"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/NibiruChain/nibiru/x/vpool/keeper"
	"github.com/NibiruChain/nibiru/x/vpool/types"
)

// NewHandler ...
func NewHandler(k keeper.Keeper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) (*sdk.Result, error) {
		// ctx = ctx.WithEventManager(sdk.NewEventManager())

		errMsg := fmt.Sprintf("unrecognized %s message type: %T", types.ModuleName, msg)
		return nil, sdkerrors.Wrap(sdkerrors.ErrUnknownRequest, errMsg)
	}
}

func NewCreatePoolProposalHandler(k keeper.Keeper) govtypes.Handler {
	return func(ctx sdk.Context, content govtypes.Content) error {
		switch m := content.(type) {
		case *types.CreatePoolProposal:
			if err := m.ValidateBasic(); err != nil {
				return err
			}
			pair := common.MustNewAssetPair(m.Pair)
			k.CreatePool(
				ctx,
				pair,
				m.TradeLimitRatio,
				m.QuoteAssetReserve,
				m.BaseAssetReserve,
				m.FluctuationLimitRatio,
				m.MaxOracleSpreadRatio,
				m.MaintenanceMarginRatio,
			)
			return nil
		default:
			return sdkerrors.Wrapf(
				sdkerrors.ErrUnknownRequest,
				"unrecognized %s proposal content type: %T", types.ModuleName, m)
		}
	}
}
