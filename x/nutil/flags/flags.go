package flags

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	sdkflags "github.com/cosmos/cosmos-sdk/client/flags"
)

func TxFlagSet() *pflag.FlagSet {
	fs := new(cobra.Command).Flags()
	fs.StringP(sdkflags.FlagOutput, "o", "json", "Output format (text|json)")
	fs.String(sdkflags.FlagFrom, "", "Name or address of private key with which to sign")
	fs.Uint64P(sdkflags.FlagAccountNumber, "a", 0, "The account number of the signing account (offline mode only)")
	fs.Uint64P(sdkflags.FlagSequence, "s", 0, "The sequence number of the signing account (offline mode only)")
	fs.String(sdkflags.FlagNote, "", "Note to add a description to the transaction (previously --memo)")
	fs.String(sdkflags.FlagFees, "", "Fees to pay along with transaction; eg: 10unibi")
	fs.String(sdkflags.FlagGasPrices, "", "Gas prices in decimal format to determine the transaction fee (e.g. 0.1unibi)")
	fs.String(sdkflags.FlagNode, "tcp://localhost:26657", "<host>:<port> to tendermint rpc interface for this chain")
	fs.Bool(sdkflags.FlagUseLedger, false, "Use a connected Ledger device")
	fs.Float64(sdkflags.FlagGasAdjustment, sdkflags.DefaultGasAdjustment, "adjustment factor to be multiplied against the estimate returned by the tx simulation; if the gas limit is set manually this flag is ignored ")
	fs.StringP(sdkflags.FlagBroadcastMode, "b", sdkflags.BroadcastSync, "Transaction broadcasting mode (sync|async)")
	fs.Bool(sdkflags.FlagDryRun, false, "ignore the --gas flag and perform a simulation of a transaction, but don't broadcast it (when enabled, the local Keybase is not accessible)")
	fs.Bool(sdkflags.FlagGenerateOnly, false, "Build an unsigned transaction and write it to STDOUT (when enabled, the local Keybase only accessed when providing a key name)")
	fs.Bool(sdkflags.FlagOffline, false, "Offline mode (does not allow any online functionality)")
	fs.BoolP(sdkflags.FlagSkipConfirmation, "y", false, "Skip tx broadcasting prompt confirmation")
	fs.String(sdkflags.FlagSignMode, "", "Choose sign mode (direct|amino-json|direct-aux), this is an advanced feature")
	fs.Uint64(sdkflags.FlagTimeoutHeight, 0, "Set a block timeout height to prevent the tx from being committed past a certain height")
	fs.String(sdkflags.FlagFeePayer, "", "Fee payer pays fees for the transaction instead of deducting from the signer")
	fs.String(sdkflags.FlagFeeGranter, "", "Fee granter grants fees for the transaction")
	fs.String(sdkflags.FlagTip, "", "Tip is the amount that is going to be transferred to the fee payer on the target chain. This flag is only valid when used with --aux, and is ignored if the target chain didn't enable the TipDecorator")
	fs.Bool(sdkflags.FlagAux, false, "Generate aux signer data instead of sending a tx")
	fs.String(sdkflags.FlagChainID, "", "The network chain ID")
	// --gas can accept integers and "auto"
	fs.String(sdkflags.FlagGas, "", fmt.Sprintf("gas limit to set per-transaction; set to %q to calculate sufficient gas automatically. Note: %q option doesn't always report accurate results. Set a valid coin value to adjust the result. Can be used instead of %q. (default %d)",
		sdkflags.GasFlagAuto, sdkflags.GasFlagAuto, sdkflags.FlagFees, sdkflags.DefaultGasLimit))

	AddKeyringFlags(fs)

	return fs
}

// AddTxFlagsToCmd adds common flags to a module tx command.
func AddTxFlagsToCmd(cmd *cobra.Command) {
	cmdFlagsSet := cmd.Flags()

	txFs := TxFlagSet()
	txFs.VisitAll(func(f *pflag.Flag) {
		f.Hidden = true
	})

	flagNameHelpVerbose := "help-verbose"
	cmd.PersistentFlags().BoolP(flagNameHelpVerbose, "v", false, "Show all flags common to each command")
	origHelpFn := cmd.HelpFunc()
	cmd.SetHelpFunc(func(c *cobra.Command, args []string) {
		if show, _ := cmd.Flags().GetBool(flagNameHelpVerbose); show {
			cmd.Flags().VisitAll(func(f *pflag.Flag) {
				f.Hidden = false
			})
		}
		origHelpFn(cmd, args)
	})

	cmdFlagsSet.AddFlagSet(txFs)
}

// AddKeyringFlags sets common keyring flags
func AddKeyringFlags(flags *pflag.FlagSet) {
	flags.String(sdkflags.FlagKeyringDir, "", "The client Keyring directory; if omitted, the default 'home' directory will be used")
	flags.String(sdkflags.FlagKeyringBackend, sdkflags.DefaultKeyringBackend, "Select keyring's backend (os|file|kwallet|pass|test|memory)")
}

// AddQueryFlagsToCmd adds common flags to a module query command.
func AddQueryFlagsToCmd(cmd *cobra.Command) {
	cmd.Flags().String(FlagNode, "tcp://localhost:26657", "<host>:<port> to Tendermint RPC interface for this chain")
	cmd.Flags().String(FlagGRPC, "", "the gRPC endpoint to use for this chain")
	cmd.Flags().Bool(FlagGRPCInsecure, false, "allow gRPC over insecure channels, if not TLS the server must use TLS")
	cmd.Flags().Int64(FlagHeight, 0, "Use a specific height to query state at (this can error if the node is pruning state)")
	cmd.Flags().StringP(FlagOutput, "o", "text", "Output format (text|json)")

	// some base commands does not require chainID e.g `simd testnet` while subcommands do
	// hence the flag should not be required for those commands
	_ = cmd.MarkFlagRequired(FlagChainID)
}

// AddPaginationFlagsToCmd adds common pagination flags to cmd
func AddPaginationFlagsToCmd(cmd *cobra.Command, query string) {
	cmd.Flags().Uint64(FlagPage, 1, fmt.Sprintf("pagination page of %s to query for. This sets offset to a multiple of limit", query))
	cmd.Flags().String(FlagPageKey, "", fmt.Sprintf("pagination page-key of %s to query for", query))
	cmd.Flags().Uint64(FlagOffset, 0, fmt.Sprintf("pagination offset of %s to query for", query))
	cmd.Flags().Uint64(FlagLimit, 100, fmt.Sprintf("pagination limit of %s to query for", query))
	cmd.Flags().Bool(FlagCountTotal, false, fmt.Sprintf("count total number of records in %s to query for", query))
	cmd.Flags().Bool(FlagReverse, false, "results are sorted in descending order")
}

const (
	// DefaultKeyringBackend
	DefaultKeyringBackend = sdkflags.DefaultKeyringBackend

	// BroadcastSync defines a tx broadcasting mode where the client waits for
	// a CheckTx execution response only.
	BroadcastSync = sdkflags.BroadcastSync
	// BroadcastAsync defines a tx broadcasting mode where the client returns
	// immediately.
	BroadcastAsync = sdkflags.BroadcastAsync

	// SignModeDirect is the value of the --sign-mode flag for SIGN_MODE_DIRECT
	SignModeDirect = sdkflags.SignModeDirect
	// SignModeLegacyAminoJSON is the value of the --sign-mode flag for SIGN_MODE_LEGACY_AMINO_JSON
	SignModeLegacyAminoJSON = sdkflags.SignModeLegacyAminoJSON
	// SignModeDirectAux is the value of the --sign-mode flag for SIGN_MODE_DIRECT_AUX
	SignModeDirectAux = sdkflags.SignModeDirectAux
	// SignModeEIP191 is the value of the --sign-mode flag for SIGN_MODE_EIP_191
	SignModeEIP191 = sdkflags.SignModeEIP191
)

// List of CLI flags
const (
	FlagHome             = sdkflags.FlagHome
	FlagKeyringDir       = sdkflags.FlagKeyringDir
	FlagUseLedger        = sdkflags.FlagUseLedger
	FlagChainID          = sdkflags.FlagChainID
	FlagNode             = sdkflags.FlagNode
	FlagGRPC             = sdkflags.FlagGRPC
	FlagGRPCInsecure     = sdkflags.FlagGRPCInsecure
	FlagHeight           = sdkflags.FlagHeight
	FlagGasAdjustment    = sdkflags.FlagGasAdjustment
	FlagFrom             = sdkflags.FlagFrom
	FlagName             = sdkflags.FlagName
	FlagAccountNumber    = sdkflags.FlagAccountNumber
	FlagSequence         = sdkflags.FlagSequence
	FlagNote             = sdkflags.FlagNote
	FlagFees             = sdkflags.FlagFees
	FlagGas              = sdkflags.FlagGas
	FlagGasPrices        = sdkflags.FlagGasPrices
	FlagBroadcastMode    = sdkflags.FlagBroadcastMode
	FlagDryRun           = sdkflags.FlagDryRun
	FlagGenerateOnly     = sdkflags.FlagGenerateOnly
	FlagOffline          = sdkflags.FlagOffline
	FlagOutputDocument   = sdkflags.FlagOutputDocument
	FlagSkipConfirmation = sdkflags.FlagSkipConfirmation
	FlagProve            = sdkflags.FlagProve
	FlagKeyringBackend   = sdkflags.FlagKeyringBackend
	FlagPage             = sdkflags.FlagPage
	FlagLimit            = sdkflags.FlagLimit
	FlagSignMode         = sdkflags.FlagSignMode
	FlagPageKey          = sdkflags.FlagPageKey
	FlagOffset           = sdkflags.FlagOffset
	FlagCountTotal       = sdkflags.FlagCountTotal
	FlagTimeoutHeight    = sdkflags.FlagTimeoutHeight
	FlagKeyType          = sdkflags.FlagKeyType
	FlagFeePayer         = sdkflags.FlagFeePayer
	FlagFeeGranter       = sdkflags.FlagFeeGranter
	FlagReverse          = sdkflags.FlagReverse
	FlagTip              = sdkflags.FlagTip
	FlagAux              = sdkflags.FlagAux
	FlagInitHeight       = sdkflags.FlagInitHeight
	// FlagOutput is the flag to set the output format.
	// This differs from FlagOutputDocument that is used to set the output file.
	FlagOutput = sdkflags.FlagOutput

	// Tendermint logging flags
	FlagLogLevel   = sdkflags.FlagLogLevel
	FlagLogFormat  = sdkflags.FlagLogFormat
	FlagLogNoColor = sdkflags.FlagLogNoColor
)
