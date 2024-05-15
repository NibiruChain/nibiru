// Copyright (c) 2023-2024 Nibi, Inc.
package keeper

import (
	// "github.com/NibiruChain/nibiru/x/evm"
	"math/big"

	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/NibiruChain/nibiru/x/evm"
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

	bankKeeper    evm.BankKeeper
	accountKeeper evm.AccountKeeper
}

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey, transientKey storetypes.StoreKey,
	authority sdk.AccAddress,
	accKeeper evm.AccountKeeper,
	bankKeeper evm.BankKeeper,
) Keeper {
	if err := sdk.VerifyAddressFormat(authority); err != nil {
		panic(err)
	}
	return Keeper{
		cdc:           cdc,
		storeKey:      storeKey,
		transientKey:  transientKey,
		authority:     authority,
		EvmState:      NewEvmState(cdc, storeKey, transientKey),
		accountKeeper: accKeeper,
		bankKeeper:    bankKeeper,
	}
}

// GetEvmGasBalance: Implements `evm.EVMKeeper` from
// "github.com/NibiruChain/nibiru/app/ante/evm": Load account's balance of gas
// tokens for EVM execution
func (k *Keeper) GetEvmGasBalance(ctx sdk.Context, addr gethcommon.Address) *big.Int {
	nibiruAddr := sdk.AccAddress(addr.Bytes())
	evmParams := k.GetParams(ctx)
	evmDenom := evmParams.GetEvmDenom()
	// if node is pruned, params is empty. Return invalid value
	if evmDenom == "" {
		return big.NewInt(-1)
	}
	coin := k.bankKeeper.GetBalance(ctx, nibiruAddr, evmDenom)
	return coin.Amount.BigInt()
}
