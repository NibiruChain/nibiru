// slack-preview renders the bot's Slack payloads locally without starting a trader.
package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/NibiruChain/nibiru/lib/sai-trading/slackmsg"
)

//go:embed fixtures/*.json
var fixtures embed.FS

type sample struct {
	Kind    string         `json:"kind"`
	Message string         `json:"message"`
	Fields  map[string]any `json:"fields"`
}

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, out, errOut io.Writer) error {
	flags := flag.NewFlagSet("slack-preview", flag.ContinueOnError)
	flags.SetOutput(errOut)
	selected := flags.String("case", "all", "all, healthy, failed-transactions, insufficient-collateral, or rpc-error")
	format := flags.String("format", "text", "text or json")
	if err := flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %s", strings.Join(flags.Args(), " "))
	}
	if *format != "text" && *format != "json" {
		return fmt.Errorf("unknown format %q: use text or json", *format)
	}

	entries, err := fixtures.ReadDir("fixtures")
	if err != nil {
		return err
	}
	var names []string
	payloads := map[string]map[string]interface{}{}
	for _, entry := range entries {
		name := strings.TrimSuffix(entry.Name(), ".json")
		if *selected != "all" && *selected != name {
			continue
		}
		data, err := fixtures.ReadFile("fixtures/" + entry.Name())
		if err != nil {
			return err
		}
		var fixture sample
		if err := json.Unmarshal(data, &fixture); err != nil {
			return fmt.Errorf("fixture %s: %w", name, err)
		}
		// The live bot supplies []string; JSON decodes arrays as []any.
		if pairs, ok := fixture.Fields["market_pairs"].([]any); ok {
			labels := make([]string, len(pairs))
			for i, pair := range pairs {
				label, ok := pair.(string)
				if !ok {
					return fmt.Errorf("fixture %s: market_pairs must contain strings", name)
				}
				labels[i] = label
			}
			fixture.Fields["market_pairs"] = labels
		}
		switch fixture.Kind {
		case "health":
			payloads[name] = slackmsg.Health(fixture.Fields)
		case "error":
			payloads[name] = slackmsg.Error(fixture.Message, fixture.Fields)
		default:
			return fmt.Errorf("fixture %s: unknown kind %q", name, fixture.Kind)
		}
		names = append(names, name)
	}
	if len(names) == 0 {
		return fmt.Errorf("unknown case %q: see --help for available cases", *selected)
	}
	if *format == "json" {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		enc.SetEscapeHTML(false)
		if *selected != "all" {
			return enc.Encode(payloads[names[0]])
		}
		return enc.Encode(payloads)
	}
	for _, name := range names {
		if _, err := fmt.Fprintf(out, "\n[%s]\n%s\n", name, renderText(payloads[name])); err != nil {
			return err
		}
	}
	return nil
}

var fieldLabel = regexp.MustCompile(`^\*([^*\n]+):\*\n`)

// renderText follows block order, flattening Slack's fields into terminal lines.
func renderText(payload map[string]interface{}) string {
	blocks, ok := payload["blocks"].([]map[string]interface{})
	if !ok || len(blocks) == 0 {
		text, _ := payload["text"].(string)
		return text
	}
	var lines []string
	for _, block := range blocks {
		if block["type"] == "divider" {
			lines = append(lines, "")
			continue
		}
		if text, ok := block["text"].(map[string]interface{}); ok {
			lines = append(lines, terminalText(text))
		}
		if fields, ok := block["fields"].([]map[string]interface{}); ok {
			for _, field := range fields {
				lines = append(lines, terminalText(field))
			}
		}
	}
	return strings.Join(lines, "\n")
}

func terminalText(field map[string]interface{}) string {
	text, _ := field["text"].(string)
	if strings.HasPrefix(text, "```\n") && strings.HasSuffix(text, "\n```") {
		text = strings.TrimSuffix(strings.TrimPrefix(text, "```\n"), "\n```")
		return strings.NewReplacer("&lt;", "<", "&gt;", ">", "&amp;", "&").Replace(text)
	}
	text = fieldLabel.ReplaceAllString(text, "$1: ")
	return strings.NewReplacer("&lt;", "<", "&gt;", ">", "&amp;", "&", "*", "").Replace(text)
}
