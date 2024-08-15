// Copyright (c) 2023-2024 Nibi, Inc.
package keeper

import (
	"github.com/NibiruChain/collections"
	sdkcodec "github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/evm"
)

// FunTokenState isolates the key-value stores (collections) for fungible token
// mappings. This struct is written as an extension of the default indexed map to
// add utility functions.
type FunTokenState struct {
	collections.IndexedMap[[]byte, evm.FunToken, IndexesFunToken]
}

func NewFunTokenState(
	cdc sdkcodec.BinaryCodec,
	storeKey storetypes.StoreKey,
) FunTokenState {
	primaryKeyEncoder := eth.KeyEncoderBytes
	valueEncoder := collections.ProtoValueEncoder[evm.FunToken](cdc)
	idxMap := collections.NewIndexedMap(
		storeKey, evm.KeyPrefixFunTokens, primaryKeyEncoder, valueEncoder,
		IndexesFunToken{
			ERC20Addr: collections.NewMultiIndex(
				storeKey, evm.KeyPrefixFunTokenIdxErc20,
				eth.KeyEncoderEthAddr, //  indexing key (IK): ERC-20 addr
				primaryKeyEncoder,
				func(v evm.FunToken) gethcommon.Address {
					return v.Erc20Addr.Addr()
				},
			),
			BankDenom: collections.NewMultiIndex(
				storeKey, evm.KeyPrefixFunTokenIdxBankDenom,
				collections.StringKeyEncoder, //  indexing key (IK): Coin denom
				primaryKeyEncoder,
				func(v evm.FunToken) string { return v.BankDenom },
			),
		},
	)
	return FunTokenState{IndexedMap: idxMap}
}

func (idxs IndexesFunToken) IndexerList() []collections.Indexer[[]byte, evm.FunToken] {
	return []collections.Indexer[[]byte, evm.FunToken]{
		idxs.ERC20Addr,
		idxs.BankDenom,
	}
}

// IndexesFunToken: Abstraction for indexing over the FunToken store.
type IndexesFunToken struct {
	// ERC20Addr (MultiIndex): Index FunToken by ERC-20 contract address.
	//  - indexing key (IK): ERC-20 addr
	//  - primary key (PK): FunToken ID
	//  - value (V): FunToken value
	ERC20Addr collections.MultiIndex[gethcommon.Address, []byte, evm.FunToken]

	// BankDenom (MultiIndex): Index FunToken by coin denomination
	//  - indexing key (IK): Coin denom
	//  - primary key (PK): FunToken ID
	//  - value (V): FunToken value
	BankDenom collections.MultiIndex[string, []byte, evm.FunToken]
}

// Insert adds an [evm.FunToken] to state with defensive validation. Errors if
// the given inputs would result in a corrupted [evm.FunToken].
func (fun FunTokenState) SafeInsert(
	ctx sdk.Context, erc20 gethcommon.Address, bankDenom string, isMadeFromCoin bool,
) error {
	funtoken := evm.NewFunToken(erc20, bankDenom, isMadeFromCoin)
	if err := funtoken.Validate(); err != nil {
		return err
	}
	fun.Insert(ctx, funtoken.ID(), funtoken)
	return nil
}
