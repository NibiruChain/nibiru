package upgrade

import (
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	sdkerrors "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/errors"
	govtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/upgrade/keeper"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/upgrade/types"
)

// NewSoftwareUpgradeProposalHandler creates a governance handler to manage new proposal types.
// It enables SoftwareUpgradeProposal to propose an Upgrade, and CancelSoftwareUpgradeProposal
// to abort a previously voted upgrade.
//
//nolint:staticcheck // we are intentionally using a deprecated proposal here.
func NewSoftwareUpgradeProposalHandler(k *keeper.Keeper) govtypes.Handler {
	return func(ctx sdk.Context, content govtypes.Content) error {
		switch c := content.(type) {
		case *types.SoftwareUpgradeProposal:
			return handleSoftwareUpgradeProposal(ctx, k, c)

		case *types.CancelSoftwareUpgradeProposal:
			return handleCancelSoftwareUpgradeProposal(ctx, k, c)

		default:
			return sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "unrecognized software upgrade proposal content type: %T", c)
		}
	}
}

//nolint:staticcheck // we are intentionally using a deprecated proposal here.
func handleSoftwareUpgradeProposal(ctx sdk.Context, k *keeper.Keeper, p *types.SoftwareUpgradeProposal) error {
	return k.ScheduleUpgrade(ctx, p.Plan)
}

//nolint:staticcheck // we are intentionally using a deprecated proposal here.
func handleCancelSoftwareUpgradeProposal(ctx sdk.Context, k *keeper.Keeper, _ *types.CancelSoftwareUpgradeProposal) error {
	k.ClearUpgradePlan(ctx)
	return nil
}
