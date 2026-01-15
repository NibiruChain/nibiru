package evmtrader

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

func parseWrappedIndex(s string) (uint64, error) {
	start := strings.LastIndex(s, "(")
	end := strings.LastIndex(s, ")")

	if start == -1 || end == -1 || start >= end {
		return 0, fmt.Errorf("invalid wrapped index format: %s", s)
	}

	numStr := s[start+1 : end]
	num, err := strconv.ParseUint(numStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse number in '%s': %w", s, err)
	}

	return num, nil
}

func parseMarketIndex(s string) (uint64, error) {
	idx, err := parseWrappedIndex(s)
	if err != nil {
		return 0, fmt.Errorf("parse market index '%s': expected MarketIndex(N), got: %w", s, err)
	}
	return idx, nil
}

func parseTokenIndex(s string) (uint64, error) {
	idx, err := parseWrappedIndex(s)
	if err != nil {
		return 0, fmt.Errorf("parse token index '%s': expected TokenIndex(N), got: %w", s, err)
	}
	return idx, nil
}

func parseUserTradeIndex(s string) (uint64, error) {
	idx, err := parseWrappedIndex(s)
	if err != nil {
		return 0, fmt.Errorf("parse user trade index '%s': expected UserTradeIndex(N), got: %w", s, err)
	}
	return idx, nil
}

func parseIndexWithFallback(s string, expectedPrefix string) (uint64, error) {
	// Try wrapped format first
	idx, err := parseWrappedIndex(s)
	if err == nil {
		return idx, nil
	}

	// Try parsing as plain number
	num, numErr := strconv.ParseUint(s, 10, 64)
	if numErr == nil {
		return num, nil
	}

	// Both failed, return error with both attempts
	return 0, fmt.Errorf("parse %s '%s': expected %s(N) or N, wrapped error: %w, number error: %v",
		expectedPrefix, s, expectedPrefix, err, numErr)
}

func tryUnmarshalIndices(responseBytes []byte) ([]uint64, bool) {
	var directIndices []uint64
	if err := json.Unmarshal(responseBytes, &directIndices); err == nil && len(directIndices) > 0 {
		return directIndices, true
	}

	var stringIndices []string
	if err := json.Unmarshal(responseBytes, &stringIndices); err == nil && len(stringIndices) > 0 {
		indices := make([]uint64, 0, len(stringIndices))
		for _, str := range stringIndices {
			if idx, err := parseIndexWithFallback(str, "TokenIndex"); err == nil {
				indices = append(indices, idx)
			}
		}
		if len(indices) > 0 {
			return indices, true
		}
	}

	var wrapped struct {
		Data []uint64 `json:"data"`
	}
	if err := json.Unmarshal(responseBytes, &wrapped); err == nil && len(wrapped.Data) > 0 {
		return wrapped.Data, true
	}

	return nil, false
}
