// Copyright (c) 2023-2024 Nibi, Inc.
package evmmodule

import (
	"bytes"
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/NibiruChain/nibiru/eth"
	"github.com/NibiruChain/nibiru/x/evm"
	"github.com/NibiruChain/nibiru/x/evm/keeper"
)

// InitGenesis initializes genesis state based on exported genesis
func InitGenesis(
	ctx sdk.Context,
	k *keeper.Keeper,
	accountKeeper evm.AccountKeeper,
	genState evm.GenesisState,
) []abci.ValidatorUpdate {
	k.SetParams(ctx, genState.Params)

	// GetModuleAccount actually creates a new module account with permissions under the hood
	if addr := accountKeeper.GetModuleAccount(ctx, evm.ModuleName); addr == nil {
		panic("the EVM module account has not been set")
	}

	for _, account := range genState.Accounts {
		address := gethcommon.HexToAddress(account.Address)
		accAddress := sdk.AccAddress(address.Bytes())
		// check that the EVM balance the matches the account balance
		acc := accountKeeper.GetAccount(ctx, accAddress)
		if acc == nil {
			panic(fmt.Errorf("account not found for address %s", account.Address))
		}

		ethAcct, ok := acc.(eth.EthAccountI)
		if !ok {
			panic(
				fmt.Errorf("account %s must be an EthAccount interface, got %T",
					account.Address, acc,
				),
			)
		}
		code := gethcommon.Hex2Bytes(account.Code)
		codeHash := crypto.Keccak256Hash(code)

		// we ignore the empty Code hash checking, see ethermint PR#1234
		if len(account.Code) != 0 && !bytes.Equal(ethAcct.GetCodeHash().Bytes(), codeHash.Bytes()) {
			s := "the evm state code doesn't match with the codehash\n"
			panic(fmt.Sprintf("%s account: %s , evm state codehash: %v, ethAccount codehash: %v, evm state code: %s\n",
				s, account.Address, codeHash, ethAcct.GetCodeHash(), account.Code))
		}

		k.SetCode(ctx, codeHash.Bytes(), code)

		for _, storage := range account.Storage {
			k.SetState(ctx, address, gethcommon.HexToHash(storage.Key), gethcommon.HexToHash(storage.Value).Bytes())
		}
	}

	return []abci.ValidatorUpdate{}
}

// ExportGenesis exports genesis state of the EVM module
func ExportGenesis(ctx sdk.Context, k *keeper.Keeper, ak evm.AccountKeeper) *evm.GenesisState {
	// TODO: impl ExportGenesis
	return &evm.GenesisState{
		Accounts: []evm.GenesisAccount{},
		Params:   evm.Params{},
	}
}
