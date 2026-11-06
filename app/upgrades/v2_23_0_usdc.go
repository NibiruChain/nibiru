package upgrades

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	gethcore "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/app/keepers"
	abci "github.com/NibiruChain/nibiru/v2/bft/abci/types"
	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/evm"
	"github.com/NibiruChain/nibiru/v2/evm/embeds"
	"github.com/NibiruChain/nibiru/v2/evm/evmstate"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/module"
	query "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/query"
	banktypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/bank/types"
	upgradetypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/upgrade/types"
	wasmkeeper "github.com/NibiruChain/nibiru/v2/x/wasm/keeper"
	wasmtypes "github.com/NibiruChain/nibiru/v2/x/wasm/types"
)

const (
	USDCHolderSetChecksumV223 = "0a18618d6d1dcf9ef0cf4a92252abf879b96a22066b1e7e57e2efd1c958204c8"
	USDCHolderSetSaltV223     = "usdc-holders-v1"
	USDCHolderSetCreatorV223  = "nibi1rlvdjfmxkyfj4tzu73p8m4g2h4y89xccf9622l"
)

// USDCMigrationConfigV223 identifies one reviewed deployment. The same profile
// can be exercised on any chain with matching predeployed contracts and mapping.
// Tests supply their own explicit profile rather than mutating production state.
type USDCMigrationConfigV223 struct {
	Registry            sdk.AccAddress
	RegistryChecksum    []byte
	Token               gethcommon.Address
	BankDenom           string
	Safe                gethcommon.Address
	SaiEvm              gethcommon.Address
	ReplacedBankHolders []sdk.AccAddress
}

// DefaultUSDCMigrationConfigV223 returns the reviewed mainnet identities.
// Contract bytes and creator/salt must be finalized before registry deployment.
// Testnet activation requires these same identities to be provisioned explicitly;
// merely deploying an address set does not bypass token/mapping/recipient checks.
func DefaultUSDCMigrationConfigV223() USDCMigrationConfigV223 {
	checksum, err := hex.DecodeString(USDCHolderSetChecksumV223)
	if err != nil {
		panic(err) // compile-time deployment constant
	}
	return USDCMigrationConfigV223{
		Registry: wasmkeeper.BuildContractAddressPredictable(checksum,
			sdk.MustAccAddressFromBech32(USDCHolderSetCreatorV223), []byte(USDCHolderSetSaltV223), nil),
		RegistryChecksum: checksum,
		Token:            gethcommon.HexToAddress("0x0829F361A05D993d5CEb035cA6DF3446b060970b"),
		BankDenom:        "erc20/0x0829F361A05D993d5CEb035cA6DF3446b060970b",
		Safe:             gethcommon.HexToAddress("0x22CBd7CbF3b33681abB3Ced4D64d71acB9a9dCd2"),
		SaiEvm:           gethcommon.HexToAddress("0x25589C8071fa10B7a8c62CdCA7035c539242A774"),
		ReplacedBankHolders: []sdk.AccAddress{
			sdk.MustAccAddressFromBech32("nibi1ntmw2dfvd0qnw5fnwdu9pev2hsnqfdj9ny9n0nzh2a5u8v0scflq930mph"),
			sdk.MustAccAddressFromBech32("nibi193m2a00pmdsvkcvugrfewqzhtq6k0srkjzvxp2sk357vlpspx5vqxu8d7p"),
			sdk.MustAccAddressFromBech32("nibi1waf5c8z55qvjay4de8wkm9cxyt6wa8zdnrvlexjrq77lqgqf258q3yn7l8"),
		},
	}
}

var Upgrade2_23_0 = Upgrade{UpgradeName: "v2.23.0", Handler: Handler_v2_23{}}

type Handler_v2_23 struct {
	USDC *USDCMigrationConfigV223
}

var _ HandlerImpl = Handler_v2_23{}

// Handler includes the enabling migrations before best-effort USDC collection.
// Mainnet upgrades directly from v2.21; testnet upgrades from its completed
// v2.22 release. A missing registry skips only the custom USDC step.
func (h Handler_v2_23) Handler(mm *module.Manager, cfg module.Configurator, nibiru *keepers.PublicKeepers) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		versions, err := mm.RunMigrations(ctx, cfg, fromVM)
		if err != nil {
			return nil, err
		}
		// Consensus creation can use the existing native parameter store as a
		// fallback. Upgrade-state activation is required for completion.
		if err := ensureXConsensus(ctx, nibiru); err != nil {
			ctx.Logger().Error("v2.23.0 upgrade failure", "err", err)
			ctx.EventManager().EmitEvent(NewEventUpgradeFailure("v2.23.0", err))
		}
		if err := nibiru.UpgradeKeeper.ActivateContract(ctx); err != nil {
			return nil, fmt.Errorf("activate x-upgrade: %w", err)
		}
		profile := DefaultUSDCMigrationConfigV223()
		if h.USDC != nil {
			profile = *h.USDC
		}
		if err := runUSDCMigrationV223(ctx, nibiru, profile); err != nil {
			ctx.Logger().Error("v2.23.0 USDC migration failure", "err", err)
			ctx.EventManager().EmitEvent(NewEventUpgradeFailure("v2.23.0", err))
		}
		return versions, nil
	}
}

func ensureXConsensus(ctx sdk.Context, nibiru *keepers.PublicKeepers) (err error) {
	defer func() {
		if value := recover(); value != nil {
			err = fmt.Errorf("create x-consensus: panic: %v", value)
		}
	}()
	return nibiru.ConsensusKeeper.EnsureXConsensus(ctx)
}

func usdcPanicErrorV223(value any) error {
	return fmt.Errorf("USDC migration panic: %v", value)
}

func usdcSummaryV223(ctx sdk.Context, cfg USDCMigrationConfigV223, status, reason string, extra ...sdk.Attribute) {
	attributes := []sdk.Attribute{
		sdk.NewAttribute("upgrade", "v2.23.0"), sdk.NewAttribute("height", strconv.FormatInt(ctx.BlockHeight(), 10)),
		sdk.NewAttribute("status", status), sdk.NewAttribute("registry", cfg.Registry.String()),
		sdk.NewAttribute("token", cfg.Token.Hex()), sdk.NewAttribute("denom", cfg.BankDenom),
		sdk.NewAttribute("safe", cfg.Safe.Hex()), sdk.NewAttribute("reason", reason),
	}
	ctx.EventManager().EmitEvent(sdk.NewEvent("usdc_migration", append(attributes, extra...)...))
}

func usdcAccountV223(ctx sdk.Context, cfg USDCMigrationConfigV223, representation, source, amount, disposition, status, reason string) {
	ctx.EventManager().EmitEvent(sdk.NewEvent("usdc_migration_account",
		sdk.NewAttribute("upgrade", "v2.23.0"), sdk.NewAttribute("representation", representation),
		sdk.NewAttribute("source", source), sdk.NewAttribute("destination", cfg.Safe.Hex()),
		sdk.NewAttribute("amount", amount), sdk.NewAttribute("disposition", disposition),
		sdk.NewAttribute("status", status), sdk.NewAttribute("reason", reason)))
}

func runUSDCMigrationV223(ctx sdk.Context, nibiru *keepers.PublicKeepers, cfg USDCMigrationConfigV223) (err error) {
	defer func() {
		if value := recover(); value != nil {
			err = usdcPanicErrorV223(value)
		}
		if err != nil {
			usdcSummaryV223(ctx, cfg, "failed", err.Error())
		}
	}()
	if err := sdk.VerifyAddressFormat(cfg.Registry); err != nil {
		return err
	}
	if nibiru.WasmKeeper.GetContractInfo(ctx, cfg.Registry) == nil {
		usdcSummaryV223(ctx, cfg, "skipped", "registry_not_deployed")
		return nil
	}
	candidates, err := ReadUSDCHolderSet(ctx, nibiru.WasmKeeper, cfg.Registry, cfg.RegistryChecksum)
	if err != nil {
		return err
	}
	if err := validateUSDCProfileV223(ctx, nibiru, cfg); err != nil {
		return err
	}
	// Finish all index traversal before conversions remove keys from that index.
	var bankOwners []*banktypes.DenomOwner
	var pageKey []byte
	for {
		page, err := nibiru.BankKeeper.DenomOwners(sdk.WrapSDKContext(ctx), &banktypes.QueryDenomOwnersRequest{
			Denom: cfg.BankDenom, Pagination: &query.PageRequest{Key: pageKey, Limit: 200},
		})
		if err != nil {
			return fmt.Errorf("enumerate bank holders: %w", err)
		}
		bankOwners = append(bankOwners, page.DenomOwners...)
		if page.Pagination == nil || len(page.Pagination.NextKey) == 0 {
			break
		}
		pageKey = page.Pagination.NextKey
	}
	usdcSummaryV223(ctx, cfg, "started", "")
	for _, holder := range bankOwners {
		source, err := sdk.AccAddressFromBech32(holder.Address)
		if err != nil {
			return err
		}
		disposition := "repayment"
		for _, replaced := range cfg.ReplacedBankHolders {
			if bytes.Equal(replaced, source) {
				disposition = "custody_recovery"
			}
		}
		coin := nibiru.BankKeeper.GetBalance(ctx, source, cfg.BankDenom)
		if coin.IsZero() {
			usdcAccountV223(ctx, cfg, "bank", source.String(), "0", disposition, "zero", "")
			continue
		}
		err = cachedUSDCOperationV223(ctx, func(cached sdk.Context) error {
			return convertUSDCBankV223(cached, nibiru, cfg, source, coin)
		})
		if err != nil {
			usdcAccountV223(ctx, cfg, "bank", source.String(), coin.Amount.String(), disposition, "failed", err.Error())
			continue
		}
		usdcAccountV223(ctx, cfg, "bank", source.String(), coin.Amount.String(), disposition, "collected", "")
	}
	// Always include the explicitly controlled source, even if it was missing
	// from an index export. The destination and module are exclusions, not EOAs.
	seen := make(map[gethcommon.Address]bool)
	candidates = append(candidates, cfg.SaiEvm)
	for _, source := range candidates {
		if seen[source] {
			continue
		}
		seen[source] = true
		if source == evm.EVM_MODULE_ADDRESS || source == cfg.Safe {
			continue
		}
		disposition := "repayment"
		if source == cfg.SaiEvm {
			disposition = "custody_recovery"
		}
		amount := new(big.Int)
		status, reason := "collected", ""
		err := cachedUSDCOperationV223(ctx, func(cached sdk.Context) error {
			sdb := nibiru.EvmKeeper.NewSDB(cached, nibiru.EvmKeeper.TxConfig(cached, gethcommon.Hash{}))
			queriedAmount, err := loadERC20ValueV218(nibiru.EvmKeeper, sdb, embeds.SmartContract_ERC20MinterWithMetadataUpdates.ABI, cfg.Token, "balanceOf", source)
			if err != nil {
				return err
			}
			amount = queriedAmount
			if amount.Sign() == 0 {
				status = "zero"
				return nil
			}
			if source != cfg.SaiEvm && sdb.GetCodeSize(source) > 0 {
				status = "skipped"
				reason = "unrelated_contract"
				return nil
			}
			resp, err := transferUSDCV223(nibiru.EvmKeeper, sdb, source, cfg.Safe, cfg.Token, amount)
			if err != nil {
				return err
			}
			if err := sdb.Ctx().EventManager().EmitTypedEvent(&evm.EventTxLog{Logs: resp.Logs}); err != nil {
				return err
			}
			sdb.Commit()
			return nil
		})
		if err != nil {
			status = "failed"
			reason = err.Error()
		}
		usdcAccountV223(ctx, cfg, "erc20", source.Hex(), amount.String(), disposition, status, reason)
	}
	// Donations to module escrow do not create bank-holder entitlements. Keep
	// them in escrow and report the residual separately from collected funds.
	sdb := nibiru.EvmKeeper.NewSDB(ctx, nibiru.EvmKeeper.TxConfig(ctx, gethcommon.Hash{}))
	escrow, err := loadERC20ValueV218(nibiru.EvmKeeper, sdb, embeds.SmartContract_ERC20MinterWithMetadataUpdates.ABI, cfg.Token, "balanceOf", evm.EVM_MODULE_ADDRESS)
	if err != nil {
		return fmt.Errorf("read remaining USDC escrow: %w", err)
	}
	bankSupply := nibiru.BankKeeper.GetSupply(ctx, cfg.BankDenom).Amount.BigInt()
	usdcSummaryV223(ctx, cfg, "completed", "",
		sdk.NewAttribute("bank_supply_remaining", bankSupply.String()),
		sdk.NewAttribute("module_escrow_remaining", escrow.String()),
		sdk.NewAttribute("escrow_excess", new(big.Int).Sub(escrow, bankSupply).String()))
	return nil
}

func validateUSDCProfileV223(ctx sdk.Context, nibiru *keepers.PublicKeepers, cfg USDCMigrationConfigV223) error {
	zero := gethcommon.Address{}
	if cfg.Token == zero || cfg.Safe == zero || cfg.SaiEvm == zero || cfg.Safe == cfg.Token || cfg.Safe == cfg.SaiEvm || cfg.Safe == evm.EVM_MODULE_ADDRESS || cfg.Token == evm.EVM_MODULE_ADDRESS || cfg.SaiEvm == cfg.Token || cfg.SaiEvm == evm.EVM_MODULE_ADDRESS {
		return fmt.Errorf("invalid USDC token/recipient profile")
	}
	if err := sdk.ValidateDenom(cfg.BankDenom); err != nil {
		return err
	}
	mapping := nibiru.EvmKeeper.FunTokens.Collect(ctx, nibiru.EvmKeeper.FunTokens.Indexes.BankDenom.ExactMatch(ctx, cfg.BankDenom))
	if len(mapping) != 1 || mapping[0].Erc20Addr.Address != cfg.Token || mapping[0].IsMadeFromCoin {
		return fmt.Errorf("USDC profile requires the exact ERC20-originated FunToken mapping")
	}
	sdb := nibiru.EvmKeeper.NewSDB(ctx, nibiru.EvmKeeper.TxConfig(ctx, gethcommon.Hash{}))
	if sdb.GetCodeSize(cfg.Token) == 0 || sdb.GetCodeSize(cfg.Safe) == 0 || sdb.GetCodeSize(cfg.SaiEvm) == 0 {
		return fmt.Errorf("USDC token, Safe and SaiEvm must be deployed for this profile")
	}
	abi := embeds.SmartContract_ERC20MinterWithMetadataUpdates.ABI
	zeroValue := new(big.Int)
	queryMsg := core.Message{
		To: &cfg.Token, From: evm.EVM_READONLY_ADDR, Value: zeroValue,
		GasLimit: evm.Erc20GasLimitQuery, GasPrice: zeroValue, GasFeeCap: zeroValue,
		GasTipCap: zeroValue, BlobGasFeeCap: zeroValue,
	}
	evmObj := nibiru.EvmKeeper.NewEVM(sdb.Ctx(), queryMsg, nibiru.EvmKeeper.GetEVMConfig(ctx), nil, sdb)
	decimals, err := nibiru.EvmKeeper.ERC20().LoadERC20Decimals(sdb.Ctx(), evmObj, abi, cfg.Token)
	if err != nil || decimals != 6 {
		return fmt.Errorf("USDC profile requires six decimals: %v", err)
	}
	escrow, err := loadERC20ValueV218(nibiru.EvmKeeper, sdb, abi, cfg.Token, "balanceOf", evm.EVM_MODULE_ADDRESS)
	if err != nil {
		return err
	}
	if escrow.Cmp(nibiru.BankKeeper.GetSupply(ctx, cfg.BankDenom).Amount.BigInt()) < 0 {
		return fmt.Errorf("USDC module escrow is below bank supply")
	}
	return nil
}

func cachedUSDCOperationV223(ctx sdk.Context, operation func(sdk.Context) error) (err error) {
	defer func() {
		if value := recover(); value != nil {
			err = usdcPanicErrorV223(value)
		}
	}()
	cached, commit := ctx.CacheContext()
	if err := operation(cached); err != nil {
		return err
	}
	commit()
	return nil
}

func convertUSDCBankV223(ctx sdk.Context, nibiru *keepers.PublicKeepers, cfg USDCMigrationConfigV223, source sdk.AccAddress, coin sdk.Coin) error {
	k := nibiru.EvmKeeper
	querySDB := k.NewSDB(ctx, k.TxConfig(ctx, gethcommon.Hash{}))
	abi := embeds.SmartContract_ERC20MinterWithMetadataUpdates.ABI
	before, err := loadERC20ValueV218(k, querySDB, abi, cfg.Token, "balanceOf", cfg.Safe)
	if err != nil {
		return err
	}
	supply, err := loadERC20ValueV218(k, querySDB, abi, cfg.Token, "totalSupply")
	if err != nil {
		return err
	}
	_, err = k.ConvertCoinToEvm(sdk.WrapSDKContext(ctx), &evm.MsgConvertCoinToEvm{Sender: source.String(), ToEthAddr: eth.EIP55Addr{Address: cfg.Safe}, BankCoin: coin})
	if err != nil {
		return err
	}
	afterSDB := k.NewSDB(ctx, k.TxConfig(ctx, gethcommon.Hash{}))
	after, err := loadERC20ValueV218(k, afterSDB, abi, cfg.Token, "balanceOf", cfg.Safe)
	if err != nil {
		return err
	}
	afterSupply, err := loadERC20ValueV218(k, afterSDB, abi, cfg.Token, "totalSupply")
	if err != nil {
		return err
	}
	if new(big.Int).Sub(after, before).Cmp(coin.Amount.BigInt()) != 0 || afterSupply.Cmp(supply) != 0 || !nibiru.BankKeeper.GetBalance(ctx, source, cfg.BankDenom).IsZero() {
		return fmt.Errorf("bank conversion did not preserve exact USDC amount and supply")
	}
	return nil
}

func transferUSDCV223(k *evmstate.Keeper, sdb *evmstate.SDB, source, destination, token gethcommon.Address, amount *big.Int) (*evm.MsgEthereumTxResponse, error) {
	abi := embeds.SmartContract_ERC20MinterWithMetadataUpdates.ABI
	before, err := loadERC20ValueV218(k, sdb, abi, token, "balanceOf", destination)
	if err != nil {
		return nil, err
	}
	supply, err := loadERC20ValueV218(k, sdb, abi, token, "totalSupply")
	if err != nil {
		return nil, err
	}
	input, err := abi.Pack("transfer", destination, amount)
	if err != nil {
		return nil, err
	}
	zero := new(big.Int)
	nonce := sdb.GetNonce(source)
	msg := core.Message{To: &token, From: source, Nonce: nonce, Value: zero, GasLimit: evm.Erc20GasLimitExecute, GasPrice: zero, GasFeeCap: zero, GasTipCap: zero, Data: input, AccessList: gethcore.AccessList{}, BlobGasFeeCap: zero, BlobHashes: []gethcommon.Hash{}, SkipNonceChecks: true, SkipFromEOACheck: true}
	evmObj := k.NewEVM(sdb.Ctx(), msg, k.GetEVMConfig(sdb.Ctx()), nil, sdb)
	resp, err := k.ApplyEvmMsg(msg, evmObj, evm.COMMIT_READONLY)
	if err != nil {
		return nil, err
	}
	if resp.Failed() {
		return nil, fmt.Errorf("USDC transfer VM failure: %s", resp.VmError)
	}
	var result evmstate.ERC20Bool
	if err := abi.UnpackIntoInterface(&result, "transfer", resp.Ret); err != nil {
		return nil, err
	}
	if !result.Value {
		return nil, fmt.Errorf("USDC transfer returned false")
	}
	sdb.SetNonce(source, nonce)
	after, err := loadERC20ValueV218(k, sdb, abi, token, "balanceOf", destination)
	if err != nil {
		return nil, err
	}
	remaining, err := loadERC20ValueV218(k, sdb, abi, token, "balanceOf", source)
	if err != nil {
		return nil, err
	}
	afterSupply, err := loadERC20ValueV218(k, sdb, abi, token, "totalSupply")
	if err != nil {
		return nil, err
	}
	if remaining.Sign() != 0 || new(big.Int).Sub(after, before).Cmp(amount) != 0 || afterSupply.Cmp(supply) != 0 {
		return nil, fmt.Errorf("USDC transfer did not preserve exact amount and supply")
	}
	return resp, nil
}

const usdcHolderPageSize = 200

// USDCHolderSetKeeper is the read-only Wasm surface used by the migration.
// The holder set must already be deployed before the upgrade runs.
type USDCHolderSetKeeper interface {
	GetContractInfo(sdk.Context, sdk.AccAddress) *wasmtypes.ContractInfo
	GetCodeInfo(sdk.Context, uint64) *wasmtypes.CodeInfo
	QuerySmart(sdk.Context, sdk.AccAddress, []byte) ([]byte, error)
}

// ReadUSDCHolderSet reads candidates, not balances or transfer permissions.
// It checks the deployed bytecode and validates every page before returning any
// candidates. Callers must not start moving funds until this succeeds.
// The gzipped contract fixture is test-only; this function never deploys code.
func ReadUSDCHolderSet(
	ctx sdk.Context, keeper USDCHolderSetKeeper, contract sdk.AccAddress, checksum []byte,
) ([]gethcommon.Address, error) {
	if err := sdk.VerifyAddressFormat(contract); err != nil {
		return nil, fmt.Errorf("USDC holder set: invalid contract address: %w", err)
	}
	if len(checksum) != 32 {
		return nil, fmt.Errorf("USDC holder set: expected a 32-byte checksum")
	}
	info := keeper.GetContractInfo(ctx, contract)
	if info == nil {
		return nil, fmt.Errorf("USDC holder set %s is not deployed", contract)
	}
	code := keeper.GetCodeInfo(ctx, info.CodeID)
	if code == nil || !bytes.Equal(code.CodeHash, checksum) {
		return nil, fmt.Errorf("USDC holder set %s has unexpected bytecode", contract)
	}

	var candidates []gethcommon.Address
	var cursor *string
	previous := ""
	for {
		request := struct {
			Addresses struct {
				StartAfter *string `json:"start_after"`
				Limit      uint32  `json:"limit"`
			} `json:"addresses"`
		}{}
		request.Addresses.StartAfter = cursor
		request.Addresses.Limit = usdcHolderPageSize
		msg, err := json.Marshal(request)
		if err != nil {
			return nil, err
		}
		bz, err := keeper.QuerySmart(ctx, contract, msg)
		if err != nil {
			return nil, fmt.Errorf("query USDC holder set: %w", err)
		}
		var page struct {
			Addresses      []string `json:"addresses"`
			NextStartAfter *string  `json:"next_start_after"`
		}
		if err := json.Unmarshal(bz, &page); err != nil {
			return nil, fmt.Errorf("decode USDC holder set: %w", err)
		}
		if len(page.Addresses) > usdcHolderPageSize {
			return nil, fmt.Errorf("USDC holder set: oversized page")
		}
		for _, address := range page.Addresses {
			if !gethcommon.IsHexAddress(address) || len(address) != 42 ||
				!strings.HasPrefix(address, "0x") || address != strings.ToLower(address) {
				return nil, fmt.Errorf("USDC holder set: noncanonical address %q", address)
			}
			if address <= previous {
				return nil, fmt.Errorf("USDC holder set: addresses are not strictly increasing")
			}
			previous = address
			candidates = append(candidates, gethcommon.HexToAddress(address))
		}
		if page.NextStartAfter == nil {
			return candidates, nil
		}
		if len(page.Addresses) == 0 || *page.NextStartAfter != previous {
			return nil, fmt.Errorf("USDC holder set: cursor does not match the last address")
		}
		cursor = page.NextStartAfter
	}
}

// USDCMigrationAccount preserves the original representation and recipient.
// Amount is an exact integer in legacy token base units. Only collected entries
// create a repayment or custody-recovery allocation.
type USDCMigrationAccount struct {
	Representation string `json:"representation"`
	Source         string `json:"source"`
	Destination    string `json:"destination"`
	Amount         string `json:"amount_base_units"`
	Disposition    string `json:"disposition"`
	Status         string `json:"status"`
	Reason         string `json:"reason,omitempty"`
	Error          string `json:"error,omitempty"`
}

// USDCMigrationReport is reconstructed without keeper state or a pre-upgrade
// balance ledger. Context records must match real conversion/transfer events.
type USDCMigrationReport struct {
	Upgrade               string                 `json:"upgrade"`
	Height                int64                  `json:"height"`
	Status                string                 `json:"status"`
	Registry              string                 `json:"registry"`
	Token                 string                 `json:"token"`
	BankDenom             string                 `json:"bank_denom"`
	Safe                  string                 `json:"safe"`
	Reason                string                 `json:"reason,omitempty"`
	Error                 string                 `json:"error,omitempty"`
	Collected             string                 `json:"collected_base_units"`
	Repayment             string                 `json:"repayment_base_units"`
	CustodyRecovery       string                 `json:"custody_recovery_base_units"`
	BankSupplyRemaining   string                 `json:"bank_supply_remaining_base_units,omitempty"`
	ModuleEscrowRemaining string                 `json:"module_escrow_remaining_base_units,omitempty"`
	EscrowExcess          string                 `json:"escrow_excess_base_units,omitempty"`
	Accounts              []USDCMigrationAccount `json:"accounts"`
}

// ReconstructUSDCMigration consumes the upgrade block's ABCI events, including
// typed bank conversion events and EVM logs. It uses the migration's start/end
// markers to exclude unrelated transactions in that same block. Repeated zero
// transaction hashes/log indices are deliberately not used for deduplication.
// Missing/duplicate records, incomplete framing, or unmatched movements return
// no report rather than an incomplete repayment plan.
func ReconstructUSDCMigration(events []abci.Event) (*USDCMigrationReport, error) {
	var report *USDCMigrationReport
	active, closed := false, false
	evidence := make(map[string]int)
	seen := make(map[string]bool)
	collected, repayment, recovery := new(big.Int), new(big.Int), new(big.Int)
	for _, event := range events {
		if event.Type == "usdc_migration" {
			attrs, err := usdcReportAttributes(event)
			if err != nil {
				return nil, err
			}
			if attrs["upgrade"] != "v2.23.0" {
				continue
			}
			status := attrs["status"]
			if report == nil {
				height, err := strconv.ParseInt(attrs["height"], 10, 64)
				if err != nil || height < 1 {
					return nil, fmt.Errorf("USDC migration: invalid event height")
				}
				report = &USDCMigrationReport{
					Upgrade: attrs["upgrade"], Height: height, Status: status,
					Registry: attrs["registry"], Token: attrs["token"], BankDenom: attrs["denom"],
					Safe: attrs["safe"], Reason: attrs["reason"], Error: attrs["error"],
					Collected: "0", Repayment: "0", CustodyRecovery: "0", Accounts: []USDCMigrationAccount{},
				}
				switch status {
				case "started":
					if !gethcommon.IsHexAddress(report.Token) || !gethcommon.IsHexAddress(report.Safe) || report.BankDenom == "" {
						return nil, fmt.Errorf("USDC migration: invalid token, Safe or bank denom")
					}
					active = true
				case "skipped", "failed":
					closed = true
				default:
					return nil, fmt.Errorf("USDC migration: expected start/skip/failure event")
				}
				continue
			}
			if !active || closed || (status != "completed" && status != "failed") {
				return nil, fmt.Errorf("USDC migration: duplicate or unexpected summary event")
			}
			for key, want := range map[string]string{"registry": report.Registry, "token": report.Token, "denom": report.BankDenom, "safe": report.Safe, "height": strconv.FormatInt(report.Height, 10)} {
				if attrs[key] != want {
					return nil, fmt.Errorf("USDC migration: closing event changed %s", key)
				}
			}
			report.Status, report.Reason, report.Error = status, attrs["reason"], attrs["error"]
			for key, target := range map[string]*string{
				"bank_supply_remaining":   &report.BankSupplyRemaining,
				"module_escrow_remaining": &report.ModuleEscrowRemaining,
				"escrow_excess":           &report.EscrowExcess,
			} {
				if value, present := attrs[key]; present {
					amount, ok := new(big.Int).SetString(value, 10)
					if !ok || amount.Sign() < 0 || amount.BitLen() > 256 || amount.String() != value {
						return nil, fmt.Errorf("USDC migration: invalid %s audit amount", key)
					}
					*target = value
				}
			}
			active, closed = false, true
			continue
		}
		if !active {
			// Account records outside a migration frame must not be silently lost.
			if event.Type == "usdc_migration_account" {
				attrs, err := usdcReportAttributes(event)
				if err != nil {
					return nil, err
				}
				if attrs["upgrade"] == "v2.23.0" {
					return nil, fmt.Errorf("USDC migration: account outside migration frame")
				}
			}
			continue
		}
		switch event.Type {
		case "eth.evm.v1.EventConvertCoinToEvm":
			parsed, err := sdk.ParseTypedEvent(event)
			if err != nil {
				return nil, fmt.Errorf("decode bank conversion event: %w", err)
			}
			conversion, ok := parsed.(*evm.EventConvertCoinToEvm)
			if !ok {
				return nil, fmt.Errorf("USDC migration: unexpected conversion event type")
			}
			if !strings.EqualFold(conversion.Erc20ContractAddress, report.Token) || !strings.EqualFold(conversion.ToEthAddr, report.Safe) {
				continue
			}
			if !conversion.BankCoin.IsValid() || !conversion.BankCoin.IsPositive() || conversion.BankCoin.Denom != report.BankDenom {
				return nil, fmt.Errorf("USDC migration: invalid bank conversion coin")
			}
			matches := 0
			for _, log := range conversion.EvmLogs {
				from, amount, match, err := usdcReportTransfer(log, report.Token, report.Safe)
				if err != nil {
					return nil, err
				}
				if match {
					if from != evm.EVM_MODULE_ADDRESS || amount.Cmp(conversion.BankCoin.Amount.BigInt()) != 0 {
						return nil, fmt.Errorf("USDC migration: bank conversion and ERC20 backing transfer disagree")
					}
					matches++
				}
			}
			if matches != 1 {
				return nil, fmt.Errorf("USDC migration: bank conversion needs exactly one backing transfer")
			}
			evidence[usdcReportKey("bank", conversion.Sender, conversion.BankCoin.Amount.String())]++
		case evm.TypeUrlEventTxLog:
			logs, err := evm.EventTxLogFromABCIEvent(event)
			if err != nil {
				return nil, err
			}
			for _, log := range logs.Logs {
				from, amount, match, err := usdcReportTransfer(log.ToLogLite(), report.Token, report.Safe)
				if err != nil {
					return nil, err
				}
				if match {
					evidence[usdcReportKey("erc20", from.Hex(), amount.String())]++
				}
			}
		case "usdc_migration_account":
			attrs, err := usdcReportAttributes(event)
			if err != nil {
				return nil, err
			}
			if attrs["upgrade"] != report.Upgrade {
				return nil, fmt.Errorf("USDC migration: account upgrade mismatch")
			}
			account := USDCMigrationAccount{
				Representation: attrs["representation"], Source: attrs["source"], Destination: attrs["destination"],
				Amount: attrs["amount"], Disposition: attrs["disposition"], Status: attrs["status"],
				Reason: attrs["reason"], Error: attrs["error"],
			}
			amount, ok := new(big.Int).SetString(account.Amount, 10)
			if !ok || amount.Sign() < 0 || amount.BitLen() > 256 || amount.String() != account.Amount {
				return nil, fmt.Errorf("USDC migration: invalid account amount")
			}
			if !strings.EqualFold(account.Destination, report.Safe) {
				return nil, fmt.Errorf("USDC migration: account destination differs from Safe")
			}
			switch account.Representation {
			case "bank":
				address, err := sdk.GetFromBech32(account.Source, appconst.AccountAddressPrefix)
				if err != nil {
					return nil, fmt.Errorf("USDC migration: invalid bank holder: %w", err)
				}
				if err := sdk.VerifyAddressFormat(address); err != nil {
					return nil, fmt.Errorf("USDC migration: invalid bank holder: %w", err)
				}
			case "erc20":
				if !gethcommon.IsHexAddress(account.Source) {
					return nil, fmt.Errorf("USDC migration: invalid ERC20 holder")
				}
			default:
				return nil, fmt.Errorf("USDC migration: invalid representation")
			}
			key := usdcReportKey(account.Representation, account.Source, "")
			if seen[key] {
				return nil, fmt.Errorf("USDC migration: duplicate account record")
			}
			seen[key] = true
			switch account.Status {
			case "collected":
				if amount.Sign() == 0 || (account.Disposition != "repayment" && account.Disposition != "custody_recovery") {
					return nil, fmt.Errorf("USDC migration: invalid collection disposition or zero collection")
				}
				evidenceKey := usdcReportKey(account.Representation, account.Source, account.Amount)
				if evidence[evidenceKey] < 1 {
					return nil, fmt.Errorf("USDC migration: collected record has no matching movement for %s", account.Source)
				}
				evidence[evidenceKey]--
				collected.Add(collected, amount)
				if account.Disposition == "repayment" {
					repayment.Add(repayment, amount)
				} else {
					recovery.Add(recovery, amount)
				}
			case "zero":
				if amount.Sign() != 0 {
					return nil, fmt.Errorf("USDC migration: nonzero zero-balance record")
				}
			case "failed", "skipped":
			default:
				return nil, fmt.Errorf("USDC migration: invalid account status")
			}
			report.Accounts = append(report.Accounts, account)
		}
	}
	if report == nil || !closed || active {
		return nil, fmt.Errorf("USDC migration: missing or incomplete migration frame")
	}
	for _, count := range evidence {
		if count != 0 {
			return nil, fmt.Errorf("USDC migration: unmatched token movement")
		}
	}
	report.Collected, report.Repayment, report.CustodyRecovery = collected.String(), repayment.String(), recovery.String()
	return report, nil
}

func usdcReportAttributes(event abci.Event) (map[string]string, error) {
	attrs := make(map[string]string, len(event.Attributes))
	for _, attr := range event.Attributes {
		if _, exists := attrs[attr.Key]; exists {
			return nil, fmt.Errorf("USDC migration: duplicate event attribute %q", attr.Key)
		}
		attrs[attr.Key] = attr.Value
	}
	return attrs, nil
}

func usdcReportKey(representation, source, amount string) string {
	if representation == "erc20" {
		source = strings.ToLower(source)
	}
	return representation + "|" + source + "|" + amount
}

func usdcReportTransfer(log evm.LogLite, token, safe string) (gethcommon.Address, *big.Int, bool, error) {
	if !strings.EqualFold(log.Address, token) {
		return gethcommon.Address{}, nil, false, nil
	}
	signature := crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)")).Hex()
	if len(log.Topics) == 0 || !strings.EqualFold(log.Topics[0], signature) {
		return gethcommon.Address{}, nil, false, nil
	}
	if len(log.Topics) != 3 || len(log.Data) != 32 {
		return gethcommon.Address{}, nil, false, fmt.Errorf("USDC migration: malformed ERC20 Transfer log")
	}
	for _, topic := range log.Topics[1:] {
		if len(topic) != 66 || !strings.HasPrefix(topic, "0x") {
			return gethcommon.Address{}, nil, false, fmt.Errorf("USDC migration: malformed indexed address")
		}
		if _, ok := new(big.Int).SetString(topic[2:], 16); !ok || !strings.EqualFold(topic[2:26], strings.Repeat("0", 24)) {
			return gethcommon.Address{}, nil, false, fmt.Errorf("USDC migration: noncanonical indexed address")
		}
	}
	from := gethcommon.BytesToAddress(gethcommon.HexToHash(log.Topics[1]).Bytes())
	to := gethcommon.BytesToAddress(gethcommon.HexToHash(log.Topics[2]).Bytes())
	if to != gethcommon.HexToAddress(safe) {
		return gethcommon.Address{}, nil, false, nil
	}
	return from, new(big.Int).SetBytes(log.Data), true, nil
}
