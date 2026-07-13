package exported

import (
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	paramtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/params/types"
)

// GenesisBalance defines a genesis balance interface that allows for account
// address and balance retrieval.
type GenesisBalance interface {
	GetAddress() sdk.AccAddress
	GetCoins() sdk.Coins
}

type (
	ParamSet = paramtypes.ParamSet

	// Subspace defines an interface that implements the legacy x/params Subspace
	// type.
	//
	// NOTE: This is used solely for migration of x/params managed parameters.
	Subspace interface {
		GetParamSet(ctx sdk.Context, ps ParamSet)
		Get(ctx sdk.Context, key []byte, ptr interface{})
	}
)
