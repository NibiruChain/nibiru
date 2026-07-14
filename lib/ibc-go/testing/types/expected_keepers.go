package types

import (
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	stakingtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/staking/types"
)

// StakingKeeper defines the expected staking keeper interface used in the
// IBC testing package
type StakingKeeper interface {
	GetHistoricalInfo(ctx sdk.Context, height int64) (stakingtypes.HistoricalInfo, bool)
}
