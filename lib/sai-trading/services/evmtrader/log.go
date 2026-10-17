package evmtrader

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/NibiruChain/nibiru/lib/sai-trading/slackmsg"
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

	slackMsg := slackmsg.Error(msg, errorFields)

	// Send to Slack (non-blocking)
	go t.sendSlackNotification(t.cfg.SlackWebhook, slackMsg)
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
	go t.sendSlackNotification(webhookURL, slackmsg.Health(fields))
}
