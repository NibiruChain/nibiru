// Copyright (c) 2023-2024 Nibi, Inc.
package keeper

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/cometbft/cometbft/libs/log"

	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/evm"
)

type contextKey string

const SimulationContextKey contextKey = "evm_simulation"

type Keeper struct {
	cdc codec.BinaryCodec
	// storeKey: For persistent storage of EVM state.
	storeKey storetypes.StoreKey
	// transientKey: Store key that resets every block during Commit
	transientKey storetypes.StoreKey

	// EvmState isolates the key-value stores (collections) for the x/evm module.
	EvmState EvmState

	// FunTokens isolates the key-value stores (collections) for fungible token
	// mappings.
	FunTokens FunTokenState

	// the address capable of executing a MsgUpdateParams message. Typically,
	// this should be the x/gov module account.
	authority sdk.AccAddress

	Bank          *NibiruBankKeeper
	accountKeeper evm.AccountKeeper
	stakingKeeper evm.StakingKeeper
	sudoKeeper    evm.SudoKeeper

	// tracer: Configures the output type for a geth `vm.EVMLogger`. Tracer types
	// include "access_list", "json", "struct", and "markdown". If any other
	// value is used, a no operation tracer is set.
	tracer string
}

// NewKeeper is a constructor for an x/evm [Keeper]. This function is necessary
// because the [Keeper] struct has private fields.
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey, transientKey storetypes.StoreKey,
	authority sdk.AccAddress,
	accKeeper evm.AccountKeeper,
	bankKeeper *NibiruBankKeeper,
	stakingKeeper evm.StakingKeeper,
	tracer string,
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
		FunTokens:     NewFunTokenState(cdc, storeKey),
		accountKeeper: accKeeper,
		Bank:          bankKeeper,
		stakingKeeper: stakingKeeper,
		tracer:        tracer,
	}
}

// GetEvmGasBalance: Used in the EVM Ante Handler,
// "github.com/NibiruChain/nibiru/v2/app/evmante": Load account's balance of gas
// tokens for EVM execution in EVM denom units.
func (k *Keeper) GetEvmGasBalance(ctx sdk.Context, addr gethcommon.Address) (balance *big.Int) {
	nibiruAddr := sdk.AccAddress(addr.Bytes())
	return k.Bank.GetBalance(ctx, nibiruAddr, evm.EVMBankDenom).Amount.BigInt()
}

func (k Keeper) EthChainID(ctx sdk.Context) *big.Int {
	return appconst.GetEthChainID(ctx.ChainID())
}

// BaseFeeMicronibiPerGas returns the gas base fee in units of the EVM denom. Note
// that this function is currently constant/stateless.
func (k Keeper) BaseFeeMicronibiPerGas(_ sdk.Context) *big.Int {
	// TODO: (someday maybe):  Consider making base fee dynamic based on
	// congestion in the previous block.
	return evm.BASE_FEE_MICRONIBI
}

// BaseFeeWeiPerGas is the same as BaseFeeMicronibiPerGas, except its in units of
// wei per gas.
func (k Keeper) BaseFeeWeiPerGas(_ sdk.Context) *big.Int {
	return evm.NativeToWei(k.BaseFeeMicronibiPerGas(sdk.Context{}))
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", evm.ModuleName)
}

// IsSimulation checks if the context is a simulation context.
func IsSimulation(ctx sdk.Context) bool {
	if val := ctx.Value(SimulationContextKey); val != nil {
		if simulation, ok := val.(bool); ok && simulation {
			return true
		}
	}
	return false
}

// IsDeliverTx checks if we're in DeliverTx, NOT in CheckTx, ReCheckTx, or simulation
func IsDeliverTx(ctx sdk.Context) bool {
	return !ctx.IsCheckTx() && !ctx.IsReCheckTx() && !IsSimulation(ctx)
}

func (k *Keeper) ImportGenesisAccount(ctx sdk.Context, account evm.GenesisAccount) (err error) {
	address := gethcommon.HexToAddress(account.Address)
	accAddress := sdk.AccAddress(address.Bytes())
	// check that the EVM balance the matches the account balance
	acc := k.accountKeeper.GetAccount(ctx, accAddress)
	if acc == nil {
		err = fmt.Errorf("account not found for address %s", account.Address)
		return
	}

	ethAcct, ok := acc.(eth.EthAccountI)
	if !ok {
		err = fmt.Errorf("account %s must be an EthAccount interface, got %T",
			account.Address, acc,
		)
		return
	}
	code := gethcommon.Hex2Bytes(account.Code)
	codeHash := crypto.Keccak256Hash(code)

	// we ignore the empty Code hash checking, see ethermint PR#1234
	if len(account.Code) != 0 && !bytes.Equal(ethAcct.GetCodeHash().Bytes(), codeHash.Bytes()) {
		err = fmt.Errorf(
			`the evm state code doesn't match with the codehash: account: "%s" , evm state codehash: "%v", ethAccount codehash: "%v", evm state code: "%s"`,
			account.Address, codeHash.Hex(), ethAcct.GetCodeHash().Hex(), account.Code)
		return
	}
	k.SetCode(ctx, codeHash.Bytes(), code)

	for _, storage := range account.Storage {
		k.SetState(ctx, address, gethcommon.HexToHash(storage.Key), gethcommon.HexToHash(storage.Value).Bytes())
	}

	return nil
}
