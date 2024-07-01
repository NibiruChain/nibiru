package erc20

import (
	"bytes"
	"embed"
	"fmt"

	precommon "github.com/NibiruChain/nibiru/precompiles/common"
	"github.com/NibiruChain/nibiru/x/evm"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
)

const (
	// abiPath defines the path to the ERC-20 precompile ABI JSON file.
	abiPath = "erc20_abi.json"

	GasTransfer          = 3_000_000
	GasApprove           = 30_956
	GasIncreaseAllowance = 34_605
	GasDecreaseAllowance = 34_519
	GasName              = 3_421
	GasSymbol            = 3_464
	GasDecimals          = 427
	GasTotalSupply       = 2_477
	GasBalanceOf         = 2_851
	GasAllowance         = 3_246
)

//go:embed erc20_abi.json
var f embed.FS

var _ vm.PrecompiledContract = &Precompile{}

// Precompile defines the precompiled contract for ERC-20.
type Precompile struct {
	precommon.Precompile
	funToken   evm.FunToken
	bankKeeper bankkeeper.Keeper
}

// NewPrecompile creates a new ERC-20 Precompile
func NewPrecompile(
	funToken evm.FunToken,
	bankKeeper bankkeeper.Keeper,
	// authzKeeper authzkeeper.Keeper,
) (*Precompile, error) {
	abiBz, err := f.ReadFile("erc20_abi.json")
	if err != nil {
		return nil, fmt.Errorf("error loading ABI %s", err)
	}

	newAbi, err := abi.JSON(bytes.NewReader(abiBz))
	if err != nil {
		return nil, err
	}

	return &Precompile{
		Precompile: precommon.Precompile{
			ABI: newAbi,
			//AuthzKeeper:          authzKeeper,
			KvGasConfig:          storetypes.GasConfig{},
			TransientKVGasConfig: storetypes.GasConfig{},
		},
		funToken:   funToken,
		bankKeeper: bankKeeper,
	}, nil
}

// Address defines the address of the ERC-20 precompile contract.
func (p Precompile) Address() common.Address {
	return common.HexToAddress(p.funToken.Erc20Addr.String())
}

// RequiredGas calculates the contract gas used for the
func (p Precompile) RequiredGas(input []byte) uint64 {
	// Validate input length
	if len(input) < 4 {
		return 0
	}

	methodID := input[:4]
	method, err := p.MethodById(methodID)
	if err != nil {
		return 0
	}

	switch method.Name {
	// ERC-20 transactions
	case TransferMethod:
		return GasTransfer
	case TransferFromMethod:
		return GasTransfer
	//case auth.ApproveMethod:
	//	return GasApprove
	//case auth.IncreaseAllowanceMethod:
	//	return GasIncreaseAllowance
	//case auth.DecreaseAllowanceMethod:
	//	return GasDecreaseAllowance
	// ERC-20 queries
	case NameMethod:
		return GasName
	case SymbolMethod:
		return GasSymbol
	case DecimalsMethod:
		return GasDecimals
	case TotalSupplyMethod:
		return GasTotalSupply
	case BalanceOfMethod:
		return GasBalanceOf
	//case auth.AllowanceMethod:
	//	return GasAllowance
	default:
		return 0
	}
}

// Run executes the precompiled contract ERC-20 methods defined in the ABI.
func (p Precompile) Run(evm *vm.EVM, contract *vm.Contract, readOnly bool) (bz []byte, err error) {
	ctx, stateDB, method, initialGas, args, err := p.RunSetup(evm, contract, readOnly, p.IsTransaction)
	if err != nil {
		return nil, err
	}

	// This handles any out of gas errors that may occur during the execution of a precompile tx or query.
	// It avoids panics and returns the out of gas error so the EVM can continue gracefully.
	defer precommon.HandleGasError(ctx, contract, initialGas, &err)()

	bz, err = p.HandleMethod(ctx, contract, stateDB, method, args)
	if err != nil {
		return nil, err
	}

	cost := ctx.GasMeter().GasConsumed() - initialGas

	if !contract.UseGas(cost) {
		return nil, vm.ErrOutOfGas
	}

	return bz, nil
}

// IsTransaction checks if the given method name corresponds to a transaction or query.
func (Precompile) IsTransaction(methodName string) bool {
	switch methodName {
	case TransferMethod,
		TransferFromMethod:
		//auth.ApproveMethod,
		//auth.IncreaseAllowanceMethod,
		//auth.DecreaseAllowanceMethod:
		return true
	default:
		return false
	}
}

// HandleMethod handles the execution of each of the ERC-20 methods.
func (p Precompile) HandleMethod(
	ctx sdk.Context,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) (bz []byte, err error) {
	switch method.Name {
	// ERC-20 transactions
	case TransferMethod:
		bz, err = p.Transfer(ctx, contract, stateDB, method, args)
	case TransferFromMethod:
		bz, err = p.TransferFrom(ctx, contract, stateDB, method, args)
	//case auth.ApproveMethod:
	//	bz, err = p.Approve(ctx, contract, stateDB, method, args)
	//case auth.IncreaseAllowanceMethod:
	//	bz, err = p.IncreaseAllowance(ctx, contract, stateDB, method, args)
	//case auth.DecreaseAllowanceMethod:
	//	bz, err = p.DecreaseAllowance(ctx, contract, stateDB, method, args)
	// ERC-20 queries
	case NameMethod:
		bz, err = p.Name(ctx, contract, stateDB, method, args)
	case SymbolMethod:
		bz, err = p.Symbol(ctx, contract, stateDB, method, args)
	case DecimalsMethod:
		bz, err = p.Decimals(ctx, contract, stateDB, method, args)
	case TotalSupplyMethod:
		bz, err = p.TotalSupply(ctx, contract, stateDB, method, args)
	case BalanceOfMethod:
		bz, err = p.BalanceOf(ctx, contract, stateDB, method, args)
		//case auth.AllowanceMethod:
		//	bz, err = p.Allowance(ctx, contract, stateDB, method, args)
		//default:
		//	return nil, fmt.Errorf(precommon.ErrUnknownMethod, method.Name)
	}

	return bz, err
}
