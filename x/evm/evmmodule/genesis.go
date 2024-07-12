// Copyright (c) 2023-2024 Nibi, Inc.
package evmmodule

import (
	"bytes"
	"fmt"

	"github.com/NibiruChain/collections"
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

	// Note that "GetModuleAccount" initializes the module account with permissions
	// under the hood if it did not already exist. This is important because the
	// EVM module needs to be able to send and receive funds during MsgEthereumTx
	if evmModule := accountKeeper.GetModuleAccount(ctx, evm.ModuleName); evmModule == nil {
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
	var genesisAccounts []evm.GenesisAccount

	iter := k.EvmState.AccState.Iterate(ctx, collections.PairRange[gethcommon.Address, gethcommon.Hash]{})
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		key := iter.Key()
		address := key.K1()
		hash := key.K1()
		val := iter.Value()

		fmt.Println("---")
		fmt.Println("Address: ", address.String())
		fmt.Println("Hash: ", hash.String())
		fmt.Println("Value: ", eth.BytesToHex(val))

		//var storage evm.Storage
		//k.ForEachStorage(ctx, address, func(key, value gethcommon.Hash) bool {
		//	storage = append(storage, evm.NewStateFromEthHashes(key, value))
		//	return false
		//})

		//account := k.GetAccountWithoutBalance(ctx, address)
		//code := eth.BytesToHex(k.GetCode(ctx, gethcommon.BytesToHash(account.CodeHash)))
		//
		//genesisAccount := evm.GenesisAccount{
		//	Address: address.String(),
		//	Code:    code,
		//	Storage: storage,
		//}
		//
		//genesisAccounts = append(genesisAccounts, genesisAccount)
	}

	return &evm.GenesisState{
		Params:           k.GetParams(ctx),
		Accounts:         genesisAccounts,
		FuntokenMappings: make([]evm.FunToken, 0),
	}
}
