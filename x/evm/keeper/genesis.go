package keeper

// Copyright (c) 2023-2024 Nibi, Inc.

import (
	"fmt"

	"github.com/NibiruChain/collections"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/evm"
)

// InitGenesis initializes genesis state based on exported genesis
func (k *Keeper) InitGenesis(
	ctx sdk.Context,
	genState evm.GenesisState,
) []abci.ValidatorUpdate {
	err := k.SetParams(ctx, genState.Params)
	if err != nil {
		panic(fmt.Errorf("failed to set params: %w", err))
	}

	// Note that "GetModuleAccount" initializes the module account with permissions
	// under the hood if it did not already exist. This is important because the
	// EVM module needs to be able to send and receive funds during MsgEthereumTx
	if evmModule := k.accountKeeper.GetModuleAccount(ctx, evm.ModuleName); evmModule == nil {
		panic("the EVM module account has not been set")
	}

	// Create evm contracts from genstate.Accounts
	for _, account := range genState.Accounts {
		err := k.ImportGenesisAccount(ctx, account)
		if err != nil {
			panic(err)
		}
	}

	// Create fungible token mappings
	for _, funToken := range genState.FuntokenMappings {
		err := k.FunTokens.SafeInsert(
			ctx, gethcommon.HexToAddress(funToken.Erc20Addr.String()), funToken.BankDenom, funToken.IsMadeFromCoin,
		)
		if err != nil {
			panic(fmt.Errorf("failed creating funtoken: %w", err))
		}
	}

	return []abci.ValidatorUpdate{}
}

// ExportGenesis exports genesis state of the EVM module
func (k *Keeper) ExportGenesis(ctx sdk.Context) *evm.GenesisState {
	var genesisAccounts []evm.GenesisAccount

	// 1. Export EVM contacts
	// TODO: find the way to get eth contract addresses from the evm keeper
	allAccounts := k.accountKeeper.GetAllAccounts(ctx)
	for _, acc := range allAccounts {
		ethAcct, ok := acc.(eth.EthAccountI)
		if ok {
			address := ethAcct.EthAddress()
			codeHash := ethAcct.GetCodeHash()
			code, err := k.EvmState.ContractBytecode.Get(ctx, codeHash.Bytes())
			if err != nil {
				// Not a contract
				continue
			}
			var storage evm.Storage

			k.ForEachStorage(ctx, address, func(key, value gethcommon.Hash) bool {
				storage = append(storage, evm.NewStateFromEthHashes(key, value))
				return true
			})
			genesisAccounts = append(genesisAccounts, evm.GenesisAccount{
				Address: address.String(),
				Code:    eth.BytesToHex(code),
				Storage: storage,
			})
		}
	}

	// 2. Export Fungible tokens
	var funTokens []evm.FunToken
	iter := k.FunTokens.Iterate(ctx, collections.Range[[]byte]{})
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		funTokens = append(funTokens, iter.Value())
	}

	return &evm.GenesisState{
		Params:           k.GetParams(ctx),
		Accounts:         genesisAccounts,
		FuntokenMappings: funTokens,
	}
}
