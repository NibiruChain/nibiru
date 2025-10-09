package precompile_test

import (
	"fmt"
	"log"
	"math/big"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/v2/x/common/testutil"
	"github.com/NibiruChain/nibiru/v2/x/evm"
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

type BlockT struct {
	Time   time.Time
	Height int64
}

func (block BlockT) String() string {
	return fmt.Sprintf("BlockT{TimeUnix:%d, Height:%d}", block.Time.Unix(), block.Height)
}

func (s *OracleSuite) TestOracle_HappyPath() {
	deps := evmtest.NewTestDeps()
	runQuery := func(deps *evmtest.TestDeps) (
		resp *evm.MsgEthereumTxResponse,
		err error,
	) {
		contractInput, err := embeds.SmartContract_Oracle.ABI.Pack(
			string(precompile.OracleMethod_queryExchangeRate),
			"unibi:uusd",
		)
		s.Require().NoError(err)
		evmObj, _ := deps.NewEVM()
		return deps.EvmKeeper.CallContract(
			evmObj,
			deps.Sender.EthAddr,
			&precompile.PrecompileAddr_Oracle,
			contractInput,
			OracleGasLimitQuery,
			evm.COMMIT_READONLY, /*commit*/
			nil,
		)
	}

	s.T().Log("Query exchange rate")
	{
		// 69 seconds + 420 nanoseconds === 69000 milliseconds for the
		// return value from the UnixMilli() function
		deps.SetCtx(deps.Ctx().WithBlockTime(time.Unix(69, 420)).WithBlockHeight(69))
		deps.App.OracleKeeper.SetPrice(deps.Ctx(), "unibi:uusd", sdk.MustNewDecFromStr("0.067"))

		resp, err := runQuery(&deps)
		s.NoError(err)

		// Check the response
		out, err := embeds.SmartContract_Oracle.ABI.Unpack(
			string(precompile.OracleMethod_queryExchangeRate), resp.Ret,
		)
		s.NoError(err)
		s.Equal(out[0].(*big.Int), big.NewInt(67_000_000_000_000_000))
		s.Equal(fmt.Sprintf("%d", out[1].(uint64)), "69000")
		s.Equal(fmt.Sprintf("%d", out[2].(uint64)), "69")
	}

	getBlock := func(deps evmtest.TestDeps) BlockT {
		return BlockT{
			Time:   deps.Ctx().BlockTime(),
			Height: deps.Ctx().BlockHeight(),
		}
	}
	setBlock := func(deps *evmtest.TestDeps, block BlockT) {
		deps.SetCtx(
			deps.Ctx().
				WithBlockTime(block.Time).
				WithBlockHeight(block.Height),
		)
	}

	s.T().Log("Query from a later time")
	{
		blockBefore := getBlock(deps)
		newBlock := BlockT{
			Time:   deps.Ctx().BlockTime().Add(100 * time.Second),
			Height: deps.Ctx().BlockHeight() + 50,
		}
		setBlock(&deps, newBlock)
		resp, err := runQuery(&deps)
		s.NoError(err)

		// Check the response
		out, err := embeds.SmartContract_Oracle.ABI.Unpack(
			string(precompile.OracleMethod_queryExchangeRate), resp.Ret,
		)
		s.NoError(err)
		// These terms should still be equal because the latest exchange rate
		// has not changed.
		s.Equal(out[0].(*big.Int), big.NewInt(67_000_000_000_000_000))
		s.Equal(fmt.Sprintf("%d", out[1].(uint64)), "69000")
		s.Equal(fmt.Sprintf("%d", out[2].(uint64)), "69")

		setBlock(&deps, blockBefore)
	}

	s.T().Log("test IOracle.chainLinkLatestRoundData")
	{
		blockBefore := getBlock(deps)
		newBlock := BlockT{
			Time:   deps.Ctx().BlockTime().Add(100 * time.Second),
			Height: deps.Ctx().BlockHeight() + 50,
		}
		setBlock(&deps, newBlock)
		log.Printf("DEBUG: blockBefore: %+v\n", blockBefore)
		log.Printf("DEBUG: newBlock: %+v\n", newBlock)

		contractInput, err := embeds.SmartContract_Oracle.ABI.Pack(
			string(precompile.OracleMethod_chainLinkLatestRoundData),
			"unibi:uusd",
		)
		s.Require().NoError(err)
		evmObj, _ := deps.NewEVM()
		resp, err := deps.EvmKeeper.CallContract(
			evmObj,
			deps.Sender.EthAddr,
			&precompile.PrecompileAddr_Oracle,
			contractInput,
			OracleGasLimitQuery,
			evm.COMMIT_READONLY, /*commit*/
			nil,
		)
		s.NoError(err)

		// Check the response
		out, err := embeds.SmartContract_Oracle.ABI.Unpack(
			string(precompile.OracleMethod_chainLinkLatestRoundData), resp.Ret,
		)
		s.NoError(err)
		// roundId : created at block height 69
		s.Equal(out[0].(*big.Int), big.NewInt(69))
		// answer : exchange rate with 18 decimals.
		// In this case, 0.067 = 67 * 10^{15}.
		s.Equal(out[1].(*big.Int), big.NewInt(67_000_000_000_000_000))
		// startedAt, updatedAt : created at block timestamp
		blockBeforeTsBig := new(big.Int).SetInt64(blockBefore.Time.Unix())
		s.Equal(out[2].(*big.Int), blockBeforeTsBig)
		s.Equal(out[3].(*big.Int), blockBeforeTsBig)
		// answeredInRound
		s.Equal(out[4].(*big.Int), big.NewInt(420))
	}
}

type OracleSuite struct {
	testutil.LogRoutingSuite
}

// TestPrecompileSuite: Runs all the tests in the suite.
func TestOracleSuite(t *testing.T) {
	suite.Run(t, new(OracleSuite))
}
