package precompile

import (
	"encoding/json"
	"fmt"
	"math/big"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gethabi "github.com/ethereum/go-ethereum/accounts/abi"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/NibiruChain/nibiru/v2/app/keepers"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmstate"
	xoracle "github.com/NibiruChain/nibiru/v2/x/oracle"
)

var (
	_ vm.PrecompiledContract = (*precompileOracle)(nil)
	_ vm.DynamicPrecompile   = (*precompileOracle)(nil)
)

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
	OracleMethod_queryExchangeRate        PrecompileMethod = "queryExchangeRate"
	OracleMethod_chainLinkLatestRoundData PrecompileMethod = "chainLinkLatestRoundData"
)

// Run runs the precompiled contract
func (p precompileOracle) Run(
	evmObj *vm.EVM,
	trueCaller gethcommon.Address,
	// Note that we use "trueCaller" here to differentiate between a delegate
	// caller ("parent.CallerAddress" in geth) and "contract.CallerAddress"
	// because these two addresses may differ.
	contract *vm.Contract,
	readonly bool,
	// isDelegatedCall: Flag to add conditional logic specific to delegate calls
	isDelegatedCall bool,
) (bz []byte, err error) {
	bz, _, err = p.DynamicRun(evmObj, trueCaller, contract, readonly, isDelegatedCall)
	return bz, err
}

func (p precompileOracle) DynamicRun(
	evmObj *vm.EVM,
	trueCaller gethcommon.Address,
	// Note that we use "trueCaller" here to differentiate between a delegate
	// caller ("parent.CallerAddress" in geth) and "contract.CallerAddress"
	// because these two addresses may differ.
	contract *vm.Contract,
	readonly bool,
	// isDelegatedCall: Flag to add conditional logic specific to delegate calls
	isDelegatedCall bool,
) (bz []byte, gasCost uint64, err error) {
	defer func() {
		err = ErrPrecompileRun(err, p)
	}()
	startResult, err := OnRunStart(evmObj, contract.Input, p.ABI(), contract.Gas)
	if err != nil {
		return nil, gasCost, err
	}
	defer func() {
		// Recover OOG panics as ErrOutOfGas; other panics become an error.
		var (
			oog  bool  // true if panic was out-of-gas
			perr error // ErrOutOfGas for OOG, or formatted error for unexpected panic
		)
		panicInfo := recover()
		if panicInfo != nil {
			oog, perr = evm.ParseOOGPanic(panicInfo, func(p any) string {
				return fmt.Sprintf("unexpected panic in precompile: %v", p)
			})
		}
		if oog {
			gasCost = startResult.Ctx.GasMeter().GasConsumed()
			err = perr
			return
		} else if perr != nil {
			err = perr
			return
		}
	}()

	if err := assertContractQuery(contract); err != nil {
		return nil, gasCost, err
	}

	method, args, ctx := startResult.Method, startResult.Args, startResult.Ctx

	switch PrecompileMethod(method.Name) {
	case OracleMethod_queryExchangeRate:
		bz, err = p.queryExchangeRate(ctx, method, args)
	// For "@chainlink/contracts/src/v0.8/shared/interfaces/AggregatorV3Interface.sol"
	case OracleMethod_chainLinkLatestRoundData:
		bz, err = p.chainLinkLatestRoundData(ctx, method, args)

	default:
		// Note that this code path should be impossible to reach since
		// "[decomposeInput]" parses methods directly from the ABI.
		err = fmt.Errorf("invalid method called with name \"%s\"", method.Name)
		return
	}
	gasCost = startResult.Ctx.GasMeter().GasConsumed()
	if err != nil {
		if len(bz) == 0 {
			bz = revertBzForErr(err)
		}
		return bz, gasCost, err
	}

	return bz, gasCost, err
}

func PrecompileOracle(keepers keepers.PublicKeepers) NibiruCustomPrecompile {
	return precompileOracle{
		evmKeeper:  keepers.EvmKeeper,
		wasmKeeper: keepers.WasmKeeper,
	}
}

type precompileOracle struct {
	evmKeeper  *evmstate.Keeper
	wasmKeeper wasmkeeper.Keeper
}

// Implements "IOracle.queryExchangeRate"
//
//	```solidity
//	function queryExchangeRate(
//	    string memory pair
//	)
//	    external
//	    view
//	    returns (uint256 price, uint64 blockTimeMs, uint64 blockHeight);
//	```
//
// blockTimeMs is the adapter's underlying Sai oracle price update time in
// milliseconds, preserving legacy freshness semantics despite the field name.
// blockHeight is the current Nibiru block height for the query.
func (p precompileOracle) queryExchangeRate(
	ctx sdk.Context,
	method *gethabi.Method,
	args []any,
) (bz []byte, err error) {
	pair, err := p.parseQueryExchangeRateArgs(args)
	if err != nil {
		return nil, err
	}

	adapterResp, err := p.queryLegacyExchangeRate(ctx, pair)
	if err != nil {
		return nil, err
	}

	price18, err := parseAdapterPrice18(adapterResp.Price18)
	if err != nil {
		return nil, err
	}

	blockTimeMs := adapterBlockTimeMs(adapterResp)
	return method.Outputs.Pack(
		price18,
		blockTimeMs,
		uint64(ctx.BlockHeight()),
	)
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
	if pair == "" {
		err = fmt.Errorf("pair cannot be empty")
		return
	}

	return pair, nil
}

// Implements "IOracle.chainLinkLatestRoundData"
//
//	```solidity
//	interface IOracle {
//	  function chainLinkLatestRoundData(
//	    string memory pair
//	  )
//	      external
//	      view
//	      returns (
//	          uint80 roundId,
//	          int256 answer,
//	          uint256 startedAt,
//	          uint256 updatedAt,
//	          uint80 answeredInRound
//	      );
//	  // ...
//	}
//	```
func (p precompileOracle) chainLinkLatestRoundData(
	ctx sdk.Context,
	method *gethabi.Method,
	args []any,
) (bz []byte, err error) {
	pair, err := p.parseQueryExchangeRateArgs(args)
	if err != nil {
		return nil, err
	}

	adapterResp, err := p.queryLegacyExchangeRate(ctx, pair)
	if err != nil {
		return nil, err
	}

	price18, err := parseAdapterPrice18(adapterResp.Price18)
	if err != nil {
		return nil, err
	}

	roundId := new(big.Int).SetUint64(uint64(ctx.BlockHeight()))
	answer := price18 // 18 decimals
	// Chainlink consumers commonly use updatedAt for staleness checks. Keep
	// both timestamps tied to the Sai oracle price update time reported by the
	// adapter, not this query's block time.
	timestampSeconds := new(big.Int).SetUint64(adapterUpdateTimeSeconds(adapterResp))
	answeredInRound := big.NewInt(420) // for no reason in particular / unused
	return method.Outputs.Pack(
		roundId,
		answer,
		timestampSeconds, // startedAt (seconds)
		timestampSeconds, // updatedAt (seconds)
		answeredInRound,
	)
}

func (p precompileOracle) queryLegacyExchangeRate(
	ctx sdk.Context,
	pair string,
) (xoracle.XOracleAdapterLegacyExchangeRateResp, error) {
	adapterAddr, err := p.evmKeeper.EVMState().WasmPlugins.Get(ctx, evm.WasmPluginNameXOracle)
	if err != nil {
		return xoracle.XOracleAdapterLegacyExchangeRateResp{},
			fmt.Errorf("x-oracle wasm plugin is not configured: %w", err)
	}
	if adapterAddr.Empty() {
		return xoracle.XOracleAdapterLegacyExchangeRateResp{},
			fmt.Errorf("x-oracle wasm plugin address is empty")
	}

	req, err := json.Marshal(xoracle.XOracleAdapterQueryMsg{
		LegacyExchangeRate: &xoracle.XOracleAdapterLegacyExchangeRateQuery{Symbol: pair},
	})
	if err != nil {
		return xoracle.XOracleAdapterLegacyExchangeRateResp{}, err
	}

	respBz, err := p.wasmKeeper.QuerySmart(ctx, adapterAddr, req)
	if err != nil {
		return xoracle.XOracleAdapterLegacyExchangeRateResp{}, err
	}

	var resp xoracle.XOracleAdapterLegacyExchangeRateResp
	if err := json.Unmarshal(respBz, &resp); err != nil {
		return xoracle.XOracleAdapterLegacyExchangeRateResp{}, err
	}
	return resp, nil
}

func parseAdapterPrice18(price18 string) (*big.Int, error) {
	price, ok := new(big.Int).SetString(price18, 10)
	if !ok {
		return nil, fmt.Errorf("invalid adapter price_18 %q", price18)
	}
	if price.Sign() < 0 {
		return nil, fmt.Errorf("adapter price_18 must be non-negative: %s", price18)
	}
	return price, nil
}

func adapterUpdateTimeSeconds(r xoracle.XOracleAdapterLegacyExchangeRateResp) uint64 {
	if r.UpdateTimeSeconds != nil {
		return *r.UpdateTimeSeconds
	}
	return 0
}

func adapterBlockTimeMs(r xoracle.XOracleAdapterLegacyExchangeRateResp) uint64 {
	return adapterUpdateTimeSeconds(r) * 1000
}
