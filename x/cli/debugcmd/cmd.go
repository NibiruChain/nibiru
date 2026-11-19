package debugcmd

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/client"
	sdkdebug "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/client/debug"
)

// AnyDecoder decodes JSON holding base64-encoded type_url/value messages.
type AnyDecoder func(jsonBz []byte, clientCtx client.Context) (any, error)

// EnvFromContext builds the decoder environment from a client context.
func EnvFromContext(clientCtx client.Context, decodeAny AnyDecoder) Env {
	env := Env{
		Codec:    clientCtx.Codec,
		Registry: clientCtx.InterfaceRegistry,
		TxConfig: clientCtx.TxConfig,
	}
	if decodeAny != nil {
		env.DecodeAny = func(bz []byte) (any, error) {
			return decodeAny(bz, clientCtx)
		}
	}
	return env
}

const (
	flagAs       = "as"
	flagTypeURL  = "type-url"
	flagPath     = "path"
	flagData     = "data"
	flagResponse = "response"
)

// Cmd builds `nibid debug`. With an argument (or piped stdin) it classifies
// the input and decodes it; the subcommands are explicit forms of the same
// decoders, plus the SDK's address, pubkey, and bytes helpers.
func Cmd(decodeAny AnyDecoder) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "debug [input]",
		Short: "Decode, convert, and build encoded data",
		Long: `Detect what an input is and decode it, offline. The input is an
argument, or stdin when the argument is "-" or absent and stdin is piped.

Recognized inputs:
  JSON-RPC body or batch (abci_query, broadcast_tx_*)   abci-query
  JSON with type_url and value, or a stargate wrapper   any
  base64 or hex transaction                             tx
  base64 or hex SimulateRequest / BroadcastTxRequest    abci-query
  bech32 address (nibi1...), 0x EVM address, hex        address
  bech32 pubkey, proto-JSON pubkey, raw key bytes       pubkey
  [10 21 13 255]                                        raw-bytes
  other valid protobuf                                  unknown-protobuf

The output is JSON with a "detected" field. A 32-byte value fits several
kinds (transaction hash, Wasm contract address, key); all candidates are
reported. Use --as to choose one.`,
		Example: `  nibid debug nibi1ah8gqrtjllhc5ld4rxgl4uglvwl93ag0sh6e6v
  nibid debug 0xAb...
  pbpaste | nibid debug
  nibid debug --as address 0x<64 hex chars>`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)
			in, err := readInput(cmd, args)
			if err != nil {
				return err
			}
			opts, err := optionsFromFlags(cmd)
			if err != nil {
				return err
			}
			res, derr := Decode(EnvFromContext(clientCtx, decodeAny), in, opts)
			cmd.SilenceUsage = true
			if perr := printJSON(cmd, res); perr != nil {
				return perr
			}
			return derr
		},
	}
	cmd.Flags().String(flagAs, "", "Treat the input as this kind (tx, any, "+
		"abci-query, address, pubkey, raw-bytes, protobuf, tx-hash)")
	cmd.Flags().String(flagTypeURL, "",
		"Decode protobuf input as this message type, e.g. /cosmos.bank.v1beta1.MsgSend")
	cmd.Flags().String(flagPath, "",
		"gRPC query path for raw abci-query data, e.g. /cosmwasm.wasm.v1.Query/SmartContractState")
	cmd.Flags().Bool(flagResponse, false,
		"Decode abci-query data as the response, not the request")

	cmd.AddCommand(
		txCmd(),
		msgCmd(decodeAny),
		abciQueryCmd(),
		NewStargateCmd("stargate [message json]", false),
		sdkdebug.PubkeyCmd(),
		sdkdebug.PubkeyRawCmd(),
		sdkdebug.AddrCmd(),
		sdkdebug.RawBytesCmd(),
		sdkdebug.PrefixesCmd(),
	)
	return cmd
}

func optionsFromFlags(cmd *cobra.Command) (Options, error) {
	var opts Options
	as, _ := cmd.Flags().GetString(flagAs)
	switch strings.ToLower(as) {
	case "":
	case "msg", "any":
		opts.As = KindAny
	case "tx", "tx-hash", "address", "pubkey", "raw-bytes", "protobuf",
		"abci-query":
		opts.As = Kind(strings.ToLower(as))
	default:
		return opts, fmt.Errorf("unknown --as kind %q", as)
	}
	opts.TypeURL, _ = cmd.Flags().GetString(flagTypeURL)
	opts.Path, _ = cmd.Flags().GetString(flagPath)
	opts.Response, _ = cmd.Flags().GetBool(flagResponse)
	return opts, nil
}

// readInput returns the argument, or stdin when the argument is "-" or when
// there is no argument and stdin is piped.
func readInput(cmd *cobra.Command, args []string) ([]byte, error) {
	if len(args) > 1 {
		return nil, errors.New("expected one input; quote JSON so the shell passes it as one argument")
	}
	if len(args) == 1 && args[0] != "-" {
		return []byte(args[0]), nil
	}
	in := cmd.InOrStdin()
	if len(args) == 0 {
		if f, ok := in.(*os.File); ok {
			st, err := f.Stat()
			if err != nil || st.Mode()&os.ModeCharDevice != 0 {
				return nil, errors.New(
					"no input: pass an argument or pipe data on stdin")
			}
		}
	}
	bz, err := io.ReadAll(in)
	if err != nil {
		return nil, err
	}
	if len(strings.TrimSpace(string(bz))) == 0 {
		return nil, errors.New("no input: stdin was empty")
	}
	return bz, nil
}

func printJSON(cmd *cobra.Command, v any) error {
	bz, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(cmd.OutOrStdout(), string(bz))
	return err
}

// decodeBinaryArg reads base64 (default) or hex protobuf bytes.
func decodeBinaryArg(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if bz, err := hex.DecodeString(strings.TrimPrefix(s, "0x")); err == nil {
		return bz, nil
	}
	bz, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, errors.New("input is neither hex nor base64")
	}
	return bz, nil
}

func txCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tx [base64|hex]",
		Short: "Decode a transaction (same output as `tx decode`)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)
			in, err := readInput(cmd, args)
			if err != nil {
				return err
			}
			bz, err := decodeBinaryArg(string(in))
			if err != nil {
				return err
			}
			out, err := DecodeTx(EnvFromContext(clientCtx, nil), bz)
			if err != nil {
				return err
			}
			return clientCtx.PrintBytes(out)
		},
	}
}

func msgCmd(decodeAny AnyDecoder) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "msg [json | protobuf bytes]",
		Short: "Decode base64 messages in JSON, or protobuf bytes of one type",
		Long: `Without --type-url, the input is JSON with type_url and value fields
(a stargate wrapper, or several nested) and every message is decoded.
With --type-url, the input is the base64 or hex protobuf bytes of one message
of that type.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)
			env := EnvFromContext(clientCtx, decodeAny)
			in, err := readInput(cmd, args)
			if err != nil {
				return err
			}
			typeURL, _ := cmd.Flags().GetString(flagTypeURL)
			var out any
			if typeURL == "" {
				out, err = env.DecodeAny(in)
			} else {
				var bz []byte
				if bz, err = decodeBinaryArg(string(in)); err != nil {
					return err
				}
				out, err = DecodeMsg(env, typeURL, bz)
			}
			if err != nil {
				return err
			}
			return printJSON(cmd, out)
		},
	}
	cmd.Flags().String(flagTypeURL, "", "Message type of the protobuf bytes")
	return cmd
}

func abciQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "abci-query [json-rpc body]",
		Short: "Decode an abci_query or broadcast_tx request",
		Long: `Decode a JSON-RPC body or batch copied from a browser or node log. The
gRPC path selects the request type, so any query decodes with real field names.
A Simulate or BroadcastTx request also decodes its transaction in full.

Or give the pieces: --path /pkg.Service/Method --data <hex>. Add --response to
decode the data as the method's response.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)
			env := EnvFromContext(clientCtx, nil)
			path, _ := cmd.Flags().GetString(flagPath)
			dataHex, _ := cmd.Flags().GetString(flagData)
			response, _ := cmd.Flags().GetBool(flagResponse)
			cmd.SilenceUsage = true

			if path != "" {
				if dataHex == "" && len(args) == 1 {
					dataHex = args[0]
				}
				bz, err := hex.DecodeString(strings.TrimPrefix(dataHex, "0x"))
				if err != nil {
					return fmt.Errorf("--data must be hex: %w", err)
				}
				out, err := DecodeABCIRequest(env, path, bz, response)
				if err != nil {
					return err
				}
				return printJSON(cmd, out)
			}
			in, err := readInput(cmd, args)
			if err != nil {
				return err
			}
			out, derr := DecodeJSONRPC(env, in)
			if out != nil {
				if err := printJSON(cmd, out); err != nil {
					return err
				}
			}
			return derr
		},
	}
	cmd.Flags().String(flagPath, "", "gRPC query path, e.g. /cosmos.auth.v1beta1.Query/Account")
	cmd.Flags().String(flagData, "", "Hex data of the query request")
	cmd.Flags().Bool(flagResponse, false, "Decode the data as the response type")
	return cmd
}
