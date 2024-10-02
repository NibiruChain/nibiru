package precompile

import (
	"fmt"
	"reflect"

	sdk "github.com/cosmos/cosmos-sdk/types"
	gethabi "github.com/ethereum/go-ethereum/accounts/abi"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	gethparams "github.com/ethereum/go-ethereum/params"

	"github.com/NibiruChain/nibiru/v2/app/keepers"
	"github.com/NibiruChain/nibiru/v2/x/common/asset"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
	"github.com/NibiruChain/nibiru/v2/x/evm/statedb"
	oraclekeeper "github.com/NibiruChain/nibiru/v2/x/oracle/keeper"
)

var _ vm.PrecompiledContract = (*precompileOracle)(nil)

// Precompile address for "Oracle.sol", the contract that enables queries for exchange rates
var PrecompileAddr_Oracle = gethcommon.HexToAddress("0x0000000000000000000000000000000000000801")

func (p precompileOracle) Address() gethcommon.Address {
	return PrecompileAddr_Oracle
}

func (p precompileOracle) RequiredGas(input []byte) (gasPrice uint64) {
	// Since [gethparams.TxGas] is the cost per (Ethereum) transaction that does not create
	// a contract, it's value can be used to derive an appropriate value for the precompile call.
	return gethparams.TxGas
}

const (
	OracleMethod_QueryExchangeRate OracleMethod = "queryExchangeRate"
)

type OracleMethod string

// Run runs the precompiled contract
func (p precompileOracle) Run(
	evm *vm.EVM, contract *vm.Contract, readonly bool,
) (bz []byte, err error) {
	// This is a `defer` pattern to add behavior that runs in the case that the error is
	// non-nil, creating a concise way to add extra information.
	defer func() {
		if err != nil {
			precompileType := reflect.TypeOf(p).Name()
			err = fmt.Errorf("precompile error: failed to run %s: %w", precompileType, err)
		}
	}()

	// 1 | Get context from StateDB
	stateDB, ok := evm.StateDB.(*statedb.StateDB)
	if !ok {
		err = fmt.Errorf("failed to load the sdk.Context from the EVM StateDB")
		return
	}
	ctx := stateDB.GetContext()

	method, args, err := DecomposeInput(embeds.SmartContract_Oracle.ABI, contract.Input)
	if err != nil {
		return nil, err
	}

	switch OracleMethod(method.Name) {
	case OracleMethod_QueryExchangeRate:
		bz, err = p.queryExchangeRate(ctx, method, args, readonly)
	default:
		err = fmt.Errorf("invalid method called with name \"%s\"", method.Name)
		return
	}

	return
}

func PrecompileOracle(keepers keepers.PublicKeepers) vm.PrecompiledContract {
	return precompileOracle{
		oracleKeeper: keepers.OracleKeeper,
	}
}

type precompileOracle struct {
	oracleKeeper oraclekeeper.Keeper
}

func (p precompileOracle) queryExchangeRate(
	ctx sdk.Context,
	method *gethabi.Method,
	args []interface{},
	readOnly bool,
) (bz []byte, err error) {
	pair, err := p.decomposeQueryExchangeRateArgs(args)
	if err != nil {
		return nil, err
	}
	assetPair, err := asset.TryNewPair(pair)
	if err != nil {
		return nil, err
	}

	price, err := p.oracleKeeper.GetExchangeRate(ctx, assetPair)
	if err != nil {
		return nil, err
	}

	return method.Outputs.Pack(price.String())
}

func (p precompileOracle) decomposeQueryExchangeRateArgs(args []any) (
	pair string,
	err error,
) {
	if len(args) != 1 {
		err = fmt.Errorf("expected 3 arguments but got %d", len(args))
		return
	}

	pair, ok := args[0].(string)
	if !ok {
		err = fmt.Errorf("type validation for failed for (address erc20) argument")
		return
	}

	return pair, nil
}
