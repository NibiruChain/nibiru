package types

import (
	"fmt"

	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	sdkerrors "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/errors"
)

// UpgradeInfoFileName file to store upgrade information
const UpgradeInfoFilename = "upgrade-info.json"

func (p Plan) String() string {
	due := p.DueAt()
	return fmt.Sprintf(`Upgrade Plan
  Name: %s
  %s
  Info: %s.`, p.Name, due, p.Info)
}

// ValidateBasic validates a concrete, executable upgrade plan. Such a plan has
// a positive block height and no time target, as required by IBC proof handling
// and existing node tooling. [Plan.ValidateScheduleRequest] also accepts a
// time-based request; the contract later resolves that request into this
// height-only form. Keeping these validators separate prevents an unresolved
// time request from being treated as an executable or IBC upgrade plan.
func (p Plan) ValidateBasic() error {
	if !p.Time.IsZero() {
		return sdkerrors.ErrInvalidRequest.Wrap("time-based upgrades have been deprecated in the SDK")
	}
	if p.UpgradedClientState != nil {
		return sdkerrors.ErrInvalidRequest.Wrap("upgrade logic for IBC has been moved to the IBC module")
	}
	if len(p.Name) == 0 {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "name cannot be empty")
	}
	if p.Height <= 0 {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "height must be greater than 0")
	}

	return nil
}

// ShouldExecute returns true if the Plan is ready to execute given the current context
func (p Plan) ShouldExecute(ctx sdk.Context) bool {
	if p.Height > 0 {
		return p.Height <= ctx.BlockHeight()
	}
	return false
}

// DueAt is a string representation of when this plan is due to be executed
func (p Plan) DueAt() string {
	return fmt.Sprintf("height: %d", p.Height)
}

// ValidateScheduleRequest validates the native request shape. Plan.ValidateBasic
// remains height-only for finalized plans, IBC proof scheduling and node tools.
// Stateful scheduling rules are enforced by the x-upgrade Rust contract.
func (p Plan) ValidateScheduleRequest() error {
	if p.Time.IsZero() {
		return p.ValidateBasic()
	}
	if p.Height != 0 {
		return sdkerrors.ErrInvalidRequest.Wrap("specify exactly one of upgrade height or time")
	}
	if p.Name == "" {
		return sdkerrors.ErrInvalidRequest.Wrap("name cannot be empty")
	}
	if p.UpgradedClientState != nil {
		return sdkerrors.ErrInvalidRequest.Wrap("upgrade logic for IBC has been moved to the IBC module")
	}
	return nil
}
