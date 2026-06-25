package upgrades_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/app/upgrades"
	"github.com/NibiruChain/nibiru/v2/x/collections"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
)

func TestUpgrade2_15_0_SetsOracleImplAdapterAddrWhenConfigured(t *testing.T) {
	deps := evmtest.NewTestDeps()
	ctx := deps.Ctx()
	oldAddrCfg := upgrades.AddrCfg_v2_15
	defer func() { upgrades.AddrCfg_v2_15 = oldAddrCfg }()

	wantAddr := deps.Sender.NibiruAddr
	upgrades.AddrCfg_v2_15.ImplAdapterAddr = wantAddr

	upgrades.RunUpgrade2_15_0(&deps.App.PublicKeepers, ctx)

	gotAddr, err := deps.App.OracleKeeper.ImplAdapterAddr.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, wantAddr.String(), gotAddr.String())
}

func TestUpgrade2_15_0_SkipsOracleImplAdapterAddrWhenUnset(t *testing.T) {
	deps := evmtest.NewTestDeps()
	ctx := deps.Ctx()
	oldAddrCfg := upgrades.AddrCfg_v2_15
	defer func() { upgrades.AddrCfg_v2_15 = oldAddrCfg }()

	upgrades.AddrCfg_v2_15.ImplAdapterAddr = nil

	upgrades.RunUpgrade2_15_0(&deps.App.PublicKeepers, ctx)

	_, err := deps.App.OracleKeeper.ImplAdapterAddr.Get(ctx)
	require.ErrorIs(t, err, collections.ErrNotFound)
}
