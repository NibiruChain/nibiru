package cmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cosmos/gogoproto/proto"
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	wasmvm "github.com/CosmWasm/wasmvm/types"
)

// YieldStargateMsgs takes a byte slice of JSON data and converts it into a slice
// of wasmvm.StargateMsg objects. This function is essential for processing
// JSON-formatted messages that contain base64-encoded protobuf messages.
//
// Args:
//   - jsonBz []byte: A byte slice containing the JSON data to be parsed.
//
// Returns:
//   - sgMsgs []wasmvm.StargateMsg: A slice of wasmvm.StargateMsg objects parsed
//     from the provided JSON data.
//   - err error: An error object, which is nil if the operation is successful.
func YieldStargateMsgs(jsonBz []byte) (sgMsgs []wasmvm.StargateMsg, err error) {
	var data interface{}
	if err := json.Unmarshal(jsonBz, &data); err != nil {
		return sgMsgs, err
	}

	parseStargateMsgs(data, &sgMsgs)
	return sgMsgs, nil
}

// parseStargateMsgs is a recursive function used by YieldStargateMsgs to
// traverse the JSON data, filter for any protobuf.Any messages in the
// "WasmVM.StargateMsg" format and decode them from base64 back to human-readable
// form as JSON objects.
//
// Args:
//   - jsonData any: JSON data to parse. According to the JSON specification,
//     possible value types are:
//     Null, Bool, Number(f64), String, Array, or Object(Map<String, Value>)
//   - msgs *[]wasmvm.StargateMsg: Mutable reference to a slice of protobuf
//     messages. These are potentially altered in place if the value is an
//     encoded base 64 string.
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

// StargateMsgDecoded is a struct designed to hold a decoded version of a
// "wasmvm.StargateMsg".
type StargateMsgDecoded struct {
	TypeURL string `json:"type_url"`
	Value   string `json:"value"`
}

// DecodeBase64StargateMsgs decodes a series of base64-encoded
// wasmvm.StargateMsg objects from the provided JSON byte slice (jsonBz).
// This function is vital for extracting and interpreting the contents of these
// protobuf messages.
//
// Args:
//   - jsonBz []byte: JSON data containing potential base64-encoded messages.
//   - clientCtx client.Context: Context for the `nibid` CLI.
//
// Returns:
//   - newSgMsgs []StargateMsgDecoded: The decoded stargate messages.
//   - err error: An error object, which is nil if the operation is successful.
func DecodeBase64StargateMsgs(
	jsonBz []byte, clientCtx client.Context,
) (newSgMsgs []StargateMsgDecoded, err error) {
	codec := clientCtx.Codec

	var data interface{}
	if err := json.Unmarshal(jsonBz, &data); err != nil {
		return newSgMsgs, fmt.Errorf(
			"failed to decode stargate msgs due to invalid JSON: %w", err)
	}

	sgMsgs, err := YieldStargateMsgs(jsonBz)
	if err != nil {
		return newSgMsgs, err
	}
	for _, sgMsg := range sgMsgs {
		valueStr := string(sgMsg.Value)
		replacer := strings.NewReplacer(
			`\"`, `"`, // old, new
			`"{`, `{`,
			`}"`, `}`,
		)
		value := replacer.Replace(string(sgMsg.Value))

		if _, err := base64.StdEncoding.DecodeString(valueStr); err == nil {
			protoMsg, err := clientCtx.InterfaceRegistry.Resolve(sgMsg.TypeURL)
			if err != nil {
				return newSgMsgs, err
			}

			decodedBz, _ := base64.StdEncoding.Strict().DecodeString(string(sgMsg.Value))
			concrete := protoMsg.(proto.Message)

			err = codec.Unmarshal(decodedBz, concrete)
			if err != nil {
				return newSgMsgs, err
			}

			outBytes, err := codec.MarshalJSON(concrete)
			if err != nil {
				return newSgMsgs, err
			}

			newSgMsgs = append(newSgMsgs,
				StargateMsgDecoded{sgMsg.TypeURL, string(outBytes)},
			)
		} else if _, err := json.Marshal(value); err == nil {
			newSgMsgs = append(newSgMsgs,
				StargateMsgDecoded{sgMsg.TypeURL, string(sgMsg.Value)},
			)
		} else {
			return newSgMsgs, fmt.Errorf(
				"parse error: encountered wasmvm.StargateMsg with unexpected format: %s", sgMsg)
		}
	}
	return newSgMsgs, nil
}

// DecodeBase64Cmd creates a Cobra command used to decode base64-encoded protobuf
// messages from a JSON input. This function enables users to input arbitrary
// JSON strings and parse the contents of base-64 encoded protobuf.Any messages.
func DecodeBase64Cmd(defaultNodeHome string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "base64-decode",
		Short: "Decode a base64-encoded protobuf message",
		Long: `Decode a base64-encoded protobuf message from JSON input.
The input should be a JSON object with 'type_url' and 'value' fields.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)

			outMessage, err := DecodeBase64StargateMsgs([]byte(args[0]), clientCtx)
			if err == nil {
				fmt.Println(outMessage)
			}
			return err
		},
	}

	cmd.Flags().String(flags.FlagHome, defaultNodeHome, "The application home directory")

	return cmd
}
