package precompile_test

import (
	"math/big"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
	"github.com/NibiruChain/nibiru/v2/x/evm/precompile"
)

const OracleGasLimitQuery = 100_000

func (s *OracleSuite) TestOracle_FailToPackABI() {
	testcases := []struct {
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
			callArgs:   []any{common.HexToAddress("0x7D4B7B8CA7E1a24928Bb96D59249c7a5bd1DfBe6")},
			wantError:  "abi: cannot use array as type string as argument",
		},
		{
			name:       "invalid method name",
			methodName: "foo",
			callArgs:   []any{"ubtc:uusdc"},
			wantError:  "method 'foo' not found",
		},
	}

	abi := embeds.SmartContract_Oracle.ABI

	for _, tc := range testcases {
		s.Run(tc.name, func() {
			input, err := abi.Pack(tc.methodName, tc.callArgs...)
			s.ErrorContains(err, tc.wantError)
			s.Nil(input)
		})
	}
}

func (s *OracleSuite) TestOracle_HappyPath() {
	deps := evmtest.NewTestDeps()

	s.T().Log("Query exchange rate")
	{
		deps.Ctx = deps.Ctx.WithBlockTime(time.Unix(69, 420)).WithBlockHeight(69)
		deps.App.OracleKeeper.SetPrice(deps.Ctx, "unibi:uusd", sdk.MustNewDecFromStr("0.067"))

		resp, err := deps.EvmKeeper.CallContract(
			deps.Ctx,
			embeds.SmartContract_Oracle.ABI,
			deps.Sender.EthAddr,
			&precompile.PrecompileAddr_Oracle,
			false,
			OracleGasLimitQuery,
			"queryExchangeRate",
			"unibi:uusd",
		)
		s.NoError(err)

		// Check the response
		out, err := embeds.SmartContract_Oracle.ABI.Unpack(string(precompile.OracleMethod_queryExchangeRate), resp.Ret)
		s.NoError(err)

		// Check the response
		s.Equal(out[0].(*big.Int), big.NewInt(67000000000000000))
		s.Equal(out[1].(uint64), uint64(69000))
		s.Equal(out[2].(uint64), uint64(69))
	}
}

type OracleSuite struct {
	suite.Suite
}

// TestPrecompileSuite: Runs all the tests in the suite.
func TestOracleSuite(t *testing.T) {
	suite.Run(t, new(OracleSuite))
}
