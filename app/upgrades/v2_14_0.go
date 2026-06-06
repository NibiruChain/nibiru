package upgrades

import (
	"encoding/json"
	"fmt"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/app/keepers"
	"github.com/NibiruChain/nibiru/v2/x/sudo"
)

var _ HandlerImpl = (*Handler_v2_14)(nil)

type Handler_v2_14 struct{}

func (h Handler_v2_14) Handler(
	mm *module.Manager,
	cfg module.Configurator,
	nibiru *keepers.PublicKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		err := h.runUpgrade2_14_0(nibiru, ctx)
		if err != nil {
			ctx.Logger().Error("v2.14.0 upgrade failure", "err", err)
			ctx.EventManager().EmitEvent(
				NewEventUpgradeFailure("v2.14.0", err),
			)
		}
		return mm.RunMigrations(ctx, cfg, fromVM)
	}
}

// Upgrade2_14_AddrCfg contains the deployed contract addresses touched
// by the v2.14 upgrade. Tests can override these values after instantiating
// real Wasm contracts, while production keeps the mainnet/testnet addresses.
type Upgrade2_14_AddrCfg struct {
	MainnetChainID  string
	Testnet2ChainID string

	TreasuryCW3      sdk.AccAddress
	TreasuryCW4Group sdk.AccAddress
	LegacyMultisig   sdk.AccAddress

	HotWalletCW3      sdk.AccAddress
	HotWalletCW4Group sdk.AccAddress
	KevinNanoS        sdk.AccAddress

	TreasuryAddSigner     sdk.AccAddress
	TreasuryRemoveSigners []sdk.AccAddress

	LayerZeroOFTAdapters []string
}

// AddrCfg_v2_14 is the deployed address configuration used by the v2.14
// upgrade handler. Tests temporarily override contract fields after
// instantiating real Wasm contracts at fresh test addresses.
var AddrCfg_v2_14 = Upgrade2_14_AddrCfg{
	MainnetChainID:  "cataclysm-1",
	Testnet2ChainID: "nibiru-testnet-2",

	TreasuryCW3:      mustAccAddress("nibi1l8dxzwz9d4peazcqjclnkj2mhvtj7mpnkqx85mg0ndrlhwrnh7gskkzg0v"),
	TreasuryCW4Group: mustAccAddress("nibi1zvwvtpluyak4x7u2yf5j3qxqgvmnfdgn9y7dphthleq4sylrsaesmm9dnz"),
	LegacyMultisig:   mustAccAddress("nibi1xfdlvvmx03nyav4840z25m33qrwzy67p4eadn7"),

	HotWalletCW3:      mustAccAddress("nibi15wd4ac2383fq65uymu72dg4u4u60t2du545fzxeakdw3kf7hd7yqtg45z7"),
	HotWalletCW4Group: mustAccAddress("nibi1pqh0j0jzasj4f7mfm7hp6dq94fcuemrl9afas4k48pw0x3j8gxysdsy4n2"),
	KevinNanoS:        mustAccAddress("nibi1w3s6gdtt09ekhmwzfszc6qtxh298fstu895gn8"),

	TreasuryAddSigner: mustAccAddress("nibi1cj6edencz3tetxuwkxcdrlfwpg8cr02ef3uvz2"),
	TreasuryRemoveSigners: []sdk.AccAddress{
		mustAccAddress("nibi1wwhdx03msygelmm8tm5z6nzh4dklwkettwd5vj"), // Gimeno
		mustAccAddress("nibi1aj6vgnj5hh0ehe5memz0f38z2lla2z5gdj5vst"), // JC
		mustAccAddress("nibi1w3s6gdtt09ekhmwzfszc6qtxh298fstu895gn8"), // Kevin NanoS
	},

	LayerZeroOFTAdapters: []string{
		"0x12a272A581feE5577A5dFa371afEB4b2F3a8C2F8", // USDC.e LayerZero OFT adapter
		"0x4DF4eFa0a3707b6Cc964F62042E8a303A0376F54", // WNIBI LayerZero OFT adapter
	},
}

type cw4Member struct {
	Addr   string `json:"addr"`
	Weight uint64 `json:"weight"`
}

type cw4ListMembersResponse struct {
	Members []cw4Member `json:"members"`
}

type cw4AdminResponse struct {
	Admin string `json:"admin"`
}

type cw4UpdateMembersMsg struct {
	UpdateMembers struct {
		Add    []cw4Member `json:"add"`
		Remove []string    `json:"remove"`
	} `json:"update_members"`
}

type cw4UpdateAdminMsg struct {
	UpdateAdmin struct {
		Admin string `json:"admin"`
	} `json:"update_admin"`
}

func (h Handler_v2_14) runUpgrade2_14_0(nibiru *keepers.PublicKeepers, ctx sdk.Context) error {
	addrCfg := AddrCfg_v2_14

	// -------------------------------------------------------------------------
	// STEP 0: Only run this upgrade logic on mainnet and testnet 2.
	// -------------------------------------------------------------------------
	switch ctx.ChainID() {
	case addrCfg.MainnetChainID, addrCfg.Testnet2ChainID:
	default:
		return nil
	}

	// -------------------------------------------------------------------------
	// STEP 1: Mainnet-only zero-gas allowlist update for LayerZero OFT adapters.
	// -------------------------------------------------------------------------
	if ctx.ChainID() == addrCfg.MainnetChainID {
		if err := h.addZeroGasContracts(ctx, nibiru, addrCfg.LayerZeroOFTAdapters); err != nil {
			return err
		}
	}

	permissionedWasmKeeper := wasmkeeper.NewDefaultPermissionKeeper(nibiru.WasmKeeper)
	updateWasmAdmin := func(contract, currentAdmin, newAdmin sdk.AccAddress) error {
		info := nibiru.WasmKeeper.GetContractInfo(ctx, contract)
		if info == nil || info.Admin == "" {
			return fmt.Errorf("contract info not found or admin empty for %s", contract.String())
		}

		admin, err := sdk.AccAddressFromBech32(info.Admin)
		if err != nil {
			return fmt.Errorf("failed to decode wasm admin for %s: %w", contract.String(), err)
		}
		if admin.Equals(newAdmin) {
			return nil
		}
		if !admin.Equals(currentAdmin) {
			return fmt.Errorf("current admin mismatch for %s: expected %s, actual %s", contract.String(), currentAdmin.String(), admin.String())
		}

		err = permissionedWasmKeeper.UpdateContractAdmin(ctx, contract, currentAdmin, newAdmin)
		if err != nil {
			return fmt.Errorf("failed to update contract admin for %s: %w", contract.String(), err)
		}
		return nil
	}

	// -------------------------------------------------------------------------
	// STEP 2: Update Treasury CW4 membership with a query-first diff.
	// -------------------------------------------------------------------------
	if !nibiru.WasmKeeper.HasContractInfo(ctx, addrCfg.TreasuryCW4Group) {
		return fmt.Errorf("Step 2: Skip CW4 group update because contract does not exist for addr %s", addrCfg.TreasuryCW4Group.String())
	}

	respBz, err := nibiru.WasmKeeper.QuerySmart(ctx, addrCfg.TreasuryCW4Group, []byte(`{"list_members":{}}`))
	if err != nil {
		return fmt.Errorf("failed to query Treasury CW4 Group members: %w", err)
	}

	var membersResp cw4ListMembersResponse
	err = json.Unmarshal(respBz, &membersResp)
	if err != nil {
		return fmt.Errorf("failed to unmarshal Treasury CW4 Group members response: %w", err)
	}

	currentMembers := make(map[string]bool, len(membersResp.Members))
	for _, member := range membersResp.Members {
		currentMembers[member.Addr] = true
	}

	removeMembers := []string{}
	for _, addr := range addrCfg.TreasuryRemoveSigners {
		if currentMembers[addr.String()] {
			removeMembers = append(removeMembers, addr.String())
		}
	}

	addMembers := []cw4Member{}
	if !currentMembers[addrCfg.TreasuryAddSigner.String()] {
		addMembers = append(addMembers, cw4Member{
			Addr:   addrCfg.TreasuryAddSigner.String(),
			Weight: 1,
		})
	}

	if len(addMembers) > 0 || len(removeMembers) > 0 {
		msg := cw4UpdateMembersMsg{}
		msg.UpdateMembers.Add = addMembers
		msg.UpdateMembers.Remove = removeMembers
		msgBz, err := json.Marshal(msg)
		if err != nil {
			return fmt.Errorf("failed to marshal Treasury CW4 Group update members message: %w", err)
		}

		_, err = permissionedWasmKeeper.Execute(
			ctx, addrCfg.TreasuryCW4Group, addrCfg.LegacyMultisig, msgBz, sdk.Coins{},
		)
		if err != nil {
			return fmt.Errorf("failed to execute Treasury CW4 Group update members: %w", err)
		}
	}

	// -------------------------------------------------------------------------
	// STEP 3: Hand Treasury CW4 group admin from nibimultisig to Treasury CW3.
	// -------------------------------------------------------------------------
	if !nibiru.WasmKeeper.HasContractInfo(ctx, addrCfg.TreasuryCW3) {
		return fmt.Errorf("Step 3: Skip Treasury CW3 update because contract does not exist for addr %s", addrCfg.TreasuryCW3.String())
	}

	respBz, err = nibiru.WasmKeeper.QuerySmart(ctx, addrCfg.TreasuryCW4Group, []byte(`{"admin":{}}`))
	if err != nil {
		return fmt.Errorf("failed to query Treasury CW4 Group admin: %w", err)
	}

	var adminResp cw4AdminResponse
	err = json.Unmarshal(respBz, &adminResp)
	if err != nil {
		return fmt.Errorf("failed to unmarshal Treasury CW4 Group admin response: %w", err)
	}

	switch adminResp.Admin {
	case addrCfg.TreasuryCW3.String():
	case addrCfg.LegacyMultisig.String():
		msg := cw4UpdateAdminMsg{}
		msg.UpdateAdmin.Admin = addrCfg.TreasuryCW3.String()
		msgBz, err := json.Marshal(msg)
		if err != nil {
			return fmt.Errorf("failed to marshal Treasury CW4 Group update admin message: %w", err)
		}

		_, err = permissionedWasmKeeper.Execute(
			ctx, addrCfg.TreasuryCW4Group, addrCfg.LegacyMultisig, msgBz, sdk.Coins{},
		)
		if err != nil {
			return fmt.Errorf("failed to execute Treasury CW4 Group update admin: %w", err)
		}
	default:
		return fmt.Errorf("unexpected Treasury CW4 Group admin: %s", adminResp.Admin)
	}

	// -------------------------------------------------------------------------
	// STEP 4: Move Treasury CW4 and CW3 wasm admin metadata to Treasury CW3.
	// -------------------------------------------------------------------------
	if err := updateWasmAdmin(addrCfg.TreasuryCW4Group, addrCfg.LegacyMultisig, addrCfg.TreasuryCW3); err != nil {
		return err
	}
	if err := updateWasmAdmin(addrCfg.TreasuryCW3, addrCfg.LegacyMultisig, addrCfg.TreasuryCW3); err != nil {
		return err
	}

	// -------------------------------------------------------------------------
	// STEP 5: Mainnet-only guard for dormant Hot Wallet cleanup.
	// -------------------------------------------------------------------------
	if ctx.ChainID() != addrCfg.MainnetChainID {
		return nil
	}
	if !nibiru.WasmKeeper.HasContractInfo(ctx, addrCfg.HotWalletCW3) {
		return nil
	}

	// -------------------------------------------------------------------------
	// STEP 6: Sweep the Hot Wallet CW3 bank balance into Treasury CW3.
	// -------------------------------------------------------------------------
	balances := nibiru.BankKeeper.GetAllBalances(ctx, addrCfg.HotWalletCW3)
	if !balances.IsZero() {
		err = nibiru.BankKeeper.SendCoins(ctx, addrCfg.HotWalletCW3, addrCfg.TreasuryCW3, balances)
		if err != nil {
			return fmt.Errorf("failed to sweep Hot Wallet CW3 balance: %w", err)
		}
	}

	// -------------------------------------------------------------------------
	// STEP 7: Move Hot Wallet CW3 and CW4 wasm admin metadata to Treasury CW3.
	// -------------------------------------------------------------------------
	if err := updateWasmAdmin(addrCfg.HotWalletCW3, addrCfg.KevinNanoS, addrCfg.TreasuryCW3); err != nil {
		return err
	}

	if nibiru.WasmKeeper.HasContractInfo(ctx, addrCfg.HotWalletCW4Group) {
		if err := updateWasmAdmin(addrCfg.HotWalletCW4Group, addrCfg.KevinNanoS, addrCfg.TreasuryCW3); err != nil {
			return err
		}
	}

	return nil
}

func (h Handler_v2_14) addZeroGasContracts(ctx sdk.Context, nibiru *keepers.PublicKeepers, addrs []string) error {
	actors := nibiru.SudoKeeper.GetZeroGasActors(ctx)

	nextActors := sudo.ZeroGasActors{
		Senders:                append([]string(nil), actors.Senders...),
		Contracts:              append([]string(nil), actors.Contracts...),
		AlwaysZeroGasContracts: make([]string, 0, len(actors.AlwaysZeroGasContracts)+len(addrs)),
	}
	seenAlwaysZeroGas := make(map[string]bool, len(actors.AlwaysZeroGasContracts)+len(addrs))
	for _, addr := range actors.AlwaysZeroGasContracts {
		if seenAlwaysZeroGas[addr] {
			continue
		}
		nextActors.AlwaysZeroGasContracts = append(nextActors.AlwaysZeroGasContracts, addr)
		seenAlwaysZeroGas[addr] = true
	}
	for _, addr := range addrs {
		if seenAlwaysZeroGas[addr] {
			continue
		}
		nextActors.AlwaysZeroGasContracts = append(nextActors.AlwaysZeroGasContracts, addr)
		seenAlwaysZeroGas[addr] = true
	}

	if err := nextActors.Validate(); err != nil {
		return fmt.Errorf("failed to validate zero-gas LayerZero OFT adapter state: %w", err)
	}
	nibiru.SudoKeeper.ZeroGasActors.Set(ctx, nextActors)
	return nil
}

// mustAccAddress decodes Nibiru account literals without relying on the SDK
// global bech32 config. AddrCfg_v2_14 is initialized at package load time, which
// can happen before app setup calls SetPrefixes and changes the default account
// prefix from "cosmos" to "nibi".
func mustAccAddress(addr string) sdk.AccAddress {
	bz, err := sdk.GetFromBech32(addr, appconst.AccountAddressPrefix)
	if err != nil {
		panic(err)
	}
	if err := sdk.VerifyAddressFormat(bz); err != nil {
		panic(err)
	}
	return sdk.AccAddress(bz)
}
