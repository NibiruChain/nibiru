package evmtrader

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

var logWriter io.Writer = os.Stdout

func gcpSeverity(level LogLevel) string {
	switch level {
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARNING"
	case LogLevelError:
		return "ERROR"
	default:
		return strings.ToUpper(string(level))
	}
}

func buildLogFields(level LogLevel, msg string, kv ...any) map[string]any {
	fields := map[string]any{
		"severity": gcpSeverity(level),
		"message":  msg,
		"ts":       time.Now().UTC().Format(time.RFC3339),
	}
	for i := 0; i+1 < len(kv); i += 2 {
		k, ok := kv[i].(string)
		if !ok || k == "" || k == "severity" || k == "message" || k == "ts" {
			continue
		}
		fields[k] = kv[i+1]
	}
	return fields
}

func (t *EVMTrader) log(level LogLevel, msg string, kv ...any) {
	fields := buildLogFields(level, msg, kv...)
	enc := json.NewEncoder(logWriter)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(fields); err != nil {
		fmt.Fprintf(os.Stderr, "log encode error: %v\n", err)
	}
}

// logInfo is a convenience method that logs at info level (backward compatibility)
func (t *EVMTrader) logInfo(msg string, kv ...any) {
	t.log(LogLevelInfo, msg, kv...)
}

// logDebug logs a debug message
func (t *EVMTrader) logDebug(msg string, kv ...any) {
	t.log(LogLevelDebug, msg, kv...)
}

// logWarn logs a warning message
func (t *EVMTrader) logWarn(msg string, kv ...any) {
	t.log(LogLevelWarn, msg, kv...)
}

// logError logs an error and optionally sends it to Slack webhook
func (t *EVMTrader) logError(msg string, kv ...any) {
	t.log(LogLevelError, msg, kv...)

	// Check if Slack webhook is configured
	if t.cfg.SlackWebhook == "" {
		return
	}

	// Build error message for Slack
	errorFields := map[string]any{}
	for i := 0; i+1 < len(kv); i += 2 {
		k, _ := kv[i].(string)
		errorFields[k] = kv[i+1]
	}

	// Apply error filters if configured
	if t.cfg.SlackErrorFilters != nil {
		// Check exclude list first - if any exclude keyword matches, skip notification
		if len(t.cfg.SlackErrorFilters.Exclude) > 0 {
			for _, keyword := range t.cfg.SlackErrorFilters.Exclude {
				// Check message
				if strings.Contains(strings.ToLower(msg), strings.ToLower(keyword)) {
					return
				}
				// Check error fields
				for _, v := range errorFields {
					vStr := fmt.Sprintf("%v", v)
					if strings.Contains(strings.ToLower(vStr), strings.ToLower(keyword)) {
						return
					}
				}
			}
		}

		// Check include list - if not empty, only send if at least one keyword matches
		if len(t.cfg.SlackErrorFilters.Include) > 0 {
			matched := false

			// Check if message contains any include keyword
			for _, keyword := range t.cfg.SlackErrorFilters.Include {
				if strings.Contains(strings.ToLower(msg), strings.ToLower(keyword)) {
					matched = true
					break
				}
			}

			// If message didn't match, check error fields
			if !matched {
				for _, v := range errorFields {
					vStr := fmt.Sprintf("%v", v)
					for _, keyword := range t.cfg.SlackErrorFilters.Include {
						if strings.Contains(strings.ToLower(vStr), strings.ToLower(keyword)) {
							matched = true
							break
						}
					}
					if matched {
						break
					}
				}
			}

			// If no include keywords matched, don't send to Slack
			if !matched {
				return
			}
		}
	}

	// Format Slack message
	slackMsg := map[string]interface{}{
		"text": fmt.Sprintf("🚨 Auto-Trader Error: %s", msg),
		"blocks": []map[string]interface{}{
			{
				"type": "section",
				"text": map[string]interface{}{
					"type": "mrkdwn",
					"text": fmt.Sprintf("*%s*\n\n*Details:*", msg),
				},
			},
			{
				"type":   "section",
				"fields": buildSlackFields(errorFields),
			},
		},
	}

	// Send to Slack (non-blocking)
	go t.sendSlackNotification(t.cfg.SlackWebhook, slackMsg)
}

// buildSlackFields converts error fields to Slack field format
func buildSlackFields(fields map[string]any) []map[string]interface{} {
	slackFields := []map[string]interface{}{}
	for k, v := range fields {
		slackFields = append(slackFields, map[string]interface{}{
			"type": "mrkdwn",
			"text": fmt.Sprintf("*%s:*\n%s", k, fmt.Sprintf("%v", v)),
		})
		if len(slackFields) >= 10 { // Slack has a limit on fields
			break
		}
	}
	return slackFields
}

// sendSlackNotification posts a payload to the Slack webhook.
func (t *EVMTrader) sendSlackNotification(webhookURL string, payload map[string]interface{}) {
	if webhookURL == "" {
		return
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		t.logWarn("Slack notification dropped: failed to marshal payload", "error", err.Error())
		return
	}

	req, err := http.NewRequest("POST", webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		t.logWarn("Slack notification dropped: failed to build request", "error", err.Error())
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.logWarn("Slack notification failed: request error", "error", err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		t.logWarn("Slack notification rejected by webhook",
			"status", resp.StatusCode,
			"response", strings.TrimSpace(string(body)),
		)
	}
}

// sendSlackHealthNotification sends a health-check summary to Slack (non-blocking).
func (t *EVMTrader) sendSlackHealthNotification(webhookURL string, fields map[string]any) {
	if webhookURL == "" {
		return
	}
	getInt := func(key string) int {
		v, ok := fields[key]
		if !ok || v == nil {
			return 0
		}
		switch vv := v.(type) {
		case int:
			return vv
		case int64:
			return int(vv)
		case uint64:
			return int(vv)
		case float64:
			return int(vv)
		case string:
			i, _ := strconv.Atoi(vv)
			return i
		default:
			return 0
		}
	}
	getStr := func(key string) string {
		v, ok := fields[key]
		if !ok || v == nil {
			return ""
		}
		return fmt.Sprintf("%v", v)
	}

	opened := getInt("positions_opened_24h")
	closed := getInt("positions_closed_24h")
	failed := getInt("failed_txs_24h")
	openNow := getInt("open_positions")
	failedReasonsBlock := getStr("failed_reason_types")
	if failedReasonsBlock == "" {
		failedReasonsBlock = getStr("failed_reason")
	}
	volume := getStr("volume_generated")
	pnl := getStr("realized_pnl")

	// Markets: prefer precomputed market_pairs; else fall back to market_indices.
	var marketLines []string
	if mp, ok := fields["market_pairs"]; ok {
		switch vv := mp.(type) {
		case []string:
			marketLines = vv
		case string:
			marketLines = []string{vv}
		default:
			marketLines = []string{fmt.Sprintf("%v", vv)}
		}
	} else {
		marketLines = []string{getStr("market_indices")}
	}

	// Balances: list any fields named balance_<denom>
	balanceLines := []string{}
	for k, v := range fields {
		if !strings.HasPrefix(k, "balance_") {
			continue
		}
		denom := strings.TrimPrefix(k, "balance_")
		amount := fmt.Sprintf("%v", v)
		// Try to keep symbol-like suffix (e.g. .../stnibi -> stnibi).
		if idx := strings.LastIndex(denom, "/"); idx >= 0 && idx+1 < len(denom) {
			denom = denom[idx+1:]
		}
		balanceLines = append(balanceLines, fmt.Sprintf("%s %s", amount, denom))
	}

	if len(balanceLines) == 0 {
		balanceLines = []string{"(no balances)"}
	}

	blocks, fallbackText := buildStatusReportBlocks(
		opened,
		closed,
		openNow,
		volume,
		pnl,
		failed,
		failedReasonsBlock,
		marketLines,
		balanceLines,
	)

	slackMsg := map[string]interface{}{
		"text":   fallbackText,
		"blocks": blocks,
	}

	go t.sendSlackNotification(webhookURL, slackMsg)
}

func mrkdwnField(label, value string) map[string]interface{} {
	return map[string]interface{}{
		"type": "mrkdwn",
		"text": fmt.Sprintf("*%s:*\n%s", label, value),
	}
}

func plainFields(items []string) []map[string]interface{} {
	fields := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		if len(fields) >= 10 {
			break
		}
		fields = append(fields, map[string]interface{}{
			"type": "mrkdwn",
			"text": item,
		})
	}
	return fields
}

func buildStatusReportBlocks(
	positionsOpened24h int,
	positionsClosed24h int,
	openPositionsNow int,
	volumeGenerated string,
	realizedPnl string,
	failedTxs int,
	failedReasonsBlock string,
	marketLines []string,
	balanceLines []string,
) ([]map[string]interface{}, string) {
	blocks := []map[string]interface{}{
		{
			"type": "section",
			"text": map[string]interface{}{
				"type": "mrkdwn",
				"text": ":robot_face: *EVM Trading Bot — Status Report*\n:white_check_mark: Status: Up and Running",
			},
		},
		{"type": "divider"},
		{
			"type": "section",
			"fields": []map[string]interface{}{
				mrkdwnField("Positions opened (24h)", strconv.Itoa(positionsOpened24h)),
				mrkdwnField("Positions closed (24h)", strconv.Itoa(positionsClosed24h)),
				mrkdwnField("Open positions (now)", strconv.Itoa(openPositionsNow)),
				mrkdwnField("Volume generated", volumeGenerated),
				mrkdwnField("Realized PnL", realizedPnl),
				mrkdwnField("Failed transactions", strconv.Itoa(failedTxs)),
			},
		},
	}

	if failedTxs > 0 {
		blocks = append(blocks, map[string]interface{}{
			"type": "section",
			"text": map[string]interface{}{
				"type": "mrkdwn",
				"text": ":warning: *Failure reasons:*\n" + failedReasonsBlock,
			},
		})
	}

	blocks = append(blocks,
		map[string]interface{}{"type": "divider"},
		map[string]interface{}{
			"type": "section",
			"text": map[string]interface{}{
				"type": "mrkdwn",
				"text": ":chart_with_upwards_trend: *Markets*",
			},
		},
		map[string]interface{}{
			"type":   "section",
			"fields": plainFields(marketLines),
		},
		map[string]interface{}{"type": "divider"},
		map[string]interface{}{
			"type": "section",
			"text": map[string]interface{}{
				"type": "mrkdwn",
				"text": ":moneybag: *Balances*",
			},
		},
		map[string]interface{}{
			"type":   "section",
			"fields": plainFields(balanceLines),
		},
	)

	fallbackText := fmt.Sprintf(
		"EVM Trading Bot Status Report — opened %d, closed %d, open_now %d, failed %d, volume %s, PnL %s",
		positionsOpened24h, positionsClosed24h, openPositionsNow, failedTxs, volumeGenerated, realizedPnl,
	)
	return blocks, fallbackText
}
