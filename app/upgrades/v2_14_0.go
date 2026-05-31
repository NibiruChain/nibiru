package upgrades

import (
	"encoding/json"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/app/keepers"
)

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

func runUpgrade2_14_0(nibiru *keepers.PublicKeepers, ctx sdk.Context) error {
	addrCfg := AddrCfg_v2_14

	// -------------------------------------------------------------------------
	// STEP 0: Only run this upgrade logic on mainnet and testnet 2.
	// -------------------------------------------------------------------------
	switch ctx.ChainID() {
	case addrCfg.MainnetChainID, addrCfg.Testnet2ChainID:
	default:
		return nil
	}

	permissionedWasmKeeper := wasmkeeper.NewDefaultPermissionKeeper(nibiru.WasmKeeper)
	updateWasmAdmin := func(contract, currentAdmin, newAdmin sdk.AccAddress) {
		info := nibiru.WasmKeeper.GetContractInfo(ctx, contract)
		if info == nil || info.Admin == "" {
			return
		}

		admin := mustAccAddress(info.Admin)
		if admin.Equals(newAdmin) {
			return
		}
		if !admin.Equals(currentAdmin) {
			return
		}

		_ = permissionedWasmKeeper.UpdateContractAdmin(ctx, contract, currentAdmin, newAdmin)
	}

	// -------------------------------------------------------------------------
	// STEP 1: Skip cleanup if the Treasury CW3 and CW4 contracts are absent.
	// -------------------------------------------------------------------------
	if !nibiru.WasmKeeper.HasContractInfo(ctx, addrCfg.TreasuryCW4Group) {
		return nil
	}
	if !nibiru.WasmKeeper.HasContractInfo(ctx, addrCfg.TreasuryCW3) {
		return nil
	}

	// -------------------------------------------------------------------------
	// STEP 2: Update Treasury CW4 membership with a query-first diff.
	// -------------------------------------------------------------------------
	respBz, err := nibiru.WasmKeeper.QuerySmart(ctx, addrCfg.TreasuryCW4Group, []byte(`{"list_members":{}}`))
	if err != nil {
		return nil
	}

	var membersResp cw4ListMembersResponse
	err = json.Unmarshal(respBz, &membersResp)
	if err != nil {
		return nil
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
			return nil
		}

		_, err = permissionedWasmKeeper.Execute(
			ctx, addrCfg.TreasuryCW4Group, addrCfg.LegacyMultisig, msgBz, sdk.Coins{},
		)
		if err != nil {
			return nil
		}
	}

	// -------------------------------------------------------------------------
	// STEP 3: Hand Treasury CW4 group admin from nibimultisig to Treasury CW3.
	// -------------------------------------------------------------------------
	respBz, err = nibiru.WasmKeeper.QuerySmart(ctx, addrCfg.TreasuryCW4Group, []byte(`{"admin":{}}`))
	if err != nil {
		return nil
	}

	var adminResp cw4AdminResponse
	err = json.Unmarshal(respBz, &adminResp)
	if err != nil {
		return nil
	}

	switch adminResp.Admin {
	case addrCfg.TreasuryCW3.String():
	case addrCfg.LegacyMultisig.String():
		msg := cw4UpdateAdminMsg{}
		msg.UpdateAdmin.Admin = addrCfg.TreasuryCW3.String()
		msgBz, err := json.Marshal(msg)
		if err != nil {
			return nil
		}

		_, err = permissionedWasmKeeper.Execute(
			ctx, addrCfg.TreasuryCW4Group, addrCfg.LegacyMultisig, msgBz, sdk.Coins{},
		)
		if err != nil {
			return nil
		}
	default:
		return nil
	}

	// -------------------------------------------------------------------------
	// STEP 4: Move Treasury CW4 and CW3 wasm admin metadata to Treasury CW3.
	// -------------------------------------------------------------------------
	updateWasmAdmin(addrCfg.TreasuryCW4Group, addrCfg.LegacyMultisig, addrCfg.TreasuryCW3)
	updateWasmAdmin(addrCfg.TreasuryCW3, addrCfg.LegacyMultisig, addrCfg.TreasuryCW3)

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
			return nil
		}
	}

	// -------------------------------------------------------------------------
	// STEP 7: Move Hot Wallet CW3 and CW4 wasm admin metadata to Treasury CW3.
	// -------------------------------------------------------------------------
	updateWasmAdmin(addrCfg.HotWalletCW3, addrCfg.KevinNanoS, addrCfg.TreasuryCW3)

	if nibiru.WasmKeeper.HasContractInfo(ctx, addrCfg.HotWalletCW4Group) {
		updateWasmAdmin(addrCfg.HotWalletCW4Group, addrCfg.KevinNanoS, addrCfg.TreasuryCW3)
	}

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
