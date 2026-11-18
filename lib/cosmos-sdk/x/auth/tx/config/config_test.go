package tx_test

import (
	"errors"
	"testing"

	txconfigv1 "cosmossdk.io/api/cosmos/tx/config/v1"
	"cosmossdk.io/depinject"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/client"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/codec"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/testutil"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/testutil/testdata"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	moduletestutil "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/module/testutil"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/auth/ante"
	antetestutil "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/auth/ante/testutil"
	authtestutil "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/auth/testutil"
	txconfig "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/auth/tx/config"
	authtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/auth/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/feegrant"
	feegrantkeeper "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/feegrant/keeper"
	feegrantmodule "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/feegrant/module"
	feegranttestutil "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/feegrant/testutil"
)

// InjectedTxInputs captures the dependencies selected by the transaction
// configuration provider so tests can exercise the same fee consumer.
type InjectedTxInputs struct {
	Inputs txconfig.TxInputs
}

// RejectingFeegrant is an alternate fee-grant implementation that records
// calls and returns a chosen error. It proves injection depends on the keeper
// interface, rather than requiring the SDK concrete implementation.
type RejectingFeegrant struct {
	err   error
	calls int
}

// UseGrantedFees records delegation to the injected implementation and returns
// its rejection so the test can verify error propagation through fee deduction.
func (k *RejectingFeegrant) UseGrantedFees(sdk.Context, sdk.AccAddress, sdk.AccAddress, sdk.Coins, []sdk.Msg) error {
	k.calls++
	return k.err
}

// TestFeegrantProviderInjection runs fee deduction with the dependencies chosen
// by depinject. It covers the real SDK allowance being debited, an alternate
// implementation rejecting the grant, a missing optional provider rejecting a
// sponsored transaction, and ordinary self-paid fees working without feegrant.
func TestFeegrantProviderInjection(t *testing.T) {
	for _, name := range []string{"sdk keeper", "substitute keeper", "absent sponsored", "absent self-paid"} {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			key := sdk.NewKVStoreKey(feegrant.StoreKey)
			ctx := testutil.DefaultContextWithDB(t, key, sdk.NewTransientStoreKey("transient")).Ctx.WithBlockHeight(1)
			enc := moduletestutil.MakeTestEncodingConfig(feegrantmodule.AppModuleBasic{})
			testdata.RegisterInterfaces(enc.InterfaceRegistry)
			_, _, payer := testdata.KeyTestPubAddr()
			_, _, granter := testdata.KeyTestPubAddr()
			fee := sdk.NewCoins(sdk.NewInt64Coin("stake", 10))
			account := antetestutil.NewMockAccountKeeper(ctrl)
			bank := authtestutil.NewMockBankKeeper(ctrl)
			account.EXPECT().GetModuleAddress(authtypes.FeeCollectorName).Return(authtypes.NewModuleAddress(authtypes.FeeCollectorName))

			configs := []depinject.Config{
				depinject.Supply(&txconfigv1.Config{}),
				depinject.Provide(
					func() codec.ProtoCodecMarshaler { return enc.Codec.(*codec.ProtoCodec) },
					func() ante.AccountKeeper { return account },
					func() authtypes.BankKeeper { return bank },
				),
				depinject.Provide(txconfig.ProvideModule, func(in txconfig.TxInputs) InjectedTxInputs {
					return InjectedTxInputs{in}
				}),
			}
			var realKeeper feegrantkeeper.Keeper
			var substitute *RejectingFeegrant
			switch name {
			case "sdk keeper":
				grantAccount := feegranttestutil.NewMockAccountKeeper(ctrl)
				grantAccount.EXPECT().GetAccount(gomock.Any(), payer).Return(authtypes.NewBaseAccountWithAddress(payer))
				realKeeper = feegrantkeeper.NewKeeper(enc.Codec, key, grantAccount)
				require.NoError(t, realKeeper.GrantAllowance(ctx, granter, payer, &feegrant.BasicAllowance{
					SpendLimit: sdk.NewCoins(sdk.NewInt64Coin("stake", 100)),
				}))
				configs = append(configs, depinject.Provide(func() feegrantkeeper.Keeper { return realKeeper }))
			case "substitute keeper":
				substitute = &RejectingFeegrant{err: errors.New("grant rejected by substitute")}
				configs = append(configs, depinject.Provide(func() *RejectingFeegrant { return substitute }))
			}

			var in InjectedTxInputs
			var txCfg client.TxConfig
			require.NoError(t, depinject.Inject(depinject.Configs(configs...), &in, &txCfg))
			builder := txCfg.NewTxBuilder()
			require.NoError(t, builder.SetMsgs(testdata.NewTestMsg(payer)))
			builder.SetGasLimit(200000)
			builder.SetFeeAmount(fee)
			builder.SetFeeGranter(granter)
			if name == "absent self-paid" {
				builder.SetFeeGranter(nil)
			}

			if name == "sdk keeper" || name == "absent self-paid" {
				charged := granter
				if name == "absent self-paid" {
					charged = payer
				}
				account.EXPECT().GetAccount(gomock.Any(), charged).Return(authtypes.NewBaseAccountWithAddress(charged))
				bank.EXPECT().SendCoinsFromAccountToModule(gomock.Any(), charged, authtypes.FeeCollectorName, fee).Return(nil)
			}

			// Exercise the fee consumer with the dependencies selected by the
			// transaction configuration provider, including concrete-to-interface
			// resolution and the absent optional provider.
			handler := sdk.ChainAnteDecorators(ante.NewDeductFeeDecorator(in.Inputs.AccountKeeper, in.Inputs.BankKeeper, in.Inputs.FeeGrantKeeper, nil))
			_, err := handler(ctx, builder.GetTx(), false)
			switch name {
			case "sdk keeper":
				require.NoError(t, err)
				allowance, err := realKeeper.GetAllowance(ctx, granter, payer)
				require.NoError(t, err)
				require.Equal(t, sdk.NewCoins(sdk.NewInt64Coin("stake", 90)), allowance.(*feegrant.BasicAllowance).SpendLimit)
			case "substitute keeper":
				require.ErrorContains(t, err, substitute.err.Error())
				require.Equal(t, 1, substitute.calls)
			case "absent sponsored":
				require.Nil(t, in.Inputs.FeeGrantKeeper)
				require.ErrorContains(t, err, "fee grants are not enabled")
			case "absent self-paid":
				require.Nil(t, in.Inputs.FeeGrantKeeper)
				require.NoError(t, err)
			}
		})
	}
}

// TestFeegrantAppProviderInjection checks the complete SDK test-app provider
// graph, rather than the isolated providers above. The real feegrant keeper
// must resolve into the transaction consumer's interface alongside account and
// bank keepers, and the resulting transaction configuration must build a tx.
func TestFeegrantAppProviderInjection(t *testing.T) {
	var in InjectedTxInputs
	var txCfg client.TxConfig
	require.NoError(t, depinject.Inject(depinject.Configs(
		feegranttestutil.AppConfig,
		depinject.ProvideInModule("tx", CaptureTxInputs),
	), &in, &txCfg))
	require.IsType(t, feegrantkeeper.Keeper{}, in.Inputs.FeeGrantKeeper)
	require.NotNil(t, in.Inputs.AccountKeeper)
	require.NotNil(t, in.Inputs.BankKeeper)
	require.NotNil(t, txCfg.NewTxBuilder())
}

// CaptureTxInputs exposes module-scoped injected dependencies for assertions
// without replacing the application transaction-configuration provider.
func CaptureTxInputs(inputs txconfig.TxInputs) InjectedTxInputs {
	return InjectedTxInputs{inputs}
}
