package main

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
)

// These fixtures track the expected outgoing Slack payloads.
func TestPayloadsMatchExpectedFormatter(t *testing.T) {
	want, err := os.ReadFile("testdata/expected-payloads.json")
	if err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	if err := run([]string{"--format", "json"}, &out, &errOut); err != nil {
		t.Fatal(err)
	}
	var expected, actual any
	if err := json.Unmarshal(want, &expected); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(out.Bytes(), &actual); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("payloads differ from expected formatting:\n%s", out.String())
	}
}

func TestIndividualCases(t *testing.T) {
	for _, name := range []string{"healthy", "failed-transactions", "insufficient-collateral", "rpc-error"} {
		t.Run(name, func(t *testing.T) {
			var out, errOut bytes.Buffer
			if err := run([]string{"--case", name, "--format", "json"}, &out, &errOut); err != nil {
				t.Fatal(err)
			}
			var payload map[string]any
			if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
				t.Fatal(err)
			}
			if payload["text"] == nil || payload["blocks"] == nil || payload[name] != nil {
				t.Fatalf("expected a direct Slack payload, got %s", out.String())
			}
			out.Reset()
			if err := run([]string{"--case", name}, &out, &errOut); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out.String(), "["+name+"]") || strings.Contains(out.String(), "*Details:*") {
				t.Fatalf("unexpected terminal display: %s", out.String())
			}
		})
	}
}

func TestTerminalContent(t *testing.T) {
	var out, errOut bytes.Buffer
	if err := run(nil, &out, &errOut); err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{
		"INFO sai.trader status markets=[BTC, ETH]",
		"24h opened=31 closed=30 failed=0 volume_usd=9.70 pnl_usd=0.000165",
		`failures="Insufficient collateral: 1; RPC unavailable: 1"`,
		"WARN sai.trader insufficient collateral",
		"collateral_index=3 available=0 required=93663",
		`http_status=502 reason="Bad Gateway"`,
	} {
		if !strings.Contains(out.String(), text) {
			t.Errorf("terminal output missing %q", text)
		}
	}
	for _, unwanted := range []string{"stnibi", "<html>", "nginx", ":robot_face:", "```", "Details:", "Up and Running", "(no balances)"} {
		if strings.Contains(strings.ToLower(out.String()), strings.ToLower(unwanted)) {
			t.Errorf("terminal output should omit %q", unwanted)
		}
	}
}

func TestInvalidArguments(t *testing.T) {
	for _, args := range [][]string{
		{"--case", "missing"}, {"--format", "html"}, {"unexpected"}, {"--unknown"}, {"--case"},
	} {
		var out, errOut bytes.Buffer
		if err := run(args, &out, &errOut); err == nil {
			t.Errorf("accepted invalid arguments %v", args)
		}
		if out.Len() != 0 {
			t.Errorf("invalid arguments produced preview output: %v", args)
		}
	}
}

func TestHelp(t *testing.T) {
	var out, errOut bytes.Buffer
	if err := run([]string{"--help"}, &out, &errOut); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(errOut.String(), "insufficient-collateral") || out.Len() != 0 {
		t.Fatal("help should list the supported cases without preview output")
	}
}
