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
	return RequiredGas(input, embeds.SmartContract_Oracle.ABI)
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
	res, err := OnRunStart(evm, contract, embeds.SmartContract_Oracle.ABI)
	if err != nil {
		return nil, err
	}
	method, args, ctx := res.Method, res.Args, res.Ctx

	switch PrecompileMethod(method.Name) {
	case OracleMethod_queryExchangeRate:
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
	if e := assertNumArgs(len(args), 1); e != nil {
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
