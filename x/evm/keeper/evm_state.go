// Copyright (c) 2023-2024 Nibi, Inc.
package keeper

import (
	"fmt"
	"math/big"

	"github.com/NibiruChain/collections"
	"github.com/cosmos/cosmos-sdk/codec"
	sdkstore "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core/types"

	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/evm"
)

type (
	CodeHash = []byte
)

// EvmState isolates the key-value stores (collections) for the x/evm module.
type EvmState struct {
	ModuleParams collections.Item[evm.Params]

	// ContractBytecode: Map from (byte)code hash -> contract bytecode
	ContractBytecode collections.Map[CodeHash, []byte]

	// AccState: Map from eth address (account) and hash of a state key -> smart
	// contract state. Each contract essentially has its own key-value store.
	//
	//  - primary key (PK): (EthAddr+EthHash). The contract address and hash for
	//    a piece of state in that contract forms the primary key.
	//  - value (V): State value bytes.
	AccState collections.Map[
		collections.Pair[gethcommon.Address, gethcommon.Hash], // account (EthAddr) + state key (EthHash)
		[]byte,
	]

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

func (state EvmState) SetAccCode(ctx sdk.Context, codeHash, code []byte) {
	if len(code) > 0 {
		state.ContractBytecode.Insert(ctx, codeHash, code)
	} else {
		// Ignore collections "key not found" error because erasing an empty
		// state is perfectly valid here.
		_ = state.ContractBytecode.Delete(ctx, codeHash)
	}
}

func (state EvmState) GetContractBytecode(
	ctx sdk.Context, codeHash []byte,
) (code []byte) {
	return state.ContractBytecode.GetOr(ctx, codeHash, []byte{})
}

// GetParams returns the total set of evm parameters.
func (k Keeper) GetParams(ctx sdk.Context) (params evm.Params) {
	params, _ = k.EvmState.ModuleParams.Get(ctx)
	return params
}

// SetParams: Setter for the module parameters.
func (k Keeper) SetParams(ctx sdk.Context, params evm.Params) (err error) {
	if params.CreateFuntokenFee.IsNegative() {
		return fmt.Errorf("createFuntokenFee cannot be negative: %s", params.CreateFuntokenFee)
	}

	k.EvmState.ModuleParams.Set(ctx, params)
	return
}

// SetState updates contract storage and deletes if the value is empty.
func (state EvmState) SetAccState(
	ctx sdk.Context, addr gethcommon.Address, stateKey gethcommon.Hash, stateValue []byte,
) {
	if len(stateValue) != 0 {
		state.AccState.Insert(ctx, collections.Join(addr, stateKey), stateValue)
	} else {
		_ = state.AccState.Delete(ctx, collections.Join(addr, stateKey))
	}
}

// GetState: Implements `statedb.Keeper` interface: Loads smart contract state.
func (k *Keeper) GetState(ctx sdk.Context, addr gethcommon.Address, stateKey gethcommon.Hash) gethcommon.Hash {
	return gethcommon.BytesToHash(k.EvmState.AccState.GetOr(
		ctx,
		collections.Join(addr, stateKey),
		[]byte{},
	))
}

// GetBlockBloomTransient returns bloom bytes for the current block height
func (state EvmState) GetBlockBloomTransient(ctx sdk.Context) *big.Int {
	bloomBz, err := state.BlockBloom.Get(ctx)
	if err != nil {
		return big.NewInt(0)
	}
	return new(big.Int).SetBytes(bloomBz)
}

func (state EvmState) CalcBloomFromLogs(
	ctx sdk.Context, newLogs []*gethcore.Log,
) (bloom gethcore.Bloom) {
	if len(newLogs) > 0 {
		bloomInt := state.GetBlockBloomTransient(ctx)
		bloomInt.Or(bloomInt, big.NewInt(0).SetBytes(gethcore.LogsBloom(newLogs)))
		bloom = gethcore.BytesToBloom(bloomInt.Bytes())
	}
	return bloom
}

// GetAccNonce returns the sequence number of an account, returns 0 if not exists.
func (k Keeper) GetAccNonce(ctx sdk.Context, addr gethcommon.Address) uint64 {
	nibiruAddr := sdk.AccAddress(addr.Bytes())
	acct := k.accountKeeper.GetAccount(ctx, nibiruAddr)
	if acct == nil {
		return 0
	}
	return acct.GetSequence()
}
