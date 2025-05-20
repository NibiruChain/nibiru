package v2_5_0

import (
	"fmt"

	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/NibiruChain/nibiru/v2/app/keepers"
	"github.com/NibiruChain/nibiru/v2/app/upgrades"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	clientkeeper "github.com/cosmos/ibc-go/v7/modules/core/02-client/keeper"
)

const UpgradeName = "v2.5.0"

var Upgrade = upgrades.Upgrade{
	UpgradeName: UpgradeName,
	CreateUpgradeHandler: func(
		mm *module.Manager,
		cfg module.Configurator,
		nibiru *keepers.PublicKeepers,
		clientKeeper clientkeeper.Keeper,
	) upgradetypes.UpgradeHandler {
		return func(
			ctx sdk.Context,
			plan upgradetypes.Plan,
			fromVM module.VersionMap,
		) (module.VersionMap, error) {
			err := UpgradeStNibiContractOnMainnet(nibiru, ctx)
			if err != nil {
				panic(fmt.Errorf("v2.5.0 upgrade failure: %w", err))
			}

			return mm.RunMigrations(ctx, cfg, fromVM)
		}
	},
	StoreUpgrades: storetypes.StoreUpgrades{},
}

// Set of addresses that held stNIBI in ERC20 form prior to the v2.5.0 upgrade
func MAINNET_STNIBI_HOLDERS() []gethcommon.Address {
	holderStrs := []string{
		"0x603871c2ddd41c26Ee77495E2E31e6De7f9957e0",
		"0x447fC0471f5d8150A790e19000930108aCCe0BC7",
		"0x2DD37531749e1AF248720fe8F9D1A51517D9748F",
		"0x07A1D54115D5B535F298231Cc6EaA360Bc4667f9",
		"0x447fC0471f5d8150A790e19000930108aCCe0BC7",
		"0x5361EBC7CF6689565EC4C380b7793e238ad6Be45",
	}
	holders := make([]gethcommon.Address, len(holderStrs))
	for idx, h := range holderStrs {
		holders[idx] = gethcommon.HexToAddress(h)
	}
	return holders
}

func UpgradeStNibiContractOnMainnet(
	nibiru *keepers.PublicKeepers,
	ctx sdk.Context,
) error {
	addrStNibiMainnet := gethcommon.HexToAddress("0xcA0a9Fb5FBF692fa12fD13c0A900EC56Bb3f0a7b")

	// Early return if the mapping for stNIBI doesn't exist
	erc20AddrIter := nibiru.EvmKeeper.FunTokens.Indexes.ERC20Addr.ExactMatch(ctx, addrStNibiMainnet)
	funTokenMappings := nibiru.EvmKeeper.FunTokens.Collect(ctx, erc20AddrIter)
	if len(funTokenMappings) != 1 {
		return nil
	}

	// Early return if the address is not a contract
	accOfContract := nibiru.EvmKeeper.GetAccount(ctx, addrStNibiMainnet)
	if accOfContract == nil || !accOfContract.IsContract() {
		return nil
	}

	// userBalances

	return nil
}
