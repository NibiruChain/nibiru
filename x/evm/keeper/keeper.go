// Copyright (c) 2023-2024 Nibi, Inc.
package keeper

import (
	"math/big"

	"github.com/ethereum/go-ethereum/core"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	gethparams "github.com/ethereum/go-ethereum/params"

	sdkerrors "cosmossdk.io/errors"
	"cosmossdk.io/math"
	"github.com/cometbft/cometbft/libs/log"

	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/NibiruChain/nibiru/eth"
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
	stakingKeeper evm.StakingKeeper

	// Integer for the Ethereum EIP155 Chain ID
	eip155ChainIDInt *big.Int
	hooks            evm.EvmHooks                                  //nolint:unused
	precompiles      map[gethcommon.Address]vm.PrecompiledContract //nolint:unused
	// tracer: Configures the output type for a geth `vm.EVMLogger`. Tracer types
	// include "access_list", "json", "struct", and "markdown". If any other
	// value is used, a no operation tracer is set.
	tracer string
}

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey, transientKey storetypes.StoreKey,
	authority sdk.AccAddress,
	accKeeper evm.AccountKeeper,
	bankKeeper evm.BankKeeper,
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
		accountKeeper: accKeeper,
		bankKeeper:    bankKeeper,
		stakingKeeper: stakingKeeper,
		tracer:        tracer,
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

// SetEvmChainID sets the chain id to the local variable in the keeper
func (k *Keeper) SetEvmChainID(ctx sdk.Context) {
	newEthChainID, err := eth.ParseEthChainID(ctx.ChainID())
	if err != nil {
		panic(err)
	}

	ethChainId := k.eip155ChainIDInt
	if ethChainId != nil && ethChainId.Cmp(newEthChainID) != 0 {
		panic("chain id already set")
	}

	k.eip155ChainIDInt = newEthChainID
}

func (k Keeper) EthChainID(ctx sdk.Context) *big.Int {
	ethChainID, err := eth.ParseEthChainID(ctx.ChainID())
	if err != nil {
		panic(err)
	}
	return ethChainID
}

// AddToBlockGasUsed accumulate gas used by each eth msgs included in current
// block tx.
func (k Keeper) AddToBlockGasUsed(
	ctx sdk.Context, gasUsed uint64,
) (uint64, error) {
	result := k.EvmState.BlockGasUsed.GetOr(ctx, 0) + gasUsed
	if result < gasUsed {
		return 0, sdkerrors.Wrap(evm.ErrGasOverflow, "transient gas used")
	}
	k.EvmState.BlockGasUsed.Set(ctx, gasUsed)
	return result, nil
}

// GetMinGasMultiplier returns minimum gas multiplier.
func (k Keeper) GetMinGasMultiplier(ctx sdk.Context) math.LegacyDec {
	return math.LegacyNewDecWithPrec(50, 2) // 50%
}

func (k Keeper) GetBaseFee(
	ctx sdk.Context, ethCfg *gethparams.ChainConfig,
) *big.Int {
	isLondon := evm.IsLondon(ethCfg, ctx.BlockHeight())
	if !isLondon {
		return nil
	}
	return big.NewInt(0)
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

// PostTxProcessing: Called after tx is processed successfully. If it errors,
// the tx will revert.
func (k *Keeper) PostTxProcessing(
	ctx sdk.Context, msg core.Message, receipt *gethcore.Receipt,
) error {
	if k.hooks == nil {
		return nil
	}
	return k.hooks.PostTxProcessing(ctx, msg, receipt)
}
