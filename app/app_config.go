package app

import (
	runtimev1alpha1 "cosmossdk.io/api/cosmos/app/runtime/v1alpha1"
	appv1alpha1 "cosmossdk.io/api/cosmos/app/v1alpha1"
	authmodulev1 "cosmossdk.io/api/cosmos/auth/module/v1"
	authzmodulev1 "cosmossdk.io/api/cosmos/authz/module/v1"
	bankmodulev1 "cosmossdk.io/api/cosmos/bank/module/v1"
	capabilitymodulev1 "cosmossdk.io/api/cosmos/capability/module/v1"
	consensusmodulev1 "cosmossdk.io/api/cosmos/consensus/module/v1"
	crisismodulev1 "cosmossdk.io/api/cosmos/crisis/module/v1"
	distrmodulev1 "cosmossdk.io/api/cosmos/distribution/module/v1"
	evidencemodulev1 "cosmossdk.io/api/cosmos/evidence/module/v1"
	feegrantmodulev1 "cosmossdk.io/api/cosmos/feegrant/module/v1"
	genutilmodulev1 "cosmossdk.io/api/cosmos/genutil/module/v1"
	govmodulev1 "cosmossdk.io/api/cosmos/gov/module/v1"
	paramsmodulev1 "cosmossdk.io/api/cosmos/params/module/v1"
	slashingmodulev1 "cosmossdk.io/api/cosmos/slashing/module/v1"
	stakingmodulev1 "cosmossdk.io/api/cosmos/staking/module/v1"
	txconfigv1 "cosmossdk.io/api/cosmos/tx/config/v1"
	upgrademodulev1 "cosmossdk.io/api/cosmos/upgrade/module/v1"
	"cosmossdk.io/core/appconfig"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/NibiruChain/nibiru/v2/x/common"
	epochstypes "github.com/NibiruChain/nibiru/v2/x/epochs/types"
	"github.com/NibiruChain/nibiru/v2/x/evm"
	evmtypes "github.com/NibiruChain/nibiru/v2/x/evm"
	inflationtypes "github.com/NibiruChain/nibiru/v2/x/inflation/types"
	oracletypes "github.com/NibiruChain/nibiru/v2/x/oracle/types"
	sudotypes "github.com/NibiruChain/nibiru/v2/x/sudo/types"
	tokenfactorytypes "github.com/NibiruChain/nibiru/v2/x/tokenfactory/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	consensustypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	evidencetypes "github.com/cosmos/cosmos-sdk/x/evidence/types"
	"github.com/cosmos/cosmos-sdk/x/feegrant"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	ibcwasmtypes "github.com/cosmos/ibc-go/modules/light-clients/08-wasm/types"
	icatypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/types"
	ibcfeetypes "github.com/cosmos/ibc-go/v7/modules/apps/29-fee/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	ibcexported "github.com/cosmos/ibc-go/v7/modules/core/exported"

	evmmodulev1 "github.com/NibiruChain/nibiru/v2/api/eth/evm/module"
	epochsmodulev1 "github.com/NibiruChain/nibiru/v2/api/nibiru/epochs/module"
	inflationmodulev1 "github.com/NibiruChain/nibiru/v2/api/nibiru/inflation/module"
	oraclemodulev1 "github.com/NibiruChain/nibiru/v2/api/nibiru/oracle/module"
	sudomodulev1 "github.com/NibiruChain/nibiru/v2/api/nibiru/sudo/module"
	tokenfactorymodulev1 "github.com/NibiruChain/nibiru/v2/api/nibiru/tokenfactory/module"
)

var (
	blockAccAddrs = func(m map[string]bool) []string {
		k := make([]string, 0, len(m))
		for key := range m {
			k = append(k, key)
		}
		return k
	}(BlockedAddresses())

	genesisModuleOrder = []string{
		capabilitytypes.ModuleName, authtypes.ModuleName, banktypes.ModuleName,
		distrtypes.ModuleName, stakingtypes.ModuleName, slashingtypes.ModuleName, govtypes.ModuleName,
		crisistypes.ModuleName, genutiltypes.ModuleName, evidencetypes.ModuleName, authz.ModuleName,
		feegrant.ModuleName, paramstypes.ModuleName, upgradetypes.ModuleName,
		consensustypes.ModuleName, ibctransfertypes.ModuleName, ibcexported.ModuleName,
		ibcfeetypes.ModuleName, icatypes.ModuleName, ibcwasmtypes.ModuleName,
		// devgastypes.ModuleName,
		tokenfactorytypes.ModuleName, epochstypes.ModuleName, oracletypes.ModuleName,
		inflationtypes.ModuleName, sudotypes.ModuleName, evmtypes.ModuleName,
	}

	// module account permissions
	moduleAccPerms = []*authmodulev1.ModuleAccountPermission{
		{Account: authtypes.FeeCollectorName},
		{Account: distrtypes.ModuleName},
		{Account: inflationtypes.ModuleName, Permissions: []string{authtypes.Minter, authtypes.Burner}},
		{Account: stakingtypes.BondedPoolName, Permissions: []string{authtypes.Burner, stakingtypes.ModuleName}},
		{Account: stakingtypes.NotBondedPoolName, Permissions: []string{authtypes.Burner, stakingtypes.ModuleName}},
		{Account: govtypes.ModuleName, Permissions: []string{authtypes.Burner}},
		{Account: oracletypes.ModuleName},
		{Account: ibctransfertypes.ModuleName, Permissions: []string{authtypes.Minter, authtypes.Burner}},
		{Account: ibcfeetypes.ModuleName},
		{Account: icatypes.ModuleName},

		{Account: evm.ModuleName, Permissions: []string{authtypes.Minter, authtypes.Burner}},
		{Account: epochstypes.ModuleName},
		{Account: sudotypes.ModuleName},
		{Account: common.TreasuryPoolModuleAccount},
		{Account: wasmtypes.ModuleName, Permissions: []string{authtypes.Burner}},
		{Account: tokenfactorytypes.ModuleName, Permissions: []string{authtypes.Minter, authtypes.Burner}},
	}

	// application configuration (used by depinject)
	AppConfig = appconfig.Compose(&appv1alpha1.Config{
		Modules: []*appv1alpha1.ModuleConfig{
			{
				Name: "runtime",
				Config: appconfig.WrapAny(&runtimev1alpha1.Module{
					AppName: "Nibiru",
					BeginBlockers: []string{
						upgradetypes.ModuleName,
						capabilitytypes.ModuleName,
						distrtypes.ModuleName,
						slashingtypes.ModuleName,
						evidencetypes.ModuleName,
						stakingtypes.ModuleName,
						authtypes.ModuleName,
						banktypes.ModuleName,
						govtypes.ModuleName,
						crisistypes.ModuleName,
						genutiltypes.ModuleName,
						authz.ModuleName,
						feegrant.ModuleName,

						paramstypes.ModuleName,
						epochstypes.ModuleName,
						oracletypes.ModuleName,
						inflationtypes.ModuleName,
						consensustypes.ModuleName,
						sudotypes.ModuleName,
						ibctransfertypes.ModuleName, ibcexported.ModuleName,
						ibcfeetypes.ModuleName, icatypes.ModuleName, ibcwasmtypes.ModuleName,
						// devgastypes.ModuleName,
						tokenfactorytypes.ModuleName,
						evmtypes.ModuleName,
					},
					EndBlockers: []string{
						crisistypes.ModuleName,
						govtypes.ModuleName,
						stakingtypes.ModuleName,
						capabilitytypes.ModuleName,
						authtypes.ModuleName,
						banktypes.ModuleName,
						distrtypes.ModuleName,
						slashingtypes.ModuleName,
						genutiltypes.ModuleName,
						evidencetypes.ModuleName,
						authz.ModuleName,
						feegrant.ModuleName,
						paramstypes.ModuleName,
						consensustypes.ModuleName,
						upgradetypes.ModuleName,
						ibctransfertypes.ModuleName, ibcexported.ModuleName,
						ibcfeetypes.ModuleName, icatypes.ModuleName, ibcwasmtypes.ModuleName,
						// devgastypes.ModuleName,
						epochstypes.ModuleName,
						oracletypes.ModuleName,
						inflationtypes.ModuleName,
						sudotypes.ModuleName,
						tokenfactorytypes.ModuleName,
						evmtypes.ModuleName,
					},
					OverrideStoreKeys: []*runtimev1alpha1.StoreKeyConfig{
						{
							ModuleName: authtypes.ModuleName,
							KvStoreKey: "acc",
						},
					},
					InitGenesis: genesisModuleOrder,
					// When ExportGenesis is not specified, the export genesis module order
					// is equal to the init genesis order
					// ExportGenesis: genesisModuleOrder,
				}),
			},
			{
				Name: authtypes.ModuleName,
				Config: appconfig.WrapAny(&authmodulev1.Module{
					Bech32Prefix:             "nibi",
					ModuleAccountPermissions: moduleAccPerms,
					Authority:                authtypes.NewModuleAddress(govtypes.ModuleName).String(),
				}),
			},
			{
				Name: banktypes.ModuleName,
				Config: appconfig.WrapAny(&bankmodulev1.Module{
					BlockedModuleAccountsOverride: blockAccAddrs,
					Authority:                     authtypes.NewModuleAddress(govtypes.ModuleName).String(),
				}),
			},
			{
				Name: stakingtypes.ModuleName,
				Config: appconfig.WrapAny(&stakingmodulev1.Module{
					Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
				}),
			},
			{
				Name:   slashingtypes.ModuleName,
				Config: appconfig.WrapAny(&slashingmodulev1.Module{}),
			},
			{
				Name:   paramstypes.ModuleName,
				Config: appconfig.WrapAny(&paramsmodulev1.Module{}),
			},
			{
				Name:   "tx",
				Config: appconfig.WrapAny(&txconfigv1.Config{}),
			},
			{
				Name:   genutiltypes.ModuleName,
				Config: appconfig.WrapAny(&genutilmodulev1.Module{}),
			},
			{
				Name:   authz.ModuleName,
				Config: appconfig.WrapAny(&authzmodulev1.Module{}),
			},
			{
				Name:   upgradetypes.ModuleName,
				Config: appconfig.WrapAny(&upgrademodulev1.Module{}),
			},
			{
				Name: distrtypes.ModuleName,
				Config: appconfig.WrapAny(&distrmodulev1.Module{
					FeeCollectorName: authtypes.FeeCollectorName,
					Authority:        authtypes.NewModuleAddress(govtypes.ModuleName).String(),
				}),
			},
			{
				Name: capabilitytypes.ModuleName,
				Config: appconfig.WrapAny(&capabilitymodulev1.Module{
					SealKeeper: true,
				}),
			},
			{
				Name:   evidencetypes.ModuleName,
				Config: appconfig.WrapAny(&evidencemodulev1.Module{}),
			},
			{
				Name:   feegrant.ModuleName,
				Config: appconfig.WrapAny(&feegrantmodulev1.Module{}),
			},
			{
				Name: govtypes.ModuleName,
				Config: appconfig.WrapAny(&govmodulev1.Module{
					Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
				}),
			},
			{
				Name:   crisistypes.ModuleName,
				Config: appconfig.WrapAny(&crisismodulev1.Module{}),
			},
			{
				Name:   consensustypes.ModuleName,
				Config: appconfig.WrapAny(&consensusmodulev1.Module{}),
			},
			// {
			// 	Name:   devgastypes.ModuleName,
			// 	Config: appconfig.WrapAny(&devgassmodulev1.Module{}),
			// },
			{
				Name:   tokenfactorytypes.ModuleName,
				Config: appconfig.WrapAny(&tokenfactorymodulev1.Module{}),
			},
			{
				Name:   sudotypes.ModuleName,
				Config: appconfig.WrapAny(&sudomodulev1.Module{}),
			},
			{
				Name:   oracletypes.ModuleName,
				Config: appconfig.WrapAny(&oraclemodulev1.Module{}),
			},
			{
				Name:   epochstypes.ModuleName,
				Config: appconfig.WrapAny(&epochsmodulev1.Module{}),
			},
			{
				Name:   inflationtypes.ModuleName,
				Config: appconfig.WrapAny(&inflationmodulev1.Module{}),
			},
			{
				Name:   evmtypes.ModuleName,
				Config: appconfig.WrapAny(&evmmodulev1.Module{}),
			},
		},
	})
)
