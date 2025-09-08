// Copyright (c) 2023-2024 Nibi, Inc.
package keeper

import (
	"fmt"
	"math/big"

	sdkioerrors "cosmossdk.io/errors"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/cometbft/cometbft/libs/log"

	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
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

func (k *Keeper) GetErc20Balance(ctx sdk.Context, account, contract gethcommon.Address) (balance *big.Int, err error) {
	if contract == (gethcommon.Address{}) {
		return nil, fmt.Errorf("contract address is empty")
	}

	txConfig := k.TxConfig(ctx, gethcommon.Hash{})
	stateDB := k.Bank.StateDB
	if stateDB == nil {
		stateDB = k.NewStateDB(ctx, txConfig)
	}
	defer func() {
		k.Bank.StateDB = nil
	}()
	evmObj := k.NewEVM(ctx, MOCK_GETH_MESSAGE, k.GetEVMConfig(ctx), nil /*tracer*/, stateDB)

	out, err := k.ERC20().BalanceOf(contract, account, ctx, evmObj)
	if err != nil {
		return nil, sdkioerrors.Wrapf(
			err, "failed to get balance of account %s for token %s",
			account.String(), contract,
		)
	}

	return out, nil
}

func (k *Keeper) Erc20Transfer(ctx sdk.Context, contract, sender, receiver gethcommon.Address, amount *big.Int) (err error) {
	if contract == (gethcommon.Address{}) {
		return sdkioerrors.Wrap(sdkerrors.ErrInvalidAddress, "contract address cannot be zero")
	}
	if amount == nil || amount.Sign() < 0 {
		return sdkioerrors.Wrap(sdkerrors.ErrInvalidRequest, "amount must be non-negative")
	}
	input, err := embeds.SmartContract_ERC20MinterWithMetadataUpdates.ABI.Pack(
		"transfer", receiver, amount,
	)
	if err != nil {
		return sdkioerrors.Wrap(err, "failed to pack ABI args for transfer")
	}
	nonce := k.GetAccNonce(ctx, sender)

	unusedBigInt := big.NewInt(0)
	evmMsg := core.Message{
		To:               &contract,
		From:             sender,
		Nonce:            nonce,
		Value:            unusedBigInt, // amount
		GasLimit:         Erc20GasLimitExecute,
		GasPrice:         unusedBigInt,
		GasFeeCap:        unusedBigInt,
		GasTipCap:        unusedBigInt,
		Data:             input,
		AccessList:       gethcore.AccessList{},
		SkipNonceChecks:  false,
		SkipFromEOACheck: false,
	}
	txConfig := k.TxConfig(ctx, gethcommon.Hash{})
	stateDB := k.Bank.StateDB
	if stateDB == nil {
		stateDB = k.NewStateDB(ctx, txConfig)
	}
	defer func() {
		k.Bank.StateDB = nil
	}()

	evmObj := k.NewEVM(ctx, evmMsg, k.GetEVMConfig(ctx), nil /*tracer*/, stateDB)
	_, resp, err := k.ERC20().Transfer(contract, sender, receiver, amount, ctx, evmObj)
	if err != nil {
		return sdkioerrors.Wrap(err, "failed to call ERC20 contract transfer")
	}
	if resp.Failed() {
		return fmt.Errorf("ERC20 transfer failed with VM error: %s", resp.VmError)
	}
	if err := stateDB.Commit(); err != nil {
		return sdkioerrors.Wrap(err, "failed to commit stateDB")
	}

	return nil
}

func (k *Keeper) Erc20Approve(ctx sdk.Context, contract, sender, spender gethcommon.Address, amount *big.Int) (err error) {
	if contract == (gethcommon.Address{}) {
		return sdkioerrors.Wrap(sdkerrors.ErrInvalidAddress, "contract address cannot be zero")
	}
	if amount == nil || amount.Sign() < 0 {
		return sdkioerrors.Wrap(sdkerrors.ErrInvalidRequest, "amount must be non-negative")
	}
	input, err := embeds.SmartContract_ERC20MinterWithMetadataUpdates.ABI.Pack(
		"approve", spender, amount,
	)
	if err != nil {
		return sdkioerrors.Wrap(err, "failed to pack ABI args for approve")
	}
	nonce := k.GetAccNonce(ctx, sender)

	unusedBigInt := big.NewInt(0)
	evmMsg := core.Message{
		To:               &contract,
		From:             sender,
		Nonce:            nonce,
		Value:            unusedBigInt, // amount
		GasLimit:         Erc20GasLimitExecute,
		GasPrice:         unusedBigInt,
		GasFeeCap:        unusedBigInt,
		GasTipCap:        unusedBigInt,
		Data:             input,
		AccessList:       gethcore.AccessList{},
		SkipNonceChecks:  false,
		SkipFromEOACheck: false,
	}
	txConfig := k.TxConfig(ctx, gethcommon.Hash{})
	stateDB := k.Bank.StateDB
	if stateDB == nil {
		stateDB = k.NewStateDB(ctx, txConfig)
	}
	defer func() {
		k.Bank.StateDB = nil
	}()

	evmObj := k.NewEVM(ctx, evmMsg, k.GetEVMConfig(ctx), nil /*tracer*/, stateDB)
	_, resp, err := k.ERC20().Approve(contract, sender, spender, amount, ctx, evmObj)
	if err != nil {
		return sdkioerrors.Wrap(err, "failed to call ERC20 contract approve")
	}
	if resp.Failed() {
		return fmt.Errorf("ERC20 transfer failed with VM error: %s", resp.VmError)
	}
	if err := stateDB.Commit(); err != nil {
		return sdkioerrors.Wrap(err, "failed to commit stateDB")
	}

	return nil
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
		if err != nil && *err != nil && format != "" {
			*err = fmt.Errorf("%s: %w", format, *err)
		}
	}
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

var MOCK_GETH_MESSAGE = core.Message{
	To:               nil,
	From:             evm.EVM_MODULE_ADDRESS,
	Nonce:            0,
	Value:            evm.Big0, // amount
	GasLimit:         0,
	GasPrice:         evm.Big0,
	GasFeeCap:        evm.Big0,
	GasTipCap:        evm.Big0,
	Data:             []byte{},
	AccessList:       gethcore.AccessList{},
	SkipNonceChecks:  false,
	SkipFromEOACheck: false,
}
