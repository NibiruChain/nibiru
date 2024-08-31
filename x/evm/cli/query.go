package cli

import (
	"fmt"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/spf13/cobra"

	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/evm"
)

// GetQueryCmd returns a cli command for this module's queries
func GetQueryCmd() *cobra.Command {
	moduleQueryCmd := &cobra.Command{
		Use: evm.ModuleName,
		Short: fmt.Sprintf(
			"Query commands for the x/%s module", evm.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	// Add subcommands
	cmds := []*cobra.Command{
		CmdQueryFunToken(),
		CmdQueryAccount(),
	}
	for _, cmd := range cmds {
		moduleQueryCmd.AddCommand(cmd)
	}
	return moduleQueryCmd
}

// CmdQueryFunToken returns fungible token mapping for either bank coin or erc20 addr
func CmdQueryFunToken() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "funtoken [coin-or-erc20addr]",
		Short: "Query evm fungible token mapping",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Query evm fungible token mapping.

Examples:
$ %s query %s get-fun-token ibc/abcdef
$ %s query %s get-fun-token 0x7D4B7B8CA7E1a24928Bb96D59249c7a5bd1DfBe6
`,
				version.AppName, evm.ModuleName,
				version.AppName, evm.ModuleName,
			),
		),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := evm.NewQueryClient(clientCtx)

			res, err := queryClient.FunTokenMapping(cmd.Context(), &evm.QueryFunTokenMappingRequest{
				Token: args[0],
			})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func CmdQueryAccount() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "account [address]",
		Short: "Query account by its hex address or bech32",
		Long:  strings.TrimSpace(""),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := evm.NewQueryClient(clientCtx)

			req := &evm.QueryEthAccountRequest{
				Address: args[0],
			}

			isBech32, err := req.Validate()
			fmt.Printf("TODO: UD-DEBUG: req.String(): %v\n", req.String())
			fmt.Printf("TODO: UD-DEBUG: err: %v\n", err)
			if err != nil {
				return err
			}

			offline, _ := cmd.Flags().GetBool("offline")

			if offline {
				var addrEth gethcommon.Address
				var addrBech32 sdk.AccAddress

				if isBech32 {
					addrBech32 = sdk.MustAccAddressFromBech32(req.Address)
					addrEth = eth.NibiruAddrToEthAddr(addrBech32)
				} else {
					addrEth = gethcommon.HexToAddress(req.Address)
					addrBech32 = eth.EthAddrToNibiruAddr(addrEth)
				}

				resp := new(evm.QueryEthAccountResponse)
				resp.EthAddress = addrEth.Hex()
				resp.Bech32Address = addrBech32.String()
				return clientCtx.PrintProto(resp)
			}

			resp, err := queryClient.EthAccount(cmd.Context(), req)
			if err != nil {
				return fmt.Errorf("consider using the \"--offline\" flag: %w", err)
			}

			return clientCtx.PrintProto(resp)
		},
	}
	cmd.Flags().Bool("offline", false, "Skip the query and only return addresses.")
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
