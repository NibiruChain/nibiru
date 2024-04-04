package types

import (
	"context"
	"cosmossdk.io/core/store"
	"cosmossdk.io/math"
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// StakingKeeper is expected keeper for staking module
type StakingKeeper interface {
	Validator(ctx context.Context, address sdk.ValAddress) (stakingtypes.ValidatorI, error)    // get validator by operator address; nil when validator not found
	TotalBondedTokens(context.Context) (sdkmath.Int, error)                                    // total bonded tokens within the validator set
	Slash(context.Context, sdk.ConsAddress, int64, int64, math.LegacyDec) (sdkmath.Int, error) // slash the validator and delegators of the validator, specifying offense height, offense power, and slash fraction
	Jail(context.Context, sdk.ConsAddress) error                                               // jail a validator
	ValidatorsPowerStoreIterator(ctx context.Context) (store.Iterator, error)                  // an iterator for the current validator power store
	MaxValidators(context.Context) (uint32, error)                                             // MaxValidators returns the maximum amount of bonded validators
	PowerReduction(ctx context.Context) (res sdkmath.Int)
}

// DistributionKeeper is expected keeper for distribution module
type DistributionKeeper interface {
	AllocateTokensToValidator(ctx context.Context, val stakingtypes.ValidatorI, tokens sdk.DecCoins) error

	// only used for simulation
	GetValidatorOutstandingRewardsCoins(ctx context.Context, val sdk.ValAddress) (sdk.DecCoins, error)
}

// AccountKeeper is expected keeper for auth module
type AccountKeeper interface {
	GetModuleAddress(name string) sdk.AccAddress
	GetModuleAccount(ctx context.Context, moduleName string) sdk.ModuleAccountI
	GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI // only used for simulation
}

// BankKeeper defines the expected interface needed to retrieve account balances.
type BankKeeper interface {
	GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin
	GetAllBalances(ctx context.Context, addr sdk.AccAddress) sdk.Coins
	SendCoinsFromModuleToModule(ctx context.Context, senderModule string, recipientModule string, amt sdk.Coins) error
	// only used for simulation
	SpendableCoins(ctx context.Context, addr sdk.AccAddress) sdk.Coins
}

type SudoKeeper interface {
	// CheckPermissions Checks if a contract is contained within the set of sudo
	// contracts defined in the x/sudo module. These smart contracts are able to
	// execute certain permissioned functions.
	CheckPermissions(contract sdk.AccAddress, ctx sdk.Context) error
}
