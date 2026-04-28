package wasm_cli_test

import (
	"encoding/hex"
	"fmt"
	"testing"

	wasmcli "github.com/CosmWasm/wasmd/x/wasm/client/cli"

	"github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/x/nutil"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil/localnet"
)

const (
	wasmStoreCodeGas           = "auto"
	wasmStoreCodeGasAdjustment = "1.5"
	wasmStoreCodeFees          = "10000000" + appconst.DENOM_UNIBI
	wasmCodeListLimit          = "100000"
)

var _ suite.TearDownAllSuite = (*TestSuite)(nil)

type TestSuite struct {
	suite.Suite

	localnetCLI localnet.CLI
}

func (s *TestSuite) SetupSuite() {
	if err := nutil.EnsureLocalBlockchain(); err != nil {
		s.T().Skipf("skipping localnet-backed wasm CLI tests: %v", err)
	}

	localnetCLI, err := localnet.NewCLI()
	s.Require().NoError(err)
	s.localnetCLI = localnetCLI
}

func (s *TestSuite) TearDownSuite() {
	s.Require().NoError(s.localnetCLI.Close())
	s.T().Log("leaving localnet state in place")
}

func (s *TestSuite) TestWasmHappyPath() {
	beforeCodeInfos := s.queryDeployedContracts()

	codeID, err := s.deployWasmContract("testdata/cw_nameservice.wasm")
	s.Require().NoError(err)
	s.Require().Positive(codeID)

	afterCodeInfos := s.queryDeployedContracts()
	s.Require().GreaterOrEqual(len(afterCodeInfos), len(beforeCodeInfos)+1)
	s.Require().True(
		hasCodeID(afterCodeInfos, codeID),
		"stored code id %d not found in list-code response",
		codeID,
	)
}

func (s *TestSuite) wasmStoreCodeTxOptions() []localnet.TxOption {
	return []localnet.TxOption{
		localnet.WithTxGas(wasmStoreCodeGas),
		localnet.WithTxGasAdjustment(wasmStoreCodeGasAdjustment),
		localnet.WithTxFees(wasmStoreCodeFees),
	}
}

// deployWasmContract deploys a wasm contract located in path.
func (s *TestSuite) deployWasmContract(path string) (uint64, error) {
	args := []string{
		"store",
		path,
	}
	txOptions := s.wasmStoreCodeTxOptions()

	cmd := wasmcli.GetTxCmd()
	s.T().Log(s.localnetCLI.RenderTxCmd(cmd, args, txOptions...))

	resp, err := s.localnetCLI.ExecTxCmd(cmd, args, txOptions...)
	if err != nil {
		return 0, err
	}

	decodedResult, err := hex.DecodeString(resp.Data)
	if err != nil {
		return 0, err
	}

	respData := sdk.TxMsgData{}
	codec := s.localnetCLI.ClientCtx.Codec
	err = codec.Unmarshal(decodedResult, &respData)
	if err != nil {
		return 0, err
	}

	if len(respData.MsgResponses) < 1 {
		return 0, fmt.Errorf("no data found in response")
	}

	var storeCodeResponse types.MsgStoreCodeResponse
	err = codec.Unmarshal(respData.MsgResponses[0].Value, &storeCodeResponse)
	if err != nil {
		return 0, err
	}

	return storeCodeResponse.CodeID, nil
}

// queryDeployedContracts lists the currently uploaded wasm code on localnet.
func (s *TestSuite) queryDeployedContracts() []types.CodeInfoResponse {
	var queryCodeResponse types.QueryCodesResponse
	cmd := wasmcli.GetQueryCmd()
	args := []string{
		"list-code",
		fmt.Sprintf("--limit=%s", wasmCodeListLimit),
	}
	s.T().Log(s.localnetCLI.RenderQueryCmd(cmd, args))

	err := s.localnetCLI.ExecQueryCmd(
		cmd,
		args,
		&queryCodeResponse,
	)
	s.Require().NoError(err)
	return queryCodeResponse.CodeInfos
}

func hasCodeID(codeInfos []types.CodeInfoResponse, codeID uint64) bool {
	for _, codeInfo := range codeInfos {
		if codeInfo.CodeID == codeID {
			return true
		}
	}
	return false
}

func Test(t *testing.T) {
	suite.Run(t, new(TestSuite))
}
