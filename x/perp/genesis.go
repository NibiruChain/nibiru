package perp

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/perp/keeper"
	"github.com/NibiruChain/nibiru/x/perp/types"
)

// InitGenesis initializes the capability module's state from a provided genesis
// state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// check fee pool balance
	feePoolAcc := k.AccountKeeper.GetModuleAccount(ctx, types.FeePoolModuleAccount)
	if feePoolAcc == nil {
		panic(fmt.Errorf("%s account was not created", types.FeePoolModuleAccount))
	}
	if balance := k.BankKeeper.GetAllBalances(ctx, feePoolAcc.GetAddress()); !balance.IsEqual(genState.FeePoolBalance) {
		panic(
			fmt.Errorf(
				"%s registered balance does not match bank balance: %s <-> %s",
				types.FeePoolModuleAccount, genState.FeePoolBalance, balance))
	}

	// check vault balance
	vaultAcc := k.AccountKeeper.GetModuleAccount(ctx, types.VaultModuleAccount)
	if vaultAcc == nil {
		panic(fmt.Errorf("%s account was not created", types.VaultModuleAccount))
	}
	if balance := k.BankKeeper.GetAllBalances(ctx, vaultAcc.GetAddress()); !balance.IsEqual(genState.VaultBalance) {
		panic(
			fmt.Errorf(
				"%s registered balance does not match bank balance: %s <-> %s",
				types.VaultModuleAccount, genState.VaultBalance, balance))
	}
	// check perp ef balance
	perpEFAccount := k.AccountKeeper.GetModuleAccount(ctx, types.PerpEFModuleAccount)
	if perpEFAccount == nil {
		panic(fmt.Errorf("%s account was not created", types.PerpEFModuleAccount))
	}
	if balance := k.BankKeeper.GetAllBalances(ctx, perpEFAccount.GetAddress()); !balance.IsEqual(genState.PerpEfBalance) {
		panic(
			fmt.Errorf(
				"%s registered balance does not match bank balance: %s <-> %s",
				types.PerpEFModuleAccount, genState.PerpEfBalance, balance))
	}

	// set pair metadata
	for _, p := range genState.PairMetadata {
		k.PairMetadataState(ctx).Set(p)
	}

	// create positions
	for _, p := range genState.Positions {
		err := k.PositionsState(ctx).Create(p)
		if err != nil {
			panic(fmt.Errorf("unable to re-create position %s: %w", p, err))
		}
	}

	// set params
	k.SetParams(ctx, genState.Params)

	// set prepaid debt position
	for _, pbd := range genState.PrepaidBadDebts {
		k.PrepaidBadDebtState(ctx).Set(pbd.Denom, pbd.Amount)
	}

	// set whitelist
	for _, whitelist := range genState.WhitelistedAddresses {
		addr, err := sdk.AccAddressFromBech32(whitelist)
		if err != nil {
			panic(err)
		}
		k.WhitelistState(ctx).Add(addr)
	}
}

// ExportGenesis returns the capability module's exported genesis.
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := new(types.GenesisState)

	genesis.Params = k.GetParams(ctx)
	// perp ef balance
	perpEFAccount := k.AccountKeeper.GetModuleAccount(ctx, types.PerpEFModuleAccount)
	if perpEFAccount == nil {
		panic(fmt.Errorf("%s module account does not exist", types.PerpEFModuleAccount))
	}
	perpEFBalance := k.BankKeeper.GetAllBalances(ctx, perpEFAccount.GetAddress())
	genesis.PerpEfBalance = perpEFBalance
	// fee pool balance
	feePoolAccount := k.AccountKeeper.GetModuleAccount(ctx, types.FeePoolModuleAccount)
	if feePoolAccount == nil {
		panic(fmt.Errorf("%s module account does not exist", types.FeePoolModuleAccount))
	}
	feePoolBalance := k.BankKeeper.GetAllBalances(ctx, feePoolAccount.GetAddress())
	genesis.FeePoolBalance = feePoolBalance
	// vault balance
	vaultAccount := k.AccountKeeper.GetModuleAccount(ctx, types.VaultModuleAccount)
	if vaultAccount == nil {
		panic(fmt.Errorf("%s module account does not exist", types.VaultModuleAccount))
	}
	vaultAccountBalance := k.BankKeeper.GetAllBalances(ctx, vaultAccount.GetAddress())
	genesis.VaultBalance = vaultAccountBalance

	// export positions
	k.PositionsState(ctx).Iterate(func(position *types.Position) (stop bool) {
		genesis.Positions = append(genesis.Positions, position)
		return false
	})

	// export prepaid bad debt
	k.PrepaidBadDebtState(ctx).Iterate(func(denom string, amount sdk.Int) (stop bool) {
		genesis.PrepaidBadDebts = append(genesis.PrepaidBadDebts, &types.PrepaidBadDebt{
			Denom:  denom,
			Amount: amount,
		})
		return false
	})

	// export whitelist
	k.WhitelistState(ctx).Iterate(func(addr sdk.AccAddress) (stop bool) {
		genesis.WhitelistedAddresses = append(genesis.WhitelistedAddresses, addr.String())
		return false
	})

	// export pairMetadata
	metadata := k.PairMetadataState(ctx).GetAll()
	genesis.PairMetadata = metadata

	return genesis
}
