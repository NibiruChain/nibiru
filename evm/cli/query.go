package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/spf13/cobra"

	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/x/nutil/flags"

	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/evm"
)

// QueryEvmBalanceResp is the JSON-only response shape for a flattened
// multi-VM token balance query.
//
// Optional token fields are represented as pointers so MarshalIndent emits
// "null" when the corresponding bank or ERC20 representation is unavailable.
type QueryEvmBalanceResp struct {
	AddrEVM    string `json:"addr_evm"`
	AddrBech32 string `json:"addr_bech32"`
	BalanceWei string `json:"balance_wei"`

	Erc20Addr         *string `json:"erc20_addr"`
	Erc20Symbol       *string `json:"erc20_symbol"`
	Erc20BalanceHuman *string `json:"erc20_balance_human"`
	Erc20Decimals     *uint8  `json:"erc20_decimals"`
	Erc20Name         *string `json:"erc20_name"`
	Erc20BalanceBase  *string `json:"erc20_balance_base"`

	BankSymbol       *string `json:"bank_symbol"`
	BankBalanceHuman *string `json:"bank_balance_human"`
	BankDecimals     *uint32 `json:"bank_decimals"`
	BankCoinDenom    *string `json:"bank_coin_denom"`
	BankBalanceBase  *string `json:"bank_balance_base"`
}

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
		CmdQueryBalance(),
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
%s query %s get-fun-token ibc/abcdef
%s query %s get-fun-token 0x7D4B7B8CA7E1a24928Bb96D59249c7a5bd1DfBe6
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

func CmdQueryBalance() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "balance [eoa-addr] [token]",
		Short: "Query a token balance across Bank and EVM representations",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Query token balances in a multi-VM context.

Examples:
  # USDC on Nibiru mainnet
  addr="0xYourAddr" token="0x0829F361A05D993d5CEb035cA6DF3446b060970b"
  %s query %s balance "$addr" "$token"

  # NIBI via bank denom
  addr="0xYourAddr" token="unibi"  # Bank coin denom works
  %s query %s balance "$addr" "$token"

  # NIBI via canonical WNIBI ERC20
  addr="0xYourAddr" token="0x0CaCF669f8446BeCA826913a3c6B96aCD4b02a97" # ERC20 addr works
  %s query %s balance "$addr" "$token"
`,
				version.AppName, evm.ModuleName,
				version.AppName, evm.ModuleName,
				version.AppName, evm.ModuleName,
			),
		),
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			evmQuery := evm.NewQueryClient(clientCtx)
			tokenInput := args[1]

			req := &evm.QueryBalanceRequest{Address: args[0], Token: tokenInput}
			addrBech32, err := req.Validate()
			if err != nil {
				return err
			}

			resp := QueryEvmBalanceResp{
				AddrBech32: addrBech32.String(),
			}
			if len(addrBech32.Bytes()) == appconst.ADDR_LEN_EOA {
				resp.AddrEVM = eth.NibiruAddrToEthAddr(addrBech32).Hex()
			}

			strPtr := func(s string) *string { return &s }
			u32Ptr := func(x uint32) *uint32 { return &x }

			queryResp, err := evmQuery.Balance(cmd.Context(), req)
			if err != nil {
				return err
			}
			resp.BalanceWei = queryResp.BalanceWei
			if bank := queryResp.Bank; bank != nil {
				resp.BankCoinDenom = strPtr(bank.CoinDenom)
				resp.BankBalanceBase = strPtr(bank.BalanceBase)
				resp.BankBalanceHuman = strPtr(bank.BalanceHuman)
				resp.BankSymbol = strPtr(bank.Symbol)
				resp.BankDecimals = u32Ptr(bank.Decimals)
			}
			if erc20 := queryResp.Erc20; erc20 != nil {
				resp.Erc20Addr = strPtr(erc20.Address)
				if erc20.Symbol != "" {
					resp.Erc20Symbol = strPtr(erc20.Symbol)
				}
				if erc20.BalanceHuman != "" {
					resp.Erc20BalanceHuman = strPtr(erc20.BalanceHuman)
				}
				decimals := uint8(erc20.Decimals)
				resp.Erc20Decimals = &decimals
				if erc20.Name != "" {
					resp.Erc20Name = strPtr(erc20.Name)
				}
				if erc20.BalanceBase != "" {
					resp.Erc20BalanceBase = strPtr(erc20.BalanceBase)
				}
			}

			bz, err := json.MarshalIndent(resp, "", "  ")
			if err != nil {
				return err
			}
			cmd.Println(string(bz))
			return nil
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

			addrBech32, err := req.Validate()
			if err != nil {
				return err
			}

			offline, _ := cmd.Flags().GetBool("offline")

			if offline {
				addrEth := eth.NibiruAddrToEthAddr(addrBech32)
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
