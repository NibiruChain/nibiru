// Package testutil holds test helpers and mocks of the Cosmos SDK
// x/consensus module.
//
// Deprecated: NibiruApp no longer uses this module. Nibiru's own x/consensus
// module (github.com/NibiruChain/nibiru/v2/x/consensus) owns the "consensus"
// store, serves the Params query, and rejects MsgUpdateParams, because the
// x-consensus contract owns the consensus params. The types moved to
// github.com/NibiruChain/nibiru/v2/x/consensus/types, which this package
// imports. It is kept only for lib's own depinject test apps (the
// testutil/app_config.go files and testutil/configurator) and the ibc-go
// simapps, and will be removed together with that test wiring.
package testutil

import (
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	bankkeeper "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/bank/keeper"
	"github.com/NibiruChain/nibiru/v2/x/mint"
)

// FundAccount is a utility function that funds an account by minting and
// sending the coins to the address. This should be used for testing purposes
// only!
//
// TODO: Instead of using the mint module account, which has the
// permission of minting, create a "faucet" account. (@fdymylja)
func FundAccount(bankKeeper bankkeeper.Keeper, ctx sdk.Context, addr sdk.AccAddress, amounts sdk.Coins) error {
	if err := bankKeeper.MintCoins(ctx, mint.ModuleName, amounts); err != nil {
		return err
	}

	return bankKeeper.SendCoinsFromModuleToAccount(ctx, mint.ModuleName, addr, amounts)
}

// FundModuleAccount is a utility function that funds a module account by
// minting and sending the coins to the address. This should be used for testing
// purposes only!
//
// TODO: Instead of using the mint module account, which has the
// permission of minting, create a "faucet" account. (@fdymylja)
func FundModuleAccount(bankKeeper bankkeeper.Keeper, ctx sdk.Context, recipientMod string, amounts sdk.Coins) error {
	if err := bankKeeper.MintCoins(ctx, mint.ModuleName, amounts); err != nil {
		return err
	}

	return bankKeeper.SendCoinsFromModuleToModule(ctx, mint.ModuleName, recipientMod, amounts)
}
