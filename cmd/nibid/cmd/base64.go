package cmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/spf13/cobra"
)

func GetBuildWasmMsg() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "build-stargate-wasm [message data]",
		Short: "Build wasm Stargate message in Base64",
		Long: `This message is used to build a Cosmos SDK,
	in the format used to be sent as a Stargate message in a CosmWasm transaction.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)

			theMsg := args[0]

			anyMsg := types.Any{}
			err := clientCtx.Codec.UnmarshalJSON([]byte(theMsg), &anyMsg)
			if err != nil {
				return err
			}

			type stargateMessage struct {
				TypeURL string `json:"type_url,omitempty"`
				Value   string `json:"value,omitempty"`
			}

			js, err := json.Marshal(map[string]interface{}{
				"stargate": stargateMessage{
					TypeURL: anyMsg.TypeUrl,
					Value:   base64.StdEncoding.EncodeToString(anyMsg.Value),
				},
			})

			fmt.Println(string(js))

			return nil
		},
	}

	return cmd
}
