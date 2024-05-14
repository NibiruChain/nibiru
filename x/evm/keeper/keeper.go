// Copyright (c) 2023-2024 Nibi, Inc.
package keeper

import (
	// "github.com/NibiruChain/nibiru/x/evm"
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type Keeper struct {
	cdc codec.BinaryCodec
	// storeKey: For persistent storage of EVM state.
	storeKey storetypes.StoreKey
	// transientKey: Store key that resets every block during Commit
	transientKey storetypes.StoreKey

	// EvmState isolates the key-value stores (collections) for the x/evm module.
	EvmState EvmState

	// the address capable of executing a MsgUpdateParams message. Typically, this should be the x/gov module account.
	authority sdk.AccAddress
}

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey, transientKey storetypes.StoreKey,
	authority sdk.AccAddress,
) Keeper {
	if err := sdk.VerifyAddressFormat(authority); err != nil {
		panic(err)
	}
	return Keeper{
		cdc:          cdc,
		storeKey:     storeKey,
		transientKey: transientKey,
		authority:    authority,
		EvmState:     NewEvmState(cdc, storeKey, transientKey),
	}
}
