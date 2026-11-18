// usdc-migration-report turns a saved block_results response into the repayment
// record. It is read-only: no wallet, RPC credentials, or signing code is used.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/NibiruChain/nibiru/v2/app/upgrades"
	abci "github.com/NibiruChain/nibiru/v2/bft/abci/types"
)

func main() {
	input := flag.String("input", "-", "saved block_results JSON, or - for stdin")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "usage: usdc-migration-report [-input block-results.json]")
		os.Exit(2)
	}
	reader := io.Reader(os.Stdin)
	if *input != "-" {
		file, err := os.Open(*input)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		defer file.Close()
		reader = file
	}
	if err := run(reader, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(input io.Reader, output io.Writer) error {
	const maxBytes = 128 << 20
	data, err := io.ReadAll(io.LimitReader(input, maxBytes+1))
	if err != nil {
		return err
	}
	if len(data) > maxBytes {
		return fmt.Errorf("block results exceed 128 MiB")
	}
	type blockEvents struct {
		Height           json.RawMessage `json:"height"`
		BeginBlockEvents []abci.Event    `json:"begin_block_events"`
	}
	var envelope struct {
		Result *blockEvents    `json:"result"`
		Error  json.RawMessage `json:"error"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return err
	}
	if len(envelope.Error) != 0 && !bytes.Equal(bytes.TrimSpace(envelope.Error), []byte("null")) {
		return fmt.Errorf("RPC returned an error: %s", envelope.Error)
	}
	result := envelope.Result
	if result == nil {
		result = new(blockEvents)
		if err := json.Unmarshal(data, result); err != nil {
			return err
		}
	}
	var heightText string
	if err := json.Unmarshal(result.Height, &heightText); err != nil {
		heightText = string(result.Height)
	}
	height, err := strconv.ParseInt(heightText, 10, 64)
	if err != nil || height < 1 {
		return fmt.Errorf("block results have no valid height")
	}
	report, err := upgrades.ReconstructUSDCMigration(result.BeginBlockEvents)
	if err != nil {
		return err
	}
	if report.Height != height {
		return fmt.Errorf("block height %d differs from migration height %d", height, report.Height)
	}
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
