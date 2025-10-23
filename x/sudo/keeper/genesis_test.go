package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/x/nutil/set"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/sudo"
)

// TestExportInitGenesis_Roundtrip tests the complete export/import cycle for the sudo module.
// This test verifies that:
// 1. Initial state can be set with Sudoers and ZeroGasActors
// 2. Genesis can be exported and is valid
// 3. Genesis can be imported into a fresh app
// 4. State is preserved exactly after roundtrip
// 5. Functional operations (CheckPermissions, queries) work after import
func (s *Suite) TestExportInitGenesis_Roundtrip() {
	// Setup Phase: Create app with complete initial state
	_, k, ctx := setup()

	// Generate test addresses
	rootAddr := testutil.AccAddress()
	_, contractAddrs := testutil.PrivKeyAddressPairs(3)
	_, senderAddrs := testutil.PrivKeyAddressPairs(2)
	_, zeroGasContractAddrs := testutil.PrivKeyAddressPairs(2)

	// Set Sudoers state
	sudoers := sudo.Sudoers{
		Root:      rootAddr.String(),
		Contracts: []string{contractAddrs[0].String(), contractAddrs[1].String(), contractAddrs[2].String()},
	}
	k.Sudoers.Set(ctx, sudoers)

	// Set ZeroGasActors state
	zeroGasActors := sudo.ZeroGasActors{
		Senders:   []string{senderAddrs[0].String(), senderAddrs[1].String()},
		Contracts: []string{zeroGasContractAddrs[0].String(), zeroGasContractAddrs[1].String()},
	}
	k.ZeroGasActors.Set(ctx, zeroGasActors)

	// Verify initial state works
	// CheckPermissions should succeed for root and contracts
	s.NoError(k.CheckPermissions(rootAddr, ctx))
	for _, contractAddr := range contractAddrs {
		s.NoError(k.CheckPermissions(contractAddr, ctx))
	}

	// CheckPermissions should fail for non-sudoer
	nonSudoer := testutil.AccAddress()
	s.Error(k.CheckPermissions(nonSudoer, ctx))

	// Export Phase: Export genesis and verify it's valid
	exported := k.ExportGenesis(ctx)
	s.NotNil(exported)
	s.NoError(exported.Validate())

	// Import Phase: Create fresh app and import genesis
	nibiru2, ctx2 := testapp.NewNibiruTestAppAndContext()
	nibiru2.SudoKeeper.InitGenesis(ctx2, *exported)

	// Verification Phase: Export from new app and compare
	reExported := nibiru2.SudoKeeper.ExportGenesis(ctx2)
	s.NotNil(reExported)

	// Compare genesis states are identical
	s.Equal(exported.Sudoers.Root, reExported.Sudoers.Root)

	// Compare contracts using set equality (order-independent)
	originalContracts := set.New(exported.Sudoers.Contracts...)
	reExportedContracts := set.New(reExported.Sudoers.Contracts...)
	s.True(originalContracts.Equals(reExportedContracts))

	// Compare ZeroGasActors (handle nil case)
	if exported.ZeroGasActors != nil {
		s.Require().NotNil(reExported.ZeroGasActors)
		s.Equal(exported.ZeroGasActors.Senders, reExported.ZeroGasActors.Senders)
		s.Equal(exported.ZeroGasActors.Contracts, reExported.ZeroGasActors.Contracts)
	} else {
		s.Nil(reExported.ZeroGasActors)
	}

	// Functional verification: CheckPermissions still works
	s.NoError(nibiru2.SudoKeeper.CheckPermissions(rootAddr, ctx2))
	for _, contractAddr := range contractAddrs {
		s.NoError(nibiru2.SudoKeeper.CheckPermissions(contractAddr, ctx2))
	}
	s.Error(nibiru2.SudoKeeper.CheckPermissions(nonSudoer, ctx2))

	// Functional verification: Query ZeroGasActors returns correct data
	queryResp, err := nibiru2.SudoKeeper.QueryZeroGasActors(sdk.WrapSDKContext(ctx2), nil)
	s.NoError(err)
	s.Equal(zeroGasActors.Senders, queryResp.Actors.Senders)
	s.Equal(zeroGasActors.Contracts, queryResp.Actors.Contracts)
}
