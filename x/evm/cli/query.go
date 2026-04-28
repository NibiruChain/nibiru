package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/version"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/spf13/cobra"
	grpccodes "google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"

	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/x/nutil/flags"

	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/embeds"
)

// QueryEvmBalanceResp is the JSON-only response shape for a flattened
// multi-VM token balance query.
//
// Optional token fields are represented as pointers so MarshalIndent emits
// "null" when the corresponding bank or ERC20 representation is unavailable.
type QueryEvmBalanceResp struct {
	AddrEVM    string `json:"addr_evm"`
	AddrBech32 string `json:"addr_bech32"`

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

func formatUnitsBig(amount *big.Int, decimals uint32) string {
	if amount == nil || amount.Sign() == 0 {
		return "0"
	}

	sign := ""
	absAmount := new(big.Int).Set(amount)
	if absAmount.Sign() < 0 {
		sign = "-"
		absAmount.Neg(absAmount)
	}
	if decimals == 0 {
		return sign + absAmount.String()
	}

	scale := new(big.Int).Exp(big.NewInt(10), new(big.Int).SetUint64(uint64(decimals)), nil)
	whole := new(big.Int)
	frac := new(big.Int)
	whole.QuoRem(absAmount, scale, frac)
	if frac.Sign() == 0 {
		return sign + whole.String()
	}

	fracStr := frac.String()
	if len(fracStr) < int(decimals) {
		fracStr = strings.Repeat("0", int(decimals)-len(fracStr)) + fracStr
	}
	fracStr = strings.TrimRight(fracStr, "0")
	if fracStr == "" {
		return sign + whole.String()
	}

	return sign + whole.String() + "." + fracStr
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
			bankQuery := banktypes.NewQueryClient(clientCtx)
			tokenInput := args[1]

			accountReq := &evm.QueryEthAccountRequest{Address: args[0]}
			addrBech32, err := accountReq.Validate()
			if err != nil {
				return err
			}
			addrEVM := eth.NibiruAddrToEthAddr(addrBech32)

			resp := QueryEvmBalanceResp{
				AddrEVM:    addrEVM.Hex(),
				AddrBech32: addrBech32.String(),
			}

			strPtr := func(s string) *string { return &s }
			u8Ptr := func(x uint8) *uint8 { return &x }
			u32Ptr := func(x uint32) *uint32 { return &x }
			jsonOut := func() error {
				bz, err := json.MarshalIndent(resp, "", "  ")
				if err != nil {
					return err
				}
				cmd.Println(string(bz))
				return nil
			}
			bankDecimalsFromMetadata := func(metadata banktypes.Metadata) uint32 {
				var decimals uint32
				for _, unit := range metadata.DenomUnits {
					if unit.Exponent > decimals {
						decimals = unit.Exponent
					}
				}
				return decimals
			}
			loadBankSide := func(denom string) (bool, error) {
				metaResp, err := bankQuery.DenomMetadata(cmd.Context(), &banktypes.QueryDenomMetadataRequest{Denom: denom})
				if err != nil {
					if grpcstatus.Code(err) == grpccodes.NotFound {
						return false, nil
					}
					return false, err
				}
				balResp, err := bankQuery.Balance(cmd.Context(), &banktypes.QueryBalanceRequest{
					Address: addrBech32.String(),
					Denom:   denom,
				})
				if err != nil {
					return false, err
				}
				decimals := bankDecimalsFromMetadata(metaResp.Metadata)
				amountBig := new(big.Int).Set(balResp.Balance.Amount.BigInt())
				amount := amountBig.String()
				balance := formatUnitsBig(amountBig, decimals)

				resp.BankCoinDenom = strPtr(denom)
				resp.BankBalanceBase = strPtr(amount)
				resp.BankBalanceHuman = strPtr(balance)
				resp.BankSymbol = strPtr(metaResp.Metadata.Symbol)
				resp.BankDecimals = u32Ptr(decimals)
				return true, nil
			}
			loadERC20Side := func(contract common.Address, useNativeNibi bool) (bool, error) {
				erc20ABI := embeds.SmartContract_ERC20MinterWithMetadataUpdates.ABI

				callERC20 := func(method string, methodArgs ...any) (*evm.MsgEthereumTxResponse, error) {
					input, err := erc20ABI.Pack(method, methodArgs...)
					if err != nil {
						return nil, err
					}
					from := evm.EVM_READONLY_ADDR
					to := contract
					inputHex := hexutil.Bytes(input)
					argsBz, err := json.Marshal(&evm.JsonTxArgs{
						From:  &from,
						To:    &to,
						Input: &inputHex,
					})
					if err != nil {
						return nil, err
					}
					evmResp, err := evmQuery.EthCall(cmd.Context(), &evm.EthCallRequest{
						Args:   argsBz,
						GasCap: evm.Erc20GasLimitQuery,
					})
					if err != nil {
						return nil, err
					}
					if evmResp.VmError != "" {
						return nil, errors.New(evmResp.VmError)
					}
					return evmResp, nil
				}

				var (
					foundAny    bool
					amountStr   *string
					amountBig   *big.Int
					decimalsVal uint8
					decimalsPtr *uint8
				)

				if useNativeNibi {
					balResp, err := evmQuery.Balance(cmd.Context(), &evm.QueryBalanceRequest{
						Address: addrEVM.Hex(),
					})
					if err != nil {
						return false, err
					}
					var ok bool
					amountBig, ok = new(big.Int).SetString(balResp.BalanceWei, 10)
					if !ok {
						return false, fmt.Errorf("invalid EVM balance %q", balResp.BalanceWei)
					}
					amount := amountBig.String()
					amountStr = strPtr(amount)
					foundAny = true
				} else {
					evmResp, err := callERC20("balanceOf", addrEVM)
					if err == nil {
						out := new(struct{ Value *big.Int })
						if unpackErr := erc20ABI.UnpackIntoInterface(out, "balanceOf", evmResp.Ret); unpackErr == nil && out.Value != nil {
							amountBig = out.Value
							amount := out.Value.String()
							amountStr = strPtr(amount)
							foundAny = true
						}
					}
				}

				if evmResp, err := callERC20("name"); err == nil {
					outString := new(struct{ Value string })
					if unpackErr := erc20ABI.UnpackIntoInterface(outString, "name", evmResp.Ret); unpackErr == nil {
						resp.Erc20Name = strPtr(outString.Value)
						foundAny = true
					} else {
						outBytes32 := new(struct{ Value [32]byte })
						if unpackErr := erc20ABI.UnpackIntoInterface(outBytes32, "name", evmResp.Ret); unpackErr == nil {
							resp.Erc20Name = strPtr(strings.TrimRight(string(outBytes32.Value[:]), "\x00"))
							foundAny = true
						}
					}
				}

				if evmResp, err := callERC20("symbol"); err == nil {
					outString := new(struct{ Value string })
					if unpackErr := erc20ABI.UnpackIntoInterface(outString, "symbol", evmResp.Ret); unpackErr == nil {
						resp.Erc20Symbol = strPtr(outString.Value)
						foundAny = true
					} else {
						outBytes32 := new(struct{ Value [32]byte })
						if unpackErr := erc20ABI.UnpackIntoInterface(outBytes32, "symbol", evmResp.Ret); unpackErr == nil {
							resp.Erc20Symbol = strPtr(strings.TrimRight(string(outBytes32.Value[:]), "\x00"))
							foundAny = true
						}
					}
				}

				if evmResp, err := callERC20("decimals"); err == nil {
					outUint8 := new(struct{ Value uint8 })
					if unpackErr := erc20ABI.UnpackIntoInterface(outUint8, "decimals", evmResp.Ret); unpackErr == nil {
						decimalsVal = outUint8.Value
						decimalsPtr = u8Ptr(decimalsVal)
						foundAny = true
					} else {
						outBigInt := new(struct{ Value *big.Int })
						if unpackErr := erc20ABI.UnpackIntoInterface(outBigInt, "decimals", evmResp.Ret); unpackErr == nil && outBigInt.Value != nil {
							decimalsVal = uint8(outBigInt.Value.Uint64())
							decimalsPtr = u8Ptr(decimalsVal)
							foundAny = true
						}
					}
				}

				if !foundAny {
					return false, nil
				}

				addr := contract.Hex()
				resp.Erc20Addr = strPtr(addr)
				resp.Erc20BalanceBase = amountStr
				resp.Erc20Decimals = decimalsPtr
				if amountBig != nil && decimalsPtr != nil {
					balance := formatUnitsBig(amountBig, uint32(*decimalsPtr))
					resp.Erc20BalanceHuman = strPtr(balance)
				}
				return true, nil
			}

			paramsResp, err := evmQuery.Params(cmd.Context(), &evm.QueryParamsRequest{})
			if err != nil {
				return err
			}
			canonicalWnibi := paramsResp.Params.CanonicalWnibi.Address

			switch {
			case tokenInput == appconst.DENOM_UNIBI || strings.EqualFold(tokenInput, canonicalWnibi.Hex()):
				if _, err := loadBankSide(appconst.DENOM_UNIBI); err != nil {
					return err
				}
				if _, err := loadERC20Side(canonicalWnibi, true); err != nil {
					return err
				}
				return jsonOut()
			default:
				funtokenResp, err := evmQuery.FunTokenMapping(cmd.Context(), &evm.QueryFunTokenMappingRequest{
					Token: tokenInput,
				})
				if err == nil && funtokenResp.FunToken != nil {
					if _, err := loadBankSide(funtokenResp.FunToken.BankDenom); err != nil {
						return err
					}
					if _, err := loadERC20Side(funtokenResp.FunToken.Erc20Addr.Address, false); err != nil {
						return err
					}
					return jsonOut()
				}
				if err != nil && grpcstatus.Code(err) != grpccodes.NotFound {
					return err
				}
			}

			if common.IsHexAddress(tokenInput) {
				if _, err := loadERC20Side(common.HexToAddress(tokenInput), false); err != nil {
					return err
				}
				return jsonOut()
			}

			if sdk.ValidateDenom(tokenInput) == nil {
				if _, err := loadBankSide(tokenInput); err != nil {
					return err
				}
			}

			return jsonOut()
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
