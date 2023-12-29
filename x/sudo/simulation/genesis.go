package simulation

// DONTCOVER

import (
	"encoding/json"
	"fmt"
	"math/rand"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/NibiruChain/nibiru/x/sudo/types"
)

// Simulation parameter constants
const (
	CommunityTax    = "community_tax"
	WithdrawEnabled = "withdraw_enabled"
)

// GenCommunityTax randomized CommunityTax
func GenCommunityTax(r *rand.Rand) math.LegacyDec {
	return sdk.NewDecWithPrec(1, 2).Add(sdk.NewDecWithPrec(int64(r.Intn(30)), 2))
}

// GenWithdrawEnabled returns a randomized WithdrawEnabled parameter.
func GenWithdrawEnabled(r *rand.Rand) bool {
	return r.Int63n(101) <= 95 // 95% chance of withdraws being enabled
}

// RandomizedGenState generates a random GenesisState for distribution
func RandomizedGenState(simState *module.SimulationState) {
	genState := types.GenesisState{
		Sudoers: types.Sudoers{
			Root:      simState.Accounts[0].Address.String(),
			Contracts: []string{},
		},
	}

	bz, err := json.MarshalIndent(&genState, "", " ")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Selected randomly generated x/sudo parameters:\n%s\n", bz)
	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(&genState)
}
