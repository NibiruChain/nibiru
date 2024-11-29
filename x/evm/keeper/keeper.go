// Copyright (c) 2023-2024 Nibi, Inc.
package keeper

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/vm"
	gethparams "github.com/ethereum/go-ethereum/params"

	"cosmossdk.io/errors"
	"github.com/cometbft/cometbft/libs/log"

	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/x/common/omap"
	"github.com/NibiruChain/nibiru/v2/x/evm"
)

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

	// precompiles is the set of active precompiled contracts used in the EVM.
	// Precompiles are special, built-in contract interfaces that exist at
	// predefined address and run custom logic outside of what is possible only
	// in Solidity.
	precompiles omap.SortedMap[gethcommon.Address, vm.PrecompiledContract]

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

// AddToBlockGasUsed accumulate gas used by each eth msgs included in current
// block tx.
func (k *Keeper) AddToBlockGasUsed(
	ctx sdk.Context, gasUsed uint64,
) (blockGasUsed uint64, err error) {
	// Either k.EvmState.BlockGasUsed.GetOr() or k.EvmState.BlockGasUsed.Set()
	// also consume gas and could panic.
	defer HandleOutOfGasPanic(&err, "")

	blockGasUsed = k.EvmState.BlockGasUsed.GetOr(ctx, 0) + gasUsed
	if blockGasUsed < gasUsed {
		return 0, errors.Wrap(core.ErrGasUintOverflow, "transient gas used")
	}
	k.EvmState.BlockGasUsed.Set(ctx, blockGasUsed)

	return blockGasUsed, nil
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

// Tracer return a default vm.Tracer based on current keeper state
func (k Keeper) Tracer(
	ctx sdk.Context, msg core.Message, ethCfg *gethparams.ChainConfig,
) vm.EVMLogger {
	return evm.NewTracer(k.tracer, msg, ethCfg, ctx.BlockHeight())
}

// HandleOutOfGasPanic gracefully captures "out of gas" panic and just sets the value to err
func HandleOutOfGasPanic(err *error, format string) func() {
	return func() {
		if r := recover(); r != nil {
			switch r.(type) {
			case sdk.ErrorOutOfGas:
				*err = vm.ErrOutOfGas
			default:
				panic(r)
			}
		}
		if err != nil && format != "" {
			*err = fmt.Errorf("%s: %w", format, *err)
		}
	}
}
