package evmtrader

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/mattn/go-colorable"
)

type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

const (
	colorReset = "\033[0m"
	colorBold  = "\033[1m"
	colorDim   = "\033[2m"

	colorRed     = "\033[31m"
	colorGreen   = "\033[32m"
	colorYellow  = "\033[33m"
	colorBlue    = "\033[34m"
	colorMagenta = "\033[35m"
	colorCyan    = "\033[36m"
	colorWhite   = "\033[37m"
	colorGray    = "\033[90m"
)

func (t *EVMTrader) log(level LogLevel, msg string, kv ...any) {
	fields := map[string]any{
		"level": string(level),
		"msg":   msg,
		"ts":    time.Now().UTC().Format(time.RFC3339),
	}

	for i := 0; i+1 < len(kv); i += 2 {
		k, _ := kv[i].(string)
		fields[k] = kv[i+1]
	}

	t.logColored(level, msg, fields)
}

func (t *EVMTrader) logColored(level LogLevel, msg string, fields map[string]any) {
	out := colorable.NewColorable(os.Stdout)

	var levelColor, levelLabel, msgColor string
	switch level {
	case LogLevelDebug:
		levelColor = colorCyan
		levelLabel = "DEBUG"
		msgColor = colorCyan
	case LogLevelInfo:
		levelColor = colorGreen
		levelLabel = "INFO "
		msgColor = colorWhite
	case LogLevelWarn:
		levelColor = colorYellow
		levelLabel = "WARN "
		msgColor = colorYellow
	case LogLevelError:
		levelColor = colorRed
		levelLabel = "ERROR"
		msgColor = colorRed
	default:
		levelColor = colorReset
		levelLabel = string(level)
		msgColor = colorReset
	}

	// Format timestamp
	ts := fields["ts"].(string)
	tsShort := ts[11:19] // Extract time part (HH:MM:SS)

	// Build the log line with improved formatting
	var buf strings.Builder

	// Timestamp in dim gray
	buf.WriteString(fmt.Sprintf("%s%s[%s]%s ",
		colorDim, colorGray, tsShort, colorReset))

	// Level badge with bold and background-like effect
	buf.WriteString(fmt.Sprintf("%s%s%s%s ",
		colorBold, levelColor, levelLabel, colorReset))

	// Message in prominent color
	buf.WriteString(fmt.Sprintf("%s%s%s%s",
		colorBold, msgColor, msg, colorReset))

	// Add key-value pairs with better formatting
	for k, v := range fields {
		if k != "level" && k != "msg" && k != "ts" {
			// Use different colors for keys and values
			keyColor := colorGray
			valColor := colorWhite

			// Special coloring for important fields
			switch k {
			case "error", "tx_hash", "trade_index", "balance", "required", "fund_this_address":
				keyColor = colorCyan
				valColor = colorWhite
			case "trade_size", "leverage", "market_index", "collateral_index":
				keyColor = colorBlue
				valColor = colorWhite
			case "current_positions", "max", "count":
				keyColor = colorMagenta
				valColor = colorWhite
			}

			buf.WriteString(fmt.Sprintf(" %s%s%s=%s%v%s",
				keyColor, k, colorReset,
				valColor, v, colorReset))
		}
	}
	buf.WriteString("\n")

	fmt.Fprint(out, buf.String())
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
	go sendSlackNotification(t.cfg.SlackWebhook, slackMsg)
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

// sendSlackNotification sends a notification to Slack webhook
func sendSlackNotification(webhookURL string, payload map[string]interface{}) {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return // Silently fail if we can't marshal
	}

	req, err := http.NewRequest("POST", webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return // Silently fail if we can't create request
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return // Silently fail if request fails
	}
	defer resp.Body.Close()
}
