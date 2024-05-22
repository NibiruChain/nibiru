package types

import (
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// StakingKeeper is expected keeper for staking module
type StakingKeeper interface {
	Validator(ctx sdk.Context, address sdk.ValAddress) stakingtypes.ValidatorI    // get validator by operator address; nil when validator not found
	TotalBondedTokens(sdk.Context) sdkmath.Int                                    // total bonded tokens within the validator set
	Slash(sdk.Context, sdk.ConsAddress, int64, int64, math.LegacyDec) sdkmath.Int // slash the validator and delegators of the validator, specifying offense height, offense power, and slash fraction
	Jail(sdk.Context, sdk.ConsAddress)                                            // jail a validator
	ValidatorsPowerStoreIterator(ctx sdk.Context) sdk.Iterator                    // an iterator for the current validator power store
	MaxValidators(sdk.Context) uint32                                             // MaxValidators returns the maximum amount of bonded validators
	PowerReduction(ctx sdk.Context) (res sdkmath.Int)
}

type SlashingKeeper interface {
	Slash(ctx sdk.Context, consAddr sdk.ConsAddress, fraction math.LegacyDec, power int64, height int64)
	Jail(sdk.Context, sdk.ConsAddress)
}

// DistributionKeeper is expected keeper for distribution module
type DistributionKeeper interface {
	AllocateTokensToValidator(ctx sdk.Context, val stakingtypes.ValidatorI, tokens math.LegacyDecCoins)

	// only used for simulation
	GetValidatorOutstandingRewardsCoins(ctx sdk.Context, val sdk.ValAddress) math.LegacyDecCoins
}

// AccountKeeper is expected keeper for auth module
type AccountKeeper interface {
	GetModuleAddress(name string) sdk.AccAddress
	GetModuleAccount(ctx sdk.Context, moduleName string) authtypes.ModuleAccountI
	GetAccount(ctx sdk.Context, addr sdk.AccAddress) authtypes.AccountI // only used for simulation
}

// BankKeeper defines the expected interface needed to retrieve account balances.
type BankKeeper interface {
	GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin
	GetAllBalances(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins
	SendCoinsFromModuleToModule(ctx sdk.Context, senderModule string, recipientModule string, amt sdk.Coins) error
	// only used for simulation
	SpendableCoins(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins
}

type SudoKeeper interface {
	// CheckPermissions Checks if a contract is contained within the set of sudo
	// contracts defined in the x/sudo module. These smart contracts are able to
	// execute certain permissioned functions.
	CheckPermissions(contract sdk.AccAddress, ctx sdk.Context) error
}
