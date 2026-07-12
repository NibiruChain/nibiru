package testutil

import (
	_ "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/auth"
	_ "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/auth/tx/config"
	_ "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/bank"
	_ "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/consensus"
	_ "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/distribution"
	_ "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/genutil"
	_ "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/mint"
	_ "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/params"
	_ "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/slashing"
	_ "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/staking"

	"cosmossdk.io/core/appconfig"
	authtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/auth/types"
	banktypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/bank/types"
	consensustypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/consensus/types"
	distrtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/distribution/types"
	genutiltypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/genutil/types"
	minttypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/mint/types"
	paramstypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/params/types"
	slashingtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/staking/types"

	runtimev1alpha1 "cosmossdk.io/api/cosmos/app/runtime/v1alpha1"
	appv1alpha1 "cosmossdk.io/api/cosmos/app/v1alpha1"
	authmodulev1 "cosmossdk.io/api/cosmos/auth/module/v1"
	bankmodulev1 "cosmossdk.io/api/cosmos/bank/module/v1"
	consensusmodulev1 "cosmossdk.io/api/cosmos/consensus/module/v1"
	distrmodulev1 "cosmossdk.io/api/cosmos/distribution/module/v1"
	genutilmodulev1 "cosmossdk.io/api/cosmos/genutil/module/v1"
	mintmodulev1 "cosmossdk.io/api/cosmos/mint/module/v1"
	paramsmodulev1 "cosmossdk.io/api/cosmos/params/module/v1"
	slashingmodulev1 "cosmossdk.io/api/cosmos/slashing/module/v1"
	stakingmodulev1 "cosmossdk.io/api/cosmos/staking/module/v1"
	txconfigv1 "cosmossdk.io/api/cosmos/tx/config/v1"
)

var AppConfig = appconfig.Compose(&appv1alpha1.Config{
	Modules: []*appv1alpha1.ModuleConfig{
		{
			Name: "runtime",
			Config: appconfig.WrapAny(&runtimev1alpha1.Module{
				AppName: "SlashingApp",
				BeginBlockers: []string{
					minttypes.ModuleName,
					distrtypes.ModuleName,
					stakingtypes.ModuleName,
					authtypes.ModuleName,
					banktypes.ModuleName,
					genutiltypes.ModuleName,
					slashingtypes.ModuleName,
					paramstypes.ModuleName,
					consensustypes.ModuleName,
				},
				EndBlockers: []string{
					stakingtypes.ModuleName,
					authtypes.ModuleName,
					banktypes.ModuleName,
					genutiltypes.ModuleName,
					distrtypes.ModuleName,
					minttypes.ModuleName,
					slashingtypes.ModuleName,
					paramstypes.ModuleName,
					consensustypes.ModuleName,
				},
				InitGenesis: []string{
					authtypes.ModuleName,
					banktypes.ModuleName,
					distrtypes.ModuleName,
					stakingtypes.ModuleName,
					minttypes.ModuleName,
					slashingtypes.ModuleName,
					genutiltypes.ModuleName,
					paramstypes.ModuleName,
					consensustypes.ModuleName,
				},
			}),
		},
		{
			Name: authtypes.ModuleName,
			Config: appconfig.WrapAny(&authmodulev1.Module{
				Bech32Prefix: "cosmos",
				ModuleAccountPermissions: []*authmodulev1.ModuleAccountPermission{
					{Account: authtypes.FeeCollectorName},
					{Account: distrtypes.ModuleName},
					{Account: minttypes.ModuleName, Permissions: []string{authtypes.Minter}},
					{Account: stakingtypes.BondedPoolName, Permissions: []string{authtypes.Burner, stakingtypes.ModuleName}},
					{Account: stakingtypes.NotBondedPoolName, Permissions: []string{authtypes.Burner, stakingtypes.ModuleName}},
				},
			}),
		},

		{
			Name:   banktypes.ModuleName,
			Config: appconfig.WrapAny(&bankmodulev1.Module{}),
		},
		{
			Name:   stakingtypes.ModuleName,
			Config: appconfig.WrapAny(&stakingmodulev1.Module{}),
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
			Name:   consensustypes.ModuleName,
			Config: appconfig.WrapAny(&consensusmodulev1.Module{}),
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
			Name:   minttypes.ModuleName,
			Config: appconfig.WrapAny(&mintmodulev1.Module{}),
		},
		{
			Name:   distrtypes.ModuleName,
			Config: appconfig.WrapAny(&distrmodulev1.Module{}),
		},
	},
})
