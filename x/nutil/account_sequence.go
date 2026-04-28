package nutil

import (
	"strconv"
	"strings"
)

// ParseAccountSequenceMismatch extracts the expected and got sequence numbers
// from SDK errors such as "account sequence mismatch, expected 27, got 26".
func ParseAccountSequenceMismatch(rawLog string) (expected uint64, got uint64, ok bool) {
	if !strings.Contains(rawLog, "account sequence mismatch") {
		return 0, 0, false
	}

	expected, ok = parseNumberAfter(rawLog, "expected ")
	if !ok {
		return 0, 0, false
	}
	got, ok = parseNumberAfter(rawLog, "got ")
	if !ok {
		return 0, 0, false
	}
	return expected, got, true
}

// parseNumberAfter returns the unsigned integer immediately following marker.
// It intentionally stays small and strict because it only parses SDK sequence
// mismatch messages such as "expected 27, got 26".
func parseNumberAfter(s string, marker string) (uint64, bool) {
	start := strings.Index(s, marker)
	if start < 0 {
		return 0, false
	}
	start += len(marker)

	end := start
	for end < len(s) && s[end] >= '0' && s[end] <= '9' {
		end++
	}
	if end == start {
		return 0, false
	}

	num, err := strconv.ParseUint(s[start:end], 10, 64)
	return num, err == nil
}
