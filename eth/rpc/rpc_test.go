// Copyright (c) 2023-2024 Nibi, Inc.
package rpc_test

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	cmt "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/client"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/v2/eth/rpc"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
)

type SuiteRPC struct {
	suite.Suite
}

func TestSuiteRPC(t *testing.T) {
	suite.Run(t, new(SuiteRPC))
}

func (s *SuiteRPC) TestRawTxToEthTx() {
	type TestCase struct {
		tx        cmt.Tx
		clientCtx client.Context
		wantErr   string
	}
	type TestCaseSetupFn = func() TestCase

	for _, tcSetup := range []TestCaseSetupFn{
		func() TestCase {
			_, _, clientCtx := evmtest.NewEthTxMsgAsCmt(s.T())
			txBz := []byte("tx")
			return TestCase{
				tx:        txBz,      // invalid bytes
				clientCtx: clientCtx, // valid clientCtx
				wantErr:   "failed to unmarshal JSON",
			}
		},
		func() TestCase {
			txBz, _, clientCtx := evmtest.NewEthTxMsgAsCmt(s.T())
			return TestCase{
				tx:        txBz,      // valid bytes
				clientCtx: clientCtx, // valid clientCtx
				wantErr:   "",        // happy
			}
		},
	} {
		tc := tcSetup()
		ethTxs, err := rpc.RawTxToEthTx(tc.clientCtx, tc.tx)
		if tc.wantErr != "" {
			s.Require().ErrorContains(err, tc.wantErr, "ethTxs: %s", ethTxs)
			continue
		}
		s.Require().NoError(err, "ethTxs: %s", ethTxs)
	}
}

func (s *SuiteRPC) TestEthHeaderFromTendermint() {
	for _, block := range []*cmt.Block{
		// Some happy path test cases for good measure
		cmt.MakeBlock(1, []cmt.Tx{}, nil, nil),
		cmt.MakeBlock(420, []cmt.Tx{}, nil, nil),
	} {
		ethHeader := rpc.EthHeaderFromTendermint(
			block.Header, gethcore.Bloom{}, sdkmath.NewInt(1).BigInt())
		s.NotNil(ethHeader)
		s.Equal(block.Header.Height, ethHeader.Number.Int64())
	}
}
