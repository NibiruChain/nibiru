package cmd

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

// DecodeBase64Cmd creates a cobra command for base64 decoding.
func DecodeBase64Cmd(defaultNodeHome string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "base64-decode",
		Short: "Decode a base64-encoded protobuf message",
		Long: `Decode a base64-encoded protobuf message from JSON input.
The input should be a JSON object with 'type_url' and 'value' fields.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)

			protoMsg := &anypb.Any{}
			if err := proto.Unmarshal(protoMsgBytes, protoMsg); err != nil {
				return err
			}

			clientCtx.PrintProto(protoMsg)
		},
	}

	cmd.Flags().String(flags.FlagHome, defaultNodeHome, "The application home directory")

	return cmd
}
