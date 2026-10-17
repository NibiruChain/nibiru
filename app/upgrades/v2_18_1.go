package upgrades

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	gethabi "github.com/ethereum/go-ethereum/accounts/abi"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	gethcore "github.com/ethereum/go-ethereum/core/types"

	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/module"
	upgradetypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/upgrade/types"

	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/app/keepers"
	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/evm"
	"github.com/NibiruChain/nibiru/v2/evm/embeds"
	"github.com/NibiruChain/nibiru/v2/evm/evmstate"
)

const (
	// IncidentRecoveryChainID_v2_18 limits the forced recovery to mainnet.
	IncidentRecoveryChainID_v2_18 = appconst.SDK_CHAIN_ID_MAINNET
	// IncidentRecoveryCW3_v2_18 receives all recovered Bank coins.
	IncidentRecoveryCW3_v2_18 = "nibi1l8dxzwz9d4peazcqjclnkj2mhvtj7mpnkqx85mg0ndrlhwrnh7gskkzg0v"
	// IncidentRecoverySafe_v2_18 receives all recovered ERC20 tokens.
	IncidentRecoverySafe_v2_18 = "0x22CBd7CbF3b33681abB3Ced4D64d71acB9a9dCd2"
)

var (
	// IncidentRecoverySources_v2_18 lists the three incident-controlled EVM accounts.
	IncidentRecoverySources_v2_18 = []string{
		"0x947Be8Ce20F2b6deFC3ed8593aaebF5E9974841B",
		"0x3E0D2aCd91F72FeD140d2b9FF5C7005D60b87972",
		"0x761ff39a47ab53ba07f66971015d656c37391517",
	}
	// IncidentRecoveryERC20s_v2_18 lists every ERC20 identified by the incident review.
	IncidentRecoveryERC20s_v2_18 = []string{
		"0x0829F361A05D993d5CEb035cA6DF3446b060970b", // USDC.e
		"0xcdA5b77E2E2268D9E09c874c1b9A4c3F07b37555", // WETH
		"0xcA0a9Fb5FBF692fa12fD13c0A900EC56Bb3f0a7b", // stNIBI
		"0xf4e097E36d2064E2bDCA96e60439f3A369522003", // USDA
		"0x84f682626302EA7BCA2A7c338b84863292131319", // sUSDa
		"0x43F2376D5D03553aE72F4A8093bbe9de4336EB08", // USDT
	}
)

var _ HandlerImpl = (*Handler_v2_18)(nil)

type RecoveryConfigV218 struct {
	ChainID               string
	Sources               []gethcommon.Address
	ERC20s                []gethcommon.Address
	BankTo                sdk.AccAddress
	ERC20To               gethcommon.Address
	AssertIncidentSources bool
}

type Handler_v2_18 struct {
	// Recovery overrides production recovery addresses in focused tests.
	Recovery *RecoveryConfigV218
}

func (h Handler_v2_18) Handler(
	mm *module.Manager,
	cfg module.Configurator,
	nibiru *keepers.PublicKeepers,
) upgradetypes.UpgradeHandler {
	return func(
		ctx sdk.Context,
		plan upgradetypes.Plan,
		fromVM module.VersionMap,
	) (module.VersionMap, error) {
		if err := h.runUpgrade2_18_1(nibiru, ctx); err != nil {
			ctx.Logger().Error("v2.18.1 upgrade failure", "err", err)
			ctx.EventManager().EmitEvent(NewEventUpgradeFailure("v2.18.1", err))
		}
		return mm.RunMigrations(ctx, cfg, fromVM)
	}
}

func (h Handler_v2_18) runUpgrade2_18_1(
	nibiru *keepers.PublicKeepers,
	ctx sdk.Context,
) error {
	recoveryCfg, err := h.config()
	if err != nil {
		return fmt.Errorf("recovery configuration: %w", err)
	}
	if ctx.ChainID() != recoveryCfg.ChainID {
		return nil
	}
	return recoverIncidentFundsV218(nibiru, ctx, recoveryCfg)
}

func (h Handler_v2_18) config() (RecoveryConfigV218, error) {
	if h.Recovery != nil {
		return *h.Recovery, validateRecoveryConfigV218(*h.Recovery)
	}

	bankTo, err := sdk.AccAddressFromBech32(IncidentRecoveryCW3_v2_18)
	if err != nil {
		return RecoveryConfigV218{}, err
	}
	cfg := RecoveryConfigV218{
		ChainID:               IncidentRecoveryChainID_v2_18,
		Sources:               hexAddressesV218(IncidentRecoverySources_v2_18),
		ERC20s:                hexAddressesV218(IncidentRecoveryERC20s_v2_18),
		BankTo:                bankTo,
		ERC20To:               gethcommon.HexToAddress(IncidentRecoverySafe_v2_18),
		AssertIncidentSources: true,
	}
	return cfg, validateRecoveryConfigV218(cfg)
}

func hexAddressesV218(addresses []string) []gethcommon.Address {
	out := make([]gethcommon.Address, len(addresses))
	for idx, address := range addresses {
		out[idx] = gethcommon.HexToAddress(address)
	}
	return out
}

func validateRecoveryConfigV218(cfg RecoveryConfigV218) error {
	if cfg.ChainID == "" {
		return fmt.Errorf("empty chain ID")
	}
	if len(cfg.Sources) == 0 || len(cfg.ERC20s) == 0 {
		return fmt.Errorf("sources and ERC20 contracts must not be empty")
	}
	if len(cfg.BankTo) == 0 || cfg.ERC20To == (gethcommon.Address{}) {
		return fmt.Errorf("recovery destinations must not be empty")
	}
	for _, source := range cfg.Sources {
		if source == (gethcommon.Address{}) {
			return fmt.Errorf("empty source address")
		}
	}
	for _, token := range cfg.ERC20s {
		if token == (gethcommon.Address{}) {
			return fmt.Errorf("empty ERC20 address")
		}
	}
	return nil
}

func recoverIncidentFundsV218(
	nibiru *keepers.PublicKeepers,
	ctx sdk.Context,
	cfg RecoveryConfigV218,
) error {
	if cfg.AssertIncidentSources {
		if err := assertDocumentedSourceMappingsV218(cfg.Sources); err != nil {
			return err
		}
	}

	var recoveryErrors []error

	// Bank transfers are simpler than EVM calls, so finish them first. Each
	// source is independent: an unexpected failure must not discard recovery
	// work that already completed or block the remaining sources.
	for _, source := range cfg.Sources {
		bankSource := eth.EthAddrToNibiruAddr(source)
		if err := recoverBankV218(nibiru, ctx, bankSource, cfg.BankTo); err != nil {
			recoveryErrors = append(recoveryErrors, err)
			ctx.EventManager().EmitEvent(newRecoveryFailureEventV218(
				"bank", bankSource.String(), cfg.BankTo.String(), "bank", err,
			))
		}
	}

	for _, source := range cfg.Sources {
		for _, token := range cfg.ERC20s {
			sdb := nibiru.EvmKeeper.NewSDB(
				ctx,
				nibiru.EvmKeeper.TxConfig(ctx, gethcommon.Hash{}),
			)
			if err := recoverERC20V218(nibiru.EvmKeeper, sdb, source, cfg.ERC20To, token); err != nil {
				err = fmt.Errorf("source %s token %s: %w", source.Hex(), token.Hex(), err)
				recoveryErrors = append(recoveryErrors, err)
				ctx.EventManager().EmitEvent(newRecoveryFailureEventV218(
					"erc20", source.Hex(), cfg.ERC20To.Hex(), token.Hex(), err,
				))
				continue
			}
			sdb.Commit()
		}
	}

	return errors.Join(recoveryErrors...)
}

func recoverBankV218(
	nibiru *keepers.PublicKeepers,
	ctx sdk.Context,
	source, destination sdk.AccAddress,
) error {
	allBalances := nibiru.BankKeeper.GetAllBalances(ctx, source)
	spendable := nibiru.BankKeeper.SpendableCoins(ctx, source)
	if !allBalances.IsEqual(spendable) {
		return fmt.Errorf("source %s has locked Bank coins: all=%s spendable=%s", source, allBalances, spendable)
	}
	if allBalances.Empty() {
		return nil
	}
	if err := nibiru.BankKeeper.SendCoins(ctx, source, destination, allBalances); err != nil {
		return fmt.Errorf("Bank transfer from %s: %w", source, err)
	}
	if remaining := nibiru.BankKeeper.GetAllBalances(ctx, source); !remaining.Empty() {
		return fmt.Errorf("source %s retains Bank coins: %s", source, remaining)
	}
	ctx.EventManager().EmitEvent(newRecoveryEventV218(
		source.String(), destination.String(), "bank", allBalances.String(),
	))
	return nil
}

func assertDocumentedSourceMappingsV218(sources []gethcommon.Address) error {
	if len(sources) < 2 {
		return nil
	}
	expected := []string{
		"nibi1j3a73n3q72mdalp7mpvn4t4lt6vhfpqm00fup8",
		"nibi18cxj4nv37uh769qd9w0lt3cqt4sts7tjnchmmw",
	}
	for idx, expectedBech32 := range expected {
		if actual := eth.EthAddrToNibiruAddr(sources[idx]).String(); actual != expectedBech32 {
			return fmt.Errorf("source %s maps to %s, expected %s", sources[idx], actual, expectedBech32)
		}
	}
	return nil
}

func recoverERC20V218(
	keeper *evmstate.Keeper,
	sdb *evmstate.SDB,
	source gethcommon.Address,
	destination gethcommon.Address,
	token gethcommon.Address,
) error {
	ctx := sdb.Ctx()
	abi := embeds.SmartContract_ERC20MinterWithMetadataUpdates.ABI

	sourceBefore, err := loadERC20ValueV218(keeper, sdb, abi, token, "balanceOf", source)
	if err != nil {
		return fmt.Errorf("read source balance: %w", err)
	}
	if sourceBefore.Sign() == 0 {
		return nil
	}
	destinationBefore, err := loadERC20ValueV218(keeper, sdb, abi, token, "balanceOf", destination)
	if err != nil {
		return fmt.Errorf("read destination balance: %w", err)
	}
	totalSupplyBefore, err := loadERC20ValueV218(keeper, sdb, abi, token, "totalSupply")
	if err != nil {
		return fmt.Errorf("read total supply: %w", err)
	}

	input, err := abi.Pack("transfer", destination, sourceBefore)
	if err != nil {
		return fmt.Errorf("pack transfer: %w", err)
	}
	zero := new(big.Int)
	sourceNonce := sdb.GetNonce(source)
	msg := core.Message{
		To:               &token,
		From:             source,
		Nonce:            sourceNonce,
		Value:            zero,
		GasLimit:         evm.Erc20GasLimitExecute,
		GasPrice:         zero,
		GasFeeCap:        zero,
		GasTipCap:        zero,
		Data:             input,
		AccessList:       gethcore.AccessList{},
		BlobGasFeeCap:    new(big.Int),
		BlobHashes:       []gethcommon.Hash{},
		SkipNonceChecks:  true,
		SkipFromEOACheck: true,
	}
	// This app-internal state-migration call does not pass through public
	// Ethereum transaction validation. The skip flags make the contract-sender
	// intent explicit without changing the public transaction path.
	evmObj := keeper.NewEVM(ctx, msg, keeper.GetEVMConfig(ctx), nil, sdb)
	resp, err := keeper.ApplyEvmMsg(msg, evmObj, evm.COMMIT_READONLY)
	if err != nil {
		return fmt.Errorf("execute transfer: %w", err)
	}
	if resp.Failed() {
		return fmt.Errorf("transfer VM error: %s", resp.VmError)
	}
	var transferResult evmstate.ERC20Bool
	if err := abi.UnpackIntoInterface(&transferResult, "transfer", resp.Ret); err != nil {
		return fmt.Errorf("decode transfer result: %w", err)
	}
	if !transferResult.Value {
		return fmt.Errorf("transfer returned false")
	}
	// Internal recovery calls must not consume an attacker account nonce.
	sdb.SetNonce(source, sourceNonce)

	sourceAfter, err := loadERC20ValueV218(keeper, sdb, abi, token, "balanceOf", source)
	if err != nil {
		return fmt.Errorf("read final source balance: %w", err)
	}
	destinationAfter, err := loadERC20ValueV218(keeper, sdb, abi, token, "balanceOf", destination)
	if err != nil {
		return fmt.Errorf("read final destination balance: %w", err)
	}
	totalSupplyAfter, err := loadERC20ValueV218(keeper, sdb, abi, token, "totalSupply")
	if err != nil {
		return fmt.Errorf("read final total supply: %w", err)
	}
	if sourceAfter.Sign() != 0 {
		return fmt.Errorf("source retains %s tokens", sourceAfter)
	}
	if increase := new(big.Int).Sub(destinationAfter, destinationBefore); increase.Cmp(sourceBefore) != 0 {
		return fmt.Errorf("destination increase %s does not equal source balance %s", increase, sourceBefore)
	}
	if totalSupplyAfter.Cmp(totalSupplyBefore) != 0 {
		return fmt.Errorf("total supply changed from %s to %s", totalSupplyBefore, totalSupplyAfter)
	}

	sdb.Ctx().EventManager().EmitEvent(newRecoveryEventV218(
		source.Hex(), destination.Hex(), token.Hex(), sourceBefore.String(),
	))
	return nil
}

func loadERC20ValueV218(
	keeper *evmstate.Keeper,
	sdb *evmstate.SDB,
	abi *gethabi.ABI,
	token gethcommon.Address,
	method string,
	args ...any,
) (*big.Int, error) {
	input, err := abi.Pack(method, args...)
	if err != nil {
		return nil, err
	}
	zero := new(big.Int)
	readonlyNonce := sdb.GetNonce(evm.EVM_READONLY_ADDR)
	defer sdb.SetNonce(evm.EVM_READONLY_ADDR, readonlyNonce)
	msg := core.Message{
		To:               &token,
		From:             evm.EVM_READONLY_ADDR,
		Nonce:            readonlyNonce,
		Value:            zero,
		GasLimit:         evm.Erc20GasLimitQuery,
		GasPrice:         zero,
		GasFeeCap:        zero,
		GasTipCap:        zero,
		Data:             input,
		AccessList:       gethcore.AccessList{},
		BlobGasFeeCap:    new(big.Int),
		BlobHashes:       []gethcommon.Hash{},
		SkipNonceChecks:  true,
		SkipFromEOACheck: true,
	}
	evmObj := keeper.NewEVM(sdb.Ctx(), msg, keeper.GetEVMConfig(sdb.Ctx()), nil, sdb)
	resp, err := keeper.ApplyEvmMsg(msg, evmObj, evm.COMMIT_READONLY)
	if err != nil {
		return nil, err
	}
	if resp.Failed() {
		return nil, fmt.Errorf("%s VM error: %s", method, resp.VmError)
	}
	var result evmstate.ERC20BigInt
	if err := abi.UnpackIntoInterface(&result, method, resp.Ret); err != nil {
		return nil, err
	}
	return result.Value, nil
}

func newRecoveryEventV218(source, destination, asset, amount string) sdk.Event {
	return sdk.NewEvent(
		"incident_fund_recovery",
		sdk.NewAttribute("upgrade", "v2.18.1"),
		sdk.NewAttribute("source", source),
		sdk.NewAttribute("destination", destination),
		sdk.NewAttribute("asset", asset),
		sdk.NewAttribute("amount", amount),
	)
}

type recoveryFailureV218 struct {
	Operation   string `json:"operation"`
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Asset       string `json:"asset"`
	Error       string `json:"error"`
}

func newRecoveryFailureEventV218(
	operation, source, destination, asset string,
	err error,
) sdk.Event {
	details, marshalErr := json.Marshal(recoveryFailureV218{
		Operation:   operation,
		Source:      source,
		Destination: destination,
		Asset:       asset,
		Error:       err.Error(),
	})
	if marshalErr != nil {
		details = []byte(fmt.Sprintf(`{"error":%q}`, marshalErr.Error()))
	}
	return sdk.NewEvent(
		"incident_fund_recovery_failure",
		sdk.NewAttribute("upgrade", "v2.18.1"),
		sdk.NewAttribute("details", string(details)),
	)
}
