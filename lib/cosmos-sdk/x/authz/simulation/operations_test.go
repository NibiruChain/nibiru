package simulation_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/codec"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	"github.com/cosmos/cosmos-sdk/x/authz/simulation"
)

func TestWeightedOperations(t *testing.T) {
	encCfg := moduletestutil.MakeTestEncodingConfig()
	appParams := make(simtypes.AppParams)

	weightedOps := simulation.WeightedOperations(
		encCfg.InterfaceRegistry,
		appParams,
		codec.NewProtoCodec(encCfg.InterfaceRegistry),
		nil,
		nil,
		authzkeeper.Keeper{},
	)

	if len(weightedOps) != 0 {
		t.Fatalf("expected authz simulation operations to be disabled, got %d", len(weightedOps))
	}
}
