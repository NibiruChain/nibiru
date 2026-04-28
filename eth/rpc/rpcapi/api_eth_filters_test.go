package rpcapi_test

import (
	"context"
	"math/big"
	"strings"

	gethcommon "github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	gethfilters "github.com/ethereum/go-ethereum/eth/filters"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
)

func (s *BackendSuite) TestFiltersGetLogs() {
	criteria := s.deployContractTransferLogCriteria()

	logs, err := s.cli.EvmRpc.Filters.GetLogs(context.Background(), criteria)
	s.Require().NoError(err)
	s.Require().NotEmpty(logs)
	s.Require().Equal(*s.SuccessfulTxDeployContract().Receipt.ContractAddress, logs[0].Address)
	s.Require().Equal(transferTopic(), logs[0].Topics[0])

	blockHashCriteria := criteria
	blockHashCriteria.BlockHash = s.SuccessfulTxDeployContract().BlockHash
	blockHashCriteria.FromBlock = nil
	blockHashCriteria.ToBlock = nil

	logs, err = s.cli.EvmRpc.Filters.GetLogs(context.Background(), blockHashCriteria)
	s.Require().NoError(err)
	s.Require().NotEmpty(logs)
	s.Require().Equal(s.SuccessfulTxDeployContract().Receipt.TxHash, logs[0].TxHash)
}

func (s *BackendSuite) TestFiltersGetFilterLogsAndUninstall() {
	filterID, err := s.cli.EvmRpc.Filters.NewFilter(s.deployContractTransferLogCriteria())
	s.Require().NoError(err)
	s.Require().NotEmpty(filterID)

	logs, err := s.cli.EvmRpc.Filters.GetFilterLogs(context.Background(), filterID)
	s.Require().NoError(err)
	s.Require().NotEmpty(logs)
	s.Require().Equal(s.SuccessfulTxDeployContract().Receipt.TxHash, logs[0].TxHash)

	changes, err := s.cli.EvmRpc.Filters.GetFilterChanges(filterID)
	s.Require().NoError(err)
	s.Require().IsType([]*gethcore.Log{}, changes)
	s.Require().Empty(changes)

	s.Require().True(s.cli.EvmRpc.Filters.UninstallFilter(filterID))
	s.Require().False(s.cli.EvmRpc.Filters.UninstallFilter(filterID))

	logs, err = s.cli.EvmRpc.Filters.GetFilterLogs(context.Background(), filterID)
	s.Require().ErrorContains(err, "not found")
	s.Require().Empty(logs)
}

func (s *BackendSuite) TestFiltersBlockAndPendingFilterChanges() {
	blockFilterID := s.cli.EvmRpc.Filters.NewBlockFilter()
	s.Require().NotEmpty(blockFilterID)
	s.Require().False(strings.Contains(string(blockFilterID), "error creating block filter"))

	changes, err := s.cli.EvmRpc.Filters.GetFilterChanges(blockFilterID)
	s.Require().NoError(err)
	s.Require().IsType([]gethcommon.Hash{}, changes)
	s.Require().Empty(changes)
	s.Require().True(s.cli.EvmRpc.Filters.UninstallFilter(blockFilterID))

	pendingFilterID := s.cli.EvmRpc.Filters.NewPendingTransactionFilter()
	s.Require().NotEmpty(pendingFilterID)
	s.Require().False(strings.Contains(string(pendingFilterID), "error creating pending tx filter"))

	changes, err = s.cli.EvmRpc.Filters.GetFilterChanges(pendingFilterID)
	s.Require().NoError(err)
	s.Require().IsType([]gethcommon.Hash{}, changes)
	s.Require().Empty(changes)
	s.Require().True(s.cli.EvmRpc.Filters.UninstallFilter(pendingFilterID))
}

func (s *BackendSuite) TestFiltersUnsupportedSubscriptionContexts() {
	_, err := s.cli.EvmRpc.Filters.NewPendingTransactions(context.Background())
	s.Require().ErrorIs(err, gethrpc.ErrNotificationsUnsupported)

	_, err = s.cli.EvmRpc.Filters.NewHeads(context.Background())
	s.Require().ErrorIs(err, gethrpc.ErrNotificationsUnsupported)

	_, err = s.cli.EvmRpc.Filters.Logs(context.Background(), s.deployContractTransferLogCriteria())
	s.Require().ErrorIs(err, gethrpc.ErrNotificationsUnsupported)
}

func (s *BackendSuite) deployContractTransferLogCriteria() gethfilters.FilterCriteria {
	blockNumber := new(big.Int).Set(s.SuccessfulTxDeployContract().BlockNumber)
	return gethfilters.FilterCriteria{
		FromBlock: blockNumber,
		ToBlock:   new(big.Int).Set(blockNumber),
		Addresses: []gethcommon.Address{*s.SuccessfulTxDeployContract().Receipt.ContractAddress},
		Topics: [][]gethcommon.Hash{
			{transferTopic()},
		},
	}
}

func transferTopic() gethcommon.Hash {
	return crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)"))
}
