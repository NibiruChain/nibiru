// Package exported holds the param store interfaces of the Cosmos SDK
// x/consensus module.
//
// Deprecated: NibiruApp no longer uses this module. Nibiru's own x/consensus
// module (github.com/NibiruChain/nibiru/v2/x/consensus) owns the "consensus"
// store, serves the Params query, and rejects MsgUpdateParams, because the
// x-consensus contract owns the consensus params. The types moved to
// github.com/NibiruChain/nibiru/v2/x/consensus/types, which this package
// imports. It is kept only for lib's own depinject test apps (the
// testutil/app_config.go files and testutil/configurator) and the ibc-go
// simapps, and will be removed together with that test wiring.
package exported

import (
	tmproto "github.com/NibiruChain/nibiru/v2/bft/proto/tendermint/types"

	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
)

type (
	// ParamStore defines an interface that implements the legacy x/params Subspace
	// type.
	//
	// NOTE: This is used solely for migration of x/params managed parameters.
	ParamStore interface {
		Get(ctx sdk.Context, key []byte, ptr interface{})
	}

	// ConsensusParamSetter defines the interface fulfilled by BaseApp's
	// ParamStore which allows setting its appVersion field.
	ConsensusParamSetter interface {
		Get(ctx sdk.Context) (*tmproto.ConsensusParams, error)
		Has(ctx sdk.Context) bool
		Set(ctx sdk.Context, cp *tmproto.ConsensusParams)
	}
)
