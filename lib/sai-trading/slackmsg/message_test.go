package slackmsg

import (
	"reflect"
	"strings"
	"testing"
)

func TestErrorPreservesComparisonsAndUsesStableFieldOrder(t *testing.T) {
	fields := map[string]any{"trade_index": 7, "error": "balance < required", "market": "BTC"}
	want := "ERROR sai.trader trade rejected\nerror=\"balance < required\"\nmarket=BTC\ntrade_index=7"
	for range 20 {
		if got := Error("Trade rejected", fields)["text"]; got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	}
}

func TestSlackBlockEscapingAndSize(t *testing.T) {
	fields := map[string]any{}
	for _, key := range []string{"a", "b", "c", "d", "e", "f", "g", "h"} {
		fields[key] = strings.Repeat("&", 400) + "```"
	}
	payload := Error("Trade failed", fields)
	block := payload["blocks"].([]map[string]interface{})[0]["text"].(map[string]interface{})["text"].(string)
	if len([]rune(block)) > 3000 || strings.Count(block, "```") != 2 {
		t.Fatal("Slack block must fit the size limit and contain one code fence")
	}
}

func TestHealthDoesNotMutateMarketsOrInventMetrics(t *testing.T) {
	pairs := []string{"btc/usdc", "eth/usdc"}
	payload := Health(map[string]any{"market_pairs": pairs, "balance_factory/address/stNIBI": "12"})
	body := payload["text"].(string)
	if !reflect.DeepEqual(pairs, []string{"btc/usdc", "eth/usdc"}) {
		t.Fatal("formatting mutated the caller's markets")
	}
	if !strings.Contains(body, "opened=N/A") || strings.Contains(strings.ToLower(body), "stnibi") {
		t.Fatal("missing metrics should be N/A and stNIBI should be omitted")
	}
}
