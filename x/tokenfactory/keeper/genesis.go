package keeper

import (
	"github.com/NibiruChain/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/NibiruChain/nibiru/x/tokenfactory/types"
)

// InitGenesis initializes the tokenfactory module's state from a provided genesis
// state.
func (k Keeper) InitGenesis(ctx sdk.Context, genState types.GenesisState) {
	k.CreateModuleAccount(ctx)

	if err := genState.Validate(); err != nil {
		panic(err)
	}

	k.Store.ModuleParams.Set(ctx, genState.Params)

	for _, genDenom := range genState.GetFactoryDenoms() {
		// We don't need to validate the struct again here because it's
		// performed inside of the genState.Validate() execution above.
		k.Store.unsafeGenesisInsertDenom(ctx, genDenom)
	}
}

// ExportGenesis returns the tokenfactory module's exported genesis.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	genDenoms := []types.GenesisDenom{}
	// iterator := k.GetAllDenomsIterator(ctx)
	iter := k.Store.Denoms.Iterate(ctx, collections.Range[string]{})
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		denom := iter.Value()

		authorityMetadata, err := k.Store.GetDenomAuthorityMetadata(
			ctx, denom.Denom().String())
		if err != nil {
			panic(err)
		}

		genDenoms = append(genDenoms, types.GenesisDenom{
			Denom:             denom.Denom().String(),
			AuthorityMetadata: authorityMetadata,
		})
	}

	moduleParams, err := k.Store.ModuleParams.Get(ctx)
	if err != nil {
		panic(err)
	}
	return &types.GenesisState{
		FactoryDenoms: genDenoms,
		Params:        moduleParams,
	}
}

// CreateModuleAccount creates a module account with minting and burning
// capabilities This account isn't intended to store any coins, it purely mints
// and burns them on behalf of the admin of respective denoms, and sends to the
// relevant address.
func (k Keeper) CreateModuleAccount(ctx sdk.Context) {
	moduleAcc := authtypes.NewEmptyModuleAccount(types.ModuleName, authtypes.Minter, authtypes.Burner)
	k.accountKeeper.SetModuleAccount(ctx, moduleAcc)
}
