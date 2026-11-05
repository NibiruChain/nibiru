package upgrades

import (
	"bytes"
	"fmt"
	"math/big"
	"strconv"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	gethcore "github.com/ethereum/go-ethereum/core/types"

	"github.com/NibiruChain/nibiru/v2/app/keepers"
	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/evm"
	"github.com/NibiruChain/nibiru/v2/evm/embeds"
	"github.com/NibiruChain/nibiru/v2/evm/evmstate"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	query "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/query"
	banktypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/bank/types"
)

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
