package v2_4_0

import (
	"context"

	"cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	clientkeeper "github.com/cosmos/ibc-go/v8/modules/core/02-client/keeper"

	"github.com/NibiruChain/nibiru/v2/app/upgrades"
)

const UpgradeName = "v2.4.0"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: upgrades.DefaultUpgradeHandler,
	StoreUpgrades:        types.StoreUpgrades{},
}
