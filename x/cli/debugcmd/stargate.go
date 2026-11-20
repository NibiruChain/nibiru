package debugcmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/client"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/codec/types"
)

// StargateMessage is the CosmWasm Stargate message form of a protobuf Any.
type StargateMessage struct {
	TypeURL string `json:"type_url,omitempty"`
	Value   string `json:"value,omitempty"`
}

// BuildStargate converts the proto-JSON of an sdk.Msg (with an "@type" field)
// into the {"stargate": {"type_url", "value"}} JSON used in CosmWasm
// transactions and CW3 proposals.
func BuildStargate(
	clientCtx client.Context, msgJSON []byte,
) ([]byte, error) {
	anyMsg := types.Any{}
	if err := clientCtx.Codec.UnmarshalJSON(msgJSON, &anyMsg); err != nil {
		return nil, err
	}
	return json.Marshal(map[string]any{
		"stargate": StargateMessage{
			TypeURL: anyMsg.TypeUrl,
			Value:   base64.StdEncoding.EncodeToString(anyMsg.Value),
		},
	})
}

// NewStargateCmd builds the Stargate message builder. It is registered as
// `debug stargate` and, hidden, as the legacy top-level `build-stargate-wasm`.
func NewStargateCmd(use string, hidden bool) *cobra.Command {
	return &cobra.Command{
		Use:    use,
		Hidden: hidden,
		Short:  "Build a Stargate message in Base64 for a CosmWasm transaction",
		Long: `Build an sdk.Msg as a protobuf Any, in the Stargate message form
used in CosmWasm transactions and CW3 proposals.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)
			js, err := BuildStargate(clientCtx, []byte(args[0]))
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), string(js))
			return err
		},
	}
}
