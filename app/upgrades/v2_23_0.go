package upgrades

import (
	"encoding/hex"
	"fmt"

	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/NibiruChain/nibiru/v2/app/keepers"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/module"
	upgradetypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/upgrade/types"
	wasmkeeper "github.com/NibiruChain/nibiru/v2/x/wasm/keeper"
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
