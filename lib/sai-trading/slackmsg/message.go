// Package slackmsg builds Slack payloads without network or trading dependencies.
package slackmsg

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

var (
	httpError       = regexp.MustCompile(`\b([45][0-9]{2})\s+([^:<\r\n]+)`)
	collateralTotal = regexp.MustCompile(`\btotal=([0-9]+)\b`)
	htmlBody        = regexp.MustCompile(`(?i)<(?:!doctype|html|head|body)\b`)
)

// Error builds a compact trading error notification. Full errors stay in stdout logs.
func Error(msg string, fields map[string]any) map[string]interface{} {
	level := "ERROR"
	summary := strings.TrimSpace(msg)
	if strings.EqualFold(summary, "Insufficient collateral") {
		level = "WARN"
		summary = "insufficient collateral"
	} else if summary == "Failed to get block number" {
		summary = "block query failed"
	} else if summary != "" {
		runes := []rune(summary)
		runes[0] = unicode.ToLower(runes[0])
		summary = string(runes)
	}
	lines := []string{level + " sai.trader " + compact(summary)}
	if level == "WARN" && strings.EqualFold(msg, "Insufficient collateral") {
		var details []string
		if v, ok := fields["collateral_index"]; ok {
			details = append(details, "collateral_index="+value(v))
		}
		if total := collateralTotal.FindStringSubmatch(fmt.Sprint(fields["error"])); len(total) > 0 {
			details = append(details, "available="+total[1])
		}
		if v, ok := fields["required"]; ok {
			details = append(details, "required="+value(v))
		}
		if len(details) > 0 {
			lines = append(lines, strings.Join(details, " "))
		}
		if address := strings.Fields(text(fields, "fund_this_address")); len(address) > 0 {
			lines = append(lines, "fund_address="+value(address[0]))
		}
		// Keep unexpected fields visible without repeating the full collateral error.
		lines = appendDetails(lines, fields, "collateral_index", "required", "fund_this_address", "error")
	} else {
		lines = appendDetails(lines, fields)
	}
	return message(strings.Join(lines, "\n"))
}

func appendDetails(lines []string, fields map[string]any, omit ...string) []string {
	skip := map[string]bool{}
	for _, key := range omit {
		skip[key] = true
	}
	keys := make([]string, 0, len(fields))
	for key := range fields {
		if !skip[key] {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	for _, key := range keys {
		if key == "error" {
			detail := fmt.Sprint(fields[key])
			if match := httpError.FindStringSubmatch(detail); len(match) > 0 {
				lines = append(lines, "http_status="+match[1]+" reason="+strconv.Quote(strings.TrimSpace(match[2])))
				continue
			}
		}
		lines = append(lines, compact(key)+"="+value(fields[key]))
	}
	return lines
}

// Health builds a status report with a fixed field order and a 24h metrics window.
func Health(fields map[string]any) map[string]interface{} {
	var markets []string
	switch pairs := fields["market_pairs"].(type) {
	case []string:
		markets = append(markets, pairs...)
	case string:
		markets = []string{pairs}
	}
	for i, pair := range markets {
		base, _, _ := strings.Cut(pair, "/")
		markets[i] = strings.ToUpper(strings.TrimSpace(base))
	}
	header := "INFO sai.trader status"
	if len(markets) > 0 {
		header += " markets=[" + strings.Join(markets, ", ") + "]"
	} else if indices := text(fields, "market_indices"); indices != "" {
		header += " market_indices=" + indices
	}
	lines := []string{
		header,
		fmt.Sprintf("24h opened=%s closed=%s failed=%s volume_usd=%s pnl_usd=%s",
			metric(fields, "positions_opened_24h"), metric(fields, "positions_closed_24h"),
			metric(fields, "failed_txs_24h"), usd(fields, "volume_generated"), usd(fields, "realized_pnl")),
		"open_positions=" + metric(fields, "open_positions"),
	}
	if failed := metric(fields, "failed_txs_24h"); failed != "0" && failed != "N/A" {
		reason := text(fields, "failed_reason_types")
		if reason == "" {
			reason = text(fields, "failed_reason")
		}
		if reason != "" {
			lines = append(lines, "failures="+value(strings.ReplaceAll(reason, "\n", "; ")))
		}
	}
	var balanceKeys []string
	for key := range fields {
		if strings.HasPrefix(key, "balance_") {
			balanceKeys = append(balanceKeys, key)
		}
	}
	sort.Strings(balanceKeys)
	for _, key := range balanceKeys {
		denom := strings.TrimPrefix(key, "balance_")
		parts := strings.Split(denom, "/")
		symbol := parts[len(parts)-1]
		if strings.EqualFold(symbol, "stnibi") {
			continue
		}
		// Without token metadata, an address and zero raw units add no useful status information.
		if strings.HasPrefix(strings.ToLower(symbol), "0x") && fmt.Sprint(fields[key]) == "0" {
			continue
		}
		lines = append(lines, "balance_raw["+denom+"]="+value(fields[key]))
	}
	return message(strings.Join(lines, "\n"))
}

func text(fields map[string]any, key string) string {
	if v := fields[key]; v != nil {
		return fmt.Sprint(v)
	}
	return ""
}

func metric(fields map[string]any, key string) string {
	if v := text(fields, key); v != "" {
		return v
	}
	return "N/A"
}

func usd(fields map[string]any, key string) string {
	return strings.TrimSuffix(metric(fields, key), " USD")
}

func compact(s string) string {
	// HTTP libraries can put full response pages into error strings.
	if match := htmlBody.FindStringIndex(s); match != nil {
		s = s[:match[0]]
	}
	s = strings.Join(strings.Fields(s), " ")
	runes := []rune(s)
	if len(runes) > 400 {
		s = string(runes[:400]) + "..."
	}
	return s
}

func value(v any) string {
	s := compact(fmt.Sprint(v))
	if s == "" || strings.ContainsAny(s, " \t\n\"`=\\") {
		return strconv.Quote(s)
	}
	return s
}

func message(body string) map[string]interface{} {
	// Keep one Slack section below its 3000-character limit and prevent nested fences.
	body = strings.ReplaceAll(body, "```", "'''")
	if runes := []rune(body); len(runes) > 2800 {
		body = string(runes[:2800]) + "\n..."
	}
	escape := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	escaped := escape.Replace(body)
	for len([]rune(escaped)) > 2900 {
		runes := []rune(body)
		body = string(runes[:len(runes)*9/10]) + "\n..."
		escaped = escape.Replace(body)
	}
	return map[string]interface{}{
		"text": body,
		"blocks": []map[string]interface{}{{
			"type": "section",
			"text": map[string]interface{}{"type": "mrkdwn", "text": "```\n" + escaped + "\n```"},
		}},
	}
}
