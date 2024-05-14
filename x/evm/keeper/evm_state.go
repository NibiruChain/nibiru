// Copyright (c) 2023-2024 Nibi, Inc.
package keeper

import (
	"github.com/NibiruChain/collections"
	"github.com/cosmos/cosmos-sdk/codec"
	sdkstore "github.com/cosmos/cosmos-sdk/store/types"
	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/NibiruChain/nibiru/eth"
	"github.com/NibiruChain/nibiru/x/evm"
)

type AccStatePrimaryKey = collections.Pair[gethcommon.Address, gethcommon.Hash]
type CodeHash = []byte

// EvmState isolates the key-value stores (collections) for the x/evm module.
type EvmState struct {
	ModuleParams collections.Item[evm.Params]

	// ContractBytecode: Map from (byte)code hash -> contract bytecode
	ContractBytecode collections.Map[CodeHash, []byte]

	// AccState: Map from eth address (account) and hash of a state key -> smart
	// contract state. Each contract essentially has its own key-value store.
	//
	//  - primary key (PK): (EthAddr+EthHash). The contract is the primary key
	//  because there's exactly one deployer and withdrawer.
	//  - value (V): State value bytes.
	AccState collections.Map[
		AccStatePrimaryKey, // account (EthAddr) + state key (EthHash)
		[]byte,
	]

	// BlockGasUsed: Gas used by Ethereum txs in the block (transient).
	BlockGasUsed collections.ItemTransient[uint64]
	// BlockLogSize: EVM tx log size for the block (transient).
	BlockLogSize collections.ItemTransient[uint64]
	// BlockTxIndex: EVM tx index for the block (transient).
	BlockTxIndex collections.ItemTransient[uint64]
	// BlockBloom: Bloom filters.
	BlockBloom collections.ItemTransient[[]byte]
}

func (k *Keeper) EVMState() EvmState { return k.EvmState }

func NewEvmState(
	cdc codec.BinaryCodec,
	storeKey sdkstore.StoreKey,
	storeKeyTransient sdkstore.StoreKey,
) EvmState {
	return EvmState{
		ModuleParams: collections.NewItem(
			storeKey, evm.KeyPrefixParams,
			collections.ProtoValueEncoder[evm.Params](cdc),
		),
		ContractBytecode: collections.NewMap(
			storeKey, evm.KeyPrefixAccCodes,
			eth.KeyEncoderBytes,
			eth.ValueEncoderBytes,
		),
		AccState: collections.NewMap(
			storeKey, evm.KeyPrefixAccState,
			collections.PairKeyEncoder(eth.KeyEncoderEthAddr, eth.KeyEncoderEthHash),
			eth.ValueEncoderBytes,
		),
		BlockGasUsed: collections.NewItemTransient(
			storeKeyTransient,
			evm.NamespaceBlockGasUsed,
			collections.Uint64ValueEncoder,
		),
		BlockLogSize: collections.NewItemTransient(
			storeKeyTransient,
			evm.NamespaceBlockLogSize,
			collections.Uint64ValueEncoder,
		),
		BlockBloom: collections.NewItemTransient(
			storeKeyTransient,
			evm.NamespaceBlockBloom,
			eth.ValueEncoderBytes,
		),
		BlockTxIndex: collections.NewItemTransient(
			storeKeyTransient,
			evm.NamespaceBlockTxIndex,
			collections.Uint64ValueEncoder,
		),
	}
}
