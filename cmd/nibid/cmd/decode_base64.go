package cmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	sdkcodec "github.com/cosmos/cosmos-sdk/codec"

	wasmvm "github.com/CosmWasm/wasmvm/types"
)

// YieldStargateMsgs parses the JSON and sends wasmvm.StargateMsg objects to a channel
func YieldStargateMsgs(jsonBz []byte) ([]wasmvm.StargateMsg, error) {
	var data interface{}
	if err := json.Unmarshal(jsonBz, &data); err != nil {
		return nil, err
	}

	var msgs []wasmvm.StargateMsg
	parseStargateMsgs(data, &msgs)
	return msgs, nil
}

func parseStargateMsgs(jsonData any, msgs *[]wasmvm.StargateMsg) {
	switch v := jsonData.(type) {
	case map[string]interface{}:
		if typeURL, ok := v["type_url"].(string); ok {
			if value, ok := v["value"].(string); ok {
				*msgs = append(*msgs, wasmvm.StargateMsg{
					TypeURL: typeURL,
					Value:   []byte(value),
				})
			}
		}
		for _, value := range v {
			parseStargateMsgs(value, msgs)
		}
	case []interface{}:
		for _, value := range v {
			parseStargateMsgs(value, msgs)
		}
	}
}

type StargateMsgDecoded struct {
	TypeURL string `json:"type_url"`
	Value   string `json:"value"`
}

func DecodeBase64StargateMsgs(
	jsonBz []byte, context client.Context,
) (newSgMsgs []StargateMsgDecoded, err error) {
	codec := context.Codec

	var data interface{}
	if err := json.Unmarshal(jsonBz, &data); err != nil {
		return []StargateMsgDecoded{}, err
	}

	sgMsgs, err := YieldStargateMsgs(jsonBz)
	if err != nil {
		return
	}
	for _, sgMsg := range sgMsgs {
		valueStr := string(sgMsg.Value)
		value := strings.Replace(string(sgMsg.Value), `\"`, `"`, -1)
		value = strings.Replace(value, `"{`, `{`, -1)
		value = strings.Replace(value, `}"`, `}`, -1)

		if _, err := base64.StdEncoding.DecodeString(valueStr); err == nil {
			protoMsg, err := context.InterfaceRegistry.Resolve(sgMsg.TypeURL)
			if err != nil {
				return newSgMsgs, err
			}

			decodedBz, _ := base64.StdEncoding.Strict().DecodeString(string(sgMsg.Value))
			concrete := protoMsg.(sdkcodec.ProtoMarshaler)

			err = codec.Unmarshal(decodedBz, concrete)
			if err != nil {
				return newSgMsgs, err
			}

			outBytes, err := codec.MarshalJSON(concrete)
			if err != nil {
				return newSgMsgs, err
			}

			newSgMsgs = append(newSgMsgs, StargateMsgDecoded{sgMsg.TypeURL, string(outBytes)})
		} else if _, err := json.Marshal(value); err == nil {
			newSgMsgs = append(newSgMsgs, StargateMsgDecoded{sgMsg.TypeURL, string(sgMsg.Value)})
		} else {
			return newSgMsgs, fmt.Errorf(
				"parse error: encountered wasmvm.StargateMsg with unexpected format: %s", sgMsg)
		}
	}
	return newSgMsgs, nil
}

// DecodeBase64Cmd creates a cobra command for base64 decoding.
func DecodeBase64Cmd(defaultNodeHome string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "base64-decode",
		Short: "Decode a base64-encoded protobuf message",
		Long: `Decode a base64-encoded protobuf message from JSON input.
The input should be a JSON object with 'type_url' and 'value' fields.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)

			outMessage, err := DecodeBase64StargateMsgs([]byte(args[0]), clientCtx)
			fmt.Println(outMessage)

			return err
		},
	}

	cmd.Flags().String(flags.FlagHome, defaultNodeHome, "The application home directory")

	return cmd
}
