// Copyright (c) 2023-2024 Nibi, Inc.
package rpcapi

import (
	"math/big"

	"cosmossdk.io/errors"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/ethereum/go-ethereum/common"
	gethcore "github.com/ethereum/go-ethereum/core/types"

	gogoproto "github.com/cosmos/gogoproto/proto"

	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/evm"
)

// FilterLogs creates a slice of logs matching the given criteria.
// [] -> anything
// [A] -> A in first position of log topics, anything after
// [null, B] -> anything in first position, B in second position
// [A, B] -> A in first position and B in second position
// [[A, B], [A, B]] -> A or B in first position, A or B in second position
func FilterLogs(logs []*gethcore.Log, fromBlock, toBlock *big.Int, addresses []common.Address, topics [][]common.Hash) []*gethcore.Log {
	var ret []*gethcore.Log
Logs:
	for _, log := range logs {
		if fromBlock != nil && fromBlock.Int64() >= 0 && fromBlock.Uint64() > log.BlockNumber {
			continue
		}
		if toBlock != nil && toBlock.Int64() >= 0 && toBlock.Uint64() < log.BlockNumber {
			continue
		}
		if len(addresses) > 0 && !includes(addresses, log.Address) {
			continue
		}
		// If the to filtered topics is greater than the amount of topics in logs, skip.
		if len(topics) > len(log.Topics) {
			continue
		}
		for i, sub := range topics {
			match := len(sub) == 0 // empty rule set == wildcard
			for _, topic := range sub {
				if log.Topics[i] == topic {
					match = true
					break
				}
			}
			if !match {
				continue Logs
			}
		}
		ret = append(ret, log)
	}
	return ret
}

func includes(addresses []common.Address, a common.Address) bool {
	for _, addr := range addresses {
		if addr == a {
			return true
		}
	}

	return false
}

// https://github.com/ethereum/go-ethereum/blob/v1.10.14/eth/filters/filter.go#L321
func bloomFilter(bloom gethcore.Bloom, addresses []common.Address, topics [][]common.Hash) bool {
	if len(addresses) > 0 {
		var included bool
		for _, addr := range addresses {
			if gethcore.BloomLookup(bloom, addr) {
				included = true
				break
			}
		}
		if !included {
			return false
		}
	}

	for _, sub := range topics {
		included := len(sub) == 0 // empty rule set == wildcard
		for _, topic := range sub {
			if gethcore.BloomLookup(bloom, topic) {
				included = true
				break
			}
		}
		if !included {
			return false
		}
	}
	return true
}

// returnHashes is a helper that will return an empty hash array case the given hash array is nil,
// otherwise the given hashes array is returned.
func returnHashes(hashes []common.Hash) []common.Hash {
	if hashes == nil {
		return []common.Hash{}
	}
	return hashes
}

// returnLogs is a helper that will return an empty log array in case the given logs array is nil,
// otherwise the given logs array is returned.
func returnLogs(logs []*gethcore.Log) []*gethcore.Log {
	if logs == nil {
		return []*gethcore.Log{}
	}
	return logs
}

// ParseBloomFromEvents iterates through the slice of events
func ParseBloomFromEvents(events []abci.Event) (bloom gethcore.Bloom, err error) {
	bloomEvent := new(evm.EventBlockBloom)
	bloomEventType := gogoproto.MessageName(bloomEvent)
	for _, event := range events {
		if event.Type != bloomEventType {
			continue
		}
		typedProtoEvent, err := sdk.ParseTypedEvent(event)
		if err != nil {
			return bloom, errors.Wrapf(
				err, "failed to parse event of type %s", bloomEventType)
		}
		bloomEvent, ok := (typedProtoEvent).(*evm.EventBlockBloom)
		if !ok {
			return bloom, errors.Wrapf(
				err, "failed to parse event of type %s", bloomEventType)
		}

		return eth.BloomFromHex(bloomEvent.Bloom)
	}
	return bloom, err
}
