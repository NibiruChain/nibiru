package precompile

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	gethabi "github.com/ethereum/go-ethereum/accounts/abi"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/NibiruChain/nibiru/v2/app/keepers"
	"github.com/NibiruChain/nibiru/v2/x/common/asset"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
	oraclekeeper "github.com/NibiruChain/nibiru/v2/x/oracle/keeper"
)

var _ vm.PrecompiledContract = (*precompileOracle)(nil)

// Precompile address for "Oracle.sol", the contract that enables queries for exchange rates
var PrecompileAddr_Oracle = gethcommon.HexToAddress("0x0000000000000000000000000000000000000801")

func (p precompileOracle) Address() gethcommon.Address {
	return PrecompileAddr_Oracle
}

func (p precompileOracle) RequiredGas(input []byte) (gasPrice uint64) {
	return requiredGas(input, p.ABI())
}

func (p precompileOracle) ABI() *gethabi.ABI {
	return embeds.SmartContract_Oracle.ABI
}

const (
	OracleMethod_queryExchangeRate PrecompileMethod = "queryExchangeRate"
)

// Run runs the precompiled contract
func (p precompileOracle) Run(
	evm *vm.EVM, contract *vm.Contract, readonly bool,
) (bz []byte, err error) {
	defer func() {
		err = ErrPrecompileRun(err, p)
	}()
	startResult, err := OnRunStart(evm, contract.Input, p.ABI(), contract.Gas)
	if err != nil {
		return nil, err
	}
	method, args, ctx := startResult.Method, startResult.Args, startResult.CacheCtx

	switch PrecompileMethod(method.Name) {
	case OracleMethod_queryExchangeRate:
		bz, err = p.queryExchangeRate(ctx, method, args)
	default:
		// Note that this code path should be impossible to reach since
		// "[decomposeInput]" parses methods directly from the ABI.
		err = fmt.Errorf("invalid method called with name \"%s\"", method.Name)
		return
	}
	if err != nil {
		return nil, err
	}

	contract.UseGas(startResult.CacheCtx.GasMeter().GasConsumed())
	return bz, err
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
	args []any,
) (bz []byte, err error) {
	pair, err := p.parseQueryExchangeRateArgs(args)
	if err != nil {
		return nil, err
	}
	assetPair, err := asset.TryNewPair(pair)
	if err != nil {
		return nil, err
	}

	price, blockTime, blockHeight, err := p.oracleKeeper.GetDatedExchangeRate(ctx, assetPair)
	if err != nil {
		return nil, err
	}

	return method.Outputs.Pack(price.BigInt(), uint64(blockTime), blockHeight)
}

func (p precompileOracle) parseQueryExchangeRateArgs(args []any) (
	pair string,
	err error,
) {
	if e := assertNumArgs(args, 1); e != nil {
		err = e
		return
	}

	pair, ok := args[0].(string)
	if !ok {
		err = ErrArgTypeValidation("string pair", args[0])
		return
	}

	return pair, nil
}
