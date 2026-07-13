package types

import (
	sdkmath "cosmossdk.io/math"
	tmprotocrypto "github.com/cometbft/cometbft/proto/tendermint/crypto"

	cryptotypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/crypto/types"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
)

// DelegationI delegation bond for a delegated proof of stake system
type DelegationI interface {
	GetDelegatorAddr() sdk.AccAddress // delegator sdk.AccAddress for the bond
	GetValidatorAddr() sdk.ValAddress // validator operator address
	GetShares() sdkmath.LegacyDec     // amount of validator's shares held in this delegation
}

// ValidatorI expected validator functions
type ValidatorI interface {
	IsJailed() bool                                             // whether the validator is jailed
	GetMoniker() string                                         // moniker of the validator
	GetStatus() BondStatus                                      // status of the validator
	IsBonded() bool                                             // check if has a bonded status
	IsUnbonded() bool                                           // check if has status unbonded
	IsUnbonding() bool                                          // check if has status unbonding
	GetOperator() sdk.ValAddress                                // operator address to receive/return validators coins
	ConsPubKey() (cryptotypes.PubKey, error)                    // validation consensus pubkey (cryptotypes.PubKey)
	TmConsPublicKey() (tmprotocrypto.PublicKey, error)          // validation consensus pubkey (Tendermint)
	GetConsAddr() (sdk.ConsAddress, error)                      // validation consensus address
	GetTokens() sdkmath.Int                                     // validation tokens
	GetBondedTokens() sdkmath.Int                               // validator bonded tokens
	GetConsensusPower(sdkmath.Int) int64                        // validation power in tendermint
	GetCommission() sdkmath.LegacyDec                           // validator commission rate
	GetMinSelfDelegation() sdkmath.Int                          // validator minimum self delegation
	GetDelegatorShares() sdkmath.LegacyDec                      // total outstanding delegator shares
	TokensFromShares(sdk.Dec) sdkmath.LegacyDec                 // token worth of provided delegator shares
	TokensFromSharesTruncated(sdk.Dec) sdkmath.LegacyDec        // token worth of provided delegator shares, truncated
	TokensFromSharesRoundUp(sdk.Dec) sdkmath.LegacyDec          // token worth of provided delegator shares, rounded up
	SharesFromTokens(amt sdkmath.Int) (sdk.Dec, error)          // shares worth of delegator's bond
	SharesFromTokensTruncated(amt sdkmath.Int) (sdk.Dec, error) // truncated shares worth of delegator's bond
}
