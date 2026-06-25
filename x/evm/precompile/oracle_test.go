package precompile_test

import (
	"encoding/json"
	"math/big"
	"strings"
	"testing"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasm "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/x/evm/precompile"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/oracle"
)

const oraclePrecompileGasLimit uint64 = 300_000

func TestOraclePrecompileFailToPackABI(t *testing.T) {
	for _, tc := range []struct {
		name       string
		methodName string
		callArgs   []any
		wantError  string
	}{
		{
			name:       "wrong amount of call args",
			methodName: string(precompile.OracleMethod_queryExchangeRate),
			callArgs:   []any{"nonsense", "args here", "to see if", "precompile is", "called"},
			wantError:  "argument count mismatch: got 5 for 1",
		},
		{
			name:       "wrong type for pair",
			methodName: string(precompile.OracleMethod_queryExchangeRate),
			callArgs:   []any{gethcommon.HexToAddress("0x7D4B7B8CA7E1a24928Bb96D59249c7a5bd1DfBe6")},
			wantError:  "abi: cannot use array as type string as argument",
		},
		{
			name:       "invalid method name",
			methodName: "foo",
			callArgs:   []any{"ubtc:uusdc"},
			wantError:  "method 'foo' not found",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input, err := embeds.SmartContract_Oracle.ABI.Pack(tc.methodName, tc.callArgs...)
			require.ErrorContains(t, err, tc.wantError)
			require.Nil(t, input)
		})
	}
}

func TestOraclePrecompileQueriesAdapter(t *testing.T) {
	deps := evmtest.NewTestDeps()
	evmObj, _ := deps.NewEVM()
	adapterAddr := instantiateXOracleAdapterFixtureForPrecompile(t, &deps)
	setXOracleWasmPluginForPrecompile(t, &deps, adapterAddr)

	t.Run("queryExchangeRate", func(t *testing.T) {
		input, err := embeds.SmartContract_Oracle.ABI.Pack(
			string(precompile.OracleMethod_queryExchangeRate),
			"unibi:uusd",
		)
		require.NoError(t, err)

		resp, err := deps.EvmKeeper.CallContract(
			evmObj,
			deps.Sender.EthAddr,
			&precompile.PrecompileAddr_Oracle,
			input,
			oraclePrecompileGasLimit,
			evm.COMMIT_READONLY,
			nil,
		)
		require.NoError(t, err)

		vals, err := embeds.SmartContract_Oracle.ABI.Unpack(
			string(precompile.OracleMethod_queryExchangeRate),
			resp.Ret,
		)
		require.NoError(t, err)
		require.Equal(t, "138000000000000000000", vals[0].(*big.Int).String())
		require.EqualValues(t, deps.Ctx().BlockTime().Unix()*1000, vals[1].(uint64))
		require.EqualValues(t, deps.Ctx().BlockHeight(), vals[2].(uint64))
	})

	t.Run("chainLinkLatestRoundData", func(t *testing.T) {
		input, err := embeds.SmartContract_Oracle.ABI.Pack(
			string(precompile.OracleMethod_chainLinkLatestRoundData),
			"ubtc:uusd",
		)
		require.NoError(t, err)

		resp, err := deps.EvmKeeper.CallContract(
			evmObj,
			deps.Sender.EthAddr,
			&precompile.PrecompileAddr_Oracle,
			input,
			oraclePrecompileGasLimit,
			evm.COMMIT_READONLY,
			nil,
		)
		require.NoError(t, err)

		vals, err := embeds.SmartContract_Oracle.ABI.Unpack(
			string(precompile.OracleMethod_chainLinkLatestRoundData),
			resp.Ret,
		)
		require.NoError(t, err)
		require.EqualValues(t, deps.Ctx().BlockHeight(), vals[0].(*big.Int).Uint64())
		require.Equal(t, "420000000000000000000", vals[1].(*big.Int).String())
		require.EqualValues(t, deps.Ctx().BlockTime().Unix(), vals[2].(*big.Int).Int64())
		require.EqualValues(t, deps.Ctx().BlockTime().Unix(), vals[3].(*big.Int).Int64())
		require.Equal(t, "420", vals[4].(*big.Int).String())
	})
}

func TestOraclePrecompileSupportsFeeHandlerQuote(t *testing.T) {
	deps := evmtest.NewTestDeps()
	adapterAddr := instantiateXOracleAdapterFixtureForPrecompile(t, &deps)
	setXOracleWasmPluginForPrecompile(t, &deps, adapterAddr)

	require.NoError(t, testapp.FundAccount(
		deps.App.BankKeeper,
		deps.Ctx(),
		deps.Sender.NibiruAddr,
		sdk.NewCoins(sdk.NewCoin(evm.EVMBankDenom, sdk.NewInt(1_000_000_000_000))),
	))

	feeHandlerDeploy, err := evmtest.DeployContract(
		&deps,
		embeds.SmartContract_TestOracleAsLZNativeFeeHandler,
		precompile.PrecompileAddr_Oracle,
	)
	require.NoError(t, err)

	evmObj, _ := deps.NewEVM()
	input, err := embeds.SmartContract_TestOracleAsLZNativeFeeHandler.ABI.Pack(
		"quoteNativeFee",
	)
	require.NoError(t, err)

	resp, err := deps.EvmKeeper.CallContract(
		evmObj,
		deps.Sender.EthAddr,
		&feeHandlerDeploy.ContractAddr,
		input,
		oraclePrecompileGasLimit,
		evm.COMMIT_READONLY,
		nil,
	)
	require.NoError(t, err)

	vals, err := embeds.SmartContract_TestOracleAsLZNativeFeeHandler.ABI.Unpack(
		"quoteNativeFee",
		resp.Ret,
	)
	require.NoError(t, err)
	require.Equal(t, "7246376811594202", vals[0].(*big.Int).String())
}

func TestOraclePrecompileErrors(t *testing.T) {
	t.Run("missing adapter address", func(t *testing.T) {
		deps := evmtest.NewTestDeps()
		evmObj, _ := deps.NewEVM()
		input, err := embeds.SmartContract_Oracle.ABI.Pack(
			string(precompile.OracleMethod_queryExchangeRate),
			"unibi:uusd",
		)
		require.NoError(t, err)

		_, err = deps.EvmKeeper.CallContract(
			evmObj,
			deps.Sender.EthAddr,
			&precompile.PrecompileAddr_Oracle,
			input,
			oraclePrecompileGasLimit,
			evm.COMMIT_READONLY,
			nil,
		)
		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), "x-oracle wasm plugin is not configured"))
	})

	t.Run("unsupported pair", func(t *testing.T) {
		deps := evmtest.NewTestDeps()
		evmObj, _ := deps.NewEVM()
		adapterAddr := instantiateXOracleAdapterFixtureForPrecompile(t, &deps)
		setXOracleWasmPluginForPrecompile(t, &deps, adapterAddr)
		input, err := embeds.SmartContract_Oracle.ABI.Pack(
			string(precompile.OracleMethod_queryExchangeRate),
			"uusdt:uusd",
		)
		require.NoError(t, err)

		_, err = deps.EvmKeeper.CallContract(
			evmObj,
			deps.Sender.EthAddr,
			&precompile.PrecompileAddr_Oracle,
			input,
			oraclePrecompileGasLimit,
			evm.COMMIT_READONLY,
			nil,
		)
		require.Error(t, err)
		require.True(t, strings.Contains(err.Error(), "unsupported legacy pair"))
	})
}

func instantiateXOracleAdapterFixtureForPrecompile(
	t *testing.T,
	deps *evmtest.TestDeps,
) sdk.AccAddress {
	t.Helper()

	wasmPermissionedKeeper := wasmkeeper.NewDefaultPermissionKeeper(deps.App.WasmKeeper)
	codeID, _, err := wasmPermissionedKeeper.Create(
		deps.Ctx(),
		deps.Sender.NibiruAddr,
		oracle.XOracleAdapterWasm,
		&wasm.AccessConfig{Permission: wasm.AccessTypeEverybody},
	)
	require.NoError(t, err)

	instantiateMsg, err := json.Marshal(oracle.XOracleAdapterInstantiateMsg{
		Owner: deps.Sender.NibiruAddr.String(),
		Mode:  oracle.XOracleAdapterFixtureMode(),
		LegacyMappings: []oracle.XOracleAdapterLegacyMapping{
			{Pair: "uusdc:uusd", TokenIndex: 1},
			{Pair: "ubtc:uusd", TokenIndex: 3},
			{Pair: "ueth:uusd", TokenIndex: 4},
			{Pair: "uatom:uusd", TokenIndex: 5},
			{Pair: "unibi:uusd", TokenIndex: 49},
		},
	})
	require.NoError(t, err)

	contractAddr, _, err := wasmPermissionedKeeper.Instantiate(
		deps.Ctx(),
		codeID,
		deps.Sender.NibiruAddr,
		deps.Sender.NibiruAddr,
		instantiateMsg,
		"test x-oracle adapter",
		sdk.Coins{},
	)
	require.NoError(t, err)

	return contractAddr
}

func setXOracleWasmPluginForPrecompile(
	t *testing.T,
	deps *evmtest.TestDeps,
	adapterAddr sdk.AccAddress,
) {
	t.Helper()

	params := deps.EvmKeeper.GetParams(deps.Ctx())
	params.WasmPlugins = []evm.WasmPlugin{
		{
			Name: evm.WasmPluginNameXOracle,
			Addr: adapterAddr.String(),
		},
	}
	require.NoError(t, deps.EvmKeeper.SetParams(deps.Ctx(), params))
}
