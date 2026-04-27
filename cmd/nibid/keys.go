package main

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/eth"

	"github.com/cosmos/cosmos-sdk/client"
	sdkkeys "github.com/cosmos/cosmos-sdk/client/keys"
	cryptokeyring "github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const flagListNames = "list-names"

// keysCommand keeps the stock Cosmos SDK keys command tree and overrides only
// the `keys list` handler so Nibiru can extend its output locally.
func keysCommand(defaultNodeHome string) *cobra.Command {
	cmd := sdkkeys.Commands(defaultNodeHome)

	for _, subCmd := range cmd.Commands() {
		if subCmd.Name() == "list" {
			subCmd.RunE = runKeysListCmd
			break
		}
	}

	return cmd
}

func runKeysListCmd(cmd *cobra.Command, _ []string) error {
	clientCtx, err := client.GetClientQueryContext(cmd)
	if err != nil {
		return err
	}

	records, err := clientCtx.Keyring.List()
	if err != nil {
		return err
	}

	if len(records) == 0 && clientCtx.OutputFormat == sdkkeys.OutputFormatJSON {
		cmd.Println("No records were found in keyring")
		return nil
	}

	if ok, _ := cmd.Flags().GetBool(flagListNames); !ok {
		return printKeyringRecords(cmd.OutOrStdout(), records, clientCtx.OutputFormat)
	}

	for _, k := range records {
		cmd.Println(k.Name)
	}

	return nil
}

type KeyOutput struct {
	cryptokeyring.KeyOutput
	EvmAddr string `json:"evm_addr,omitempty"`
}

func printKeyringRecords(w io.Writer, records []*cryptokeyring.Record, output string) error {
	kosRaw, err := cryptokeyring.MkAccKeysOutput(records)
	if err != nil {
		return err
	}

	kos := make([]KeyOutput, len(kosRaw))
	for idx, ko := range kosRaw {
		kos[idx] = KeyOutput{KeyOutput: ko}

		addrBech32, err := sdk.AccAddressFromBech32(ko.Address)
		if err != nil || len(addrBech32) != appconst.ADDR_LEN_EOA {
			continue
		}
		kos[idx].EvmAddr = eth.NibiruAddrToEthAddr(addrBech32).Hex()
	}

	switch output {
	case sdkkeys.OutputFormatText, sdkkeys.OutputFormatJSON:
		// TODO https://github.com/cosmos/cosmos-sdk/issues/8046
		out, err := json.Marshal(kos)
		if err != nil {
			return err
		}

		if _, err := fmt.Fprintf(w, "%s", out); err != nil {
			return err
		}
	}

	return nil
}
