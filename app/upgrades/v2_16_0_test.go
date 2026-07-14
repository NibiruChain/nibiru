package upgrades_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/app/upgrades"
	"github.com/NibiruChain/nibiru/v2/evm/evmtest"
)

func TestUpgrade2_16_0_SetsGovVotingPeriodOnMainnet(t *testing.T) {
	deps := evmtest.NewTestDeps()
	deps.SetCtx(deps.Ctx().WithChainID(appconst.SDK_CHAIN_ID_MAINNET))

	require.NoError(t, deps.RunUpgrade(upgrades.Upgrade2_16_0))

	params := deps.App.GovKeeper.GetParams(deps.Ctx())
	require.NotNil(t, params.VotingPeriod)
	require.Equal(t, 24*time.Hour, *params.VotingPeriod)
}

func TestUpgrade2_16_0_SkipsGovVotingPeriodOnNonMainnet(t *testing.T) {
	deps := evmtest.NewTestDeps()
	ctx := deps.Ctx()

	before := deps.App.GovKeeper.GetParams(ctx)
	require.NotNil(t, before.VotingPeriod)
	beforePeriod := *before.VotingPeriod

	require.NoError(t, deps.RunUpgrade(upgrades.Upgrade2_16_0))

	after := deps.App.GovKeeper.GetParams(ctx)
	require.NotNil(t, after.VotingPeriod)
	require.Equal(t, beforePeriod, *after.VotingPeriod)
}
