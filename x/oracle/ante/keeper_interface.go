package ante

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// We only need these two methods from StakingKeeper in the AnteDecorator
type StakingKeeperI interface {
	GetValidator(ctx sdk.Context, addr sdk.ValAddress) (stakingtypes.Validator, bool)
	TotalBondedTokens(ctx sdk.Context) sdk.Int
}

// We only need this method from the oracle keeper in the AnteDecorator.
type OracleKeeperI interface {
	HasVotedInCurrentPeriod(ctx sdk.Context, valAddr sdk.ValAddress) bool
}
