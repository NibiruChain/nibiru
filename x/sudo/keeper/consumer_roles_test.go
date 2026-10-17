package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/evm"
	authtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/auth/types"
	banktypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/bank/types"
	govtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/gov/types"
	"github.com/NibiruChain/nibiru/v2/x/mint"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil"
	"github.com/NibiruChain/nibiru/v2/x/sudo"
	tftypes "github.com/NibiruChain/nibiru/v2/x/tokenfactory/types"
)

func TestScopedRoleConsumers(t *testing.T) {
	for _, actorKind := range []string{"root", sudo.RoleTFOper, sudo.RoleChainParams, sudo.RoleWasmDeployer, "stranger", "revoked", "governance"} {
		t.Run(actorKind, func(t *testing.T) {
			app, k, ctx := setup()
			ctx = ctx.WithChainID(appconst.SDK_CHAIN_ID_MAINNET)
			root, actor := testutil.NewAccAddress(), testutil.NewAccAddress()
			if actorKind == "root" {
				actor = root
			}
			if actorKind == "governance" {
				actor = authtypes.NewModuleAddress(govtypes.ModuleName)
			}
			k.Sudoers.Set(ctx, sudo.Sudoers{Root: root.String()})
			switch actorKind {
			case "revoked":
				for _, role := range []string{sudo.RoleTFOper, sudo.RoleChainParams} {
					_, err := k.UpdateRoleMembers(ctx, &sudo.MsgUpdateRoleMembers{Sender: root.String(), Role: role, Add: []string{actor.String()}})
					require.NoError(t, err)
					_, err = k.UpdateRoleMembers(ctx, &sudo.MsgUpdateRoleMembers{Sender: root.String(), Role: role, Remove: []string{actor.String()}})
					require.NoError(t, err)
				}
			case sudo.RoleTFOper, sudo.RoleChainParams, sudo.RoleWasmDeployer:
				_, err := k.UpdateRoleMembers(ctx, &sudo.MsgUpdateRoleMembers{Sender: root.String(), Role: actorKind, Add: []string{actor.String()}})
				require.NoError(t, err)
			}
			check := func(name, role string, governance bool, err error) {
				t.Helper()
				allowed := actorKind == "root" || actorKind == role || (governance && actorKind == "governance")
				if allowed {
					require.NoError(t, err, name)
				} else {
					require.Error(t, err, name)
				}
			}
			_, err := app.TokenFactoryKeeper.CreateDenom(ctx, &tftypes.MsgCreateDenom{Sender: actor.String(), Subdenom: "scoped"})
			check("create denom", sudo.RoleTFOper, true, err)
			metadata := banktypes.Metadata{Base: "scopedcoin", Display: "scopedcoin", Name: "Scoped coin", Symbol: "SCOPE", DenomUnits: []*banktypes.DenomUnit{{Denom: "scopedcoin", Exponent: 0}}}
			_, err = app.TokenFactoryKeeper.SudoSetDenomMetadata(ctx, &tftypes.MsgSudoSetDenomMetadata{Sender: actor.String(), Metadata: metadata})
			check("sudo metadata", sudo.RoleTFOper, false, err)
			err = app.InflationKeeper.Sudo().EditInflationParams(ctx, mint.MsgEditInflationParams{}, actor)
			check("inflation params", sudo.RoleChainParams, false, err)
			err = app.InflationKeeper.Sudo().ToggleInflation(ctx, false, actor)
			check("toggle inflation", sudo.RoleChainParams, false, err)
			_, err = k.EditZeroGasActors(ctx, &sudo.MsgEditZeroGasActors{Sender: actor.String()})
			check("zero gas actors", sudo.RoleChainParams, false, err)
			_, err = app.EvmKeeper.UpdateParams(ctx, &evm.MsgUpdateParams{Authority: actor.String(), Params: app.EvmKeeper.GetParams(ctx)})
			check("EVM params", sudo.RoleChainParams, true, err)
			app.BankKeeper.SetDenomMetaData(ctx, metadata)
			require.NoError(t, testapp.FundAccount(app.BankKeeper, ctx, actor, app.EvmKeeper.FeeForCreateFunToken(ctx)))
			_, err = app.EvmKeeper.CreateFunToken(ctx, &evm.MsgCreateFunToken{Sender: actor.String(), FromBankDenom: metadata.Base, AllowZeroDecimals: true})
			check("mainnet FunToken", sudo.RoleChainParams, true, err)
			if actorKind == "stranger" {
				localCtx := ctx.WithChainID("nibiru-localnet-0")
				_, err = app.EvmKeeper.CreateFunToken(localCtx, &evm.MsgCreateFunToken{Sender: actor.String(), FromBankDenom: metadata.Base, AllowZeroDecimals: true})
				require.NoError(t, err, "FunToken admission remains open outside mainnet")
			}
			if actorKind == sudo.RoleTFOper || actorKind == sudo.RoleChainParams {
				_, err = k.UpdateRoleMembers(ctx, &sudo.MsgUpdateRoleMembers{Sender: actor.String(), Role: sudo.RoleWasmDeployer, Add: []string{actor.String()}})
				require.Error(t, err, "compatibility roles cannot grant roles")
			}
		})
	}
}
