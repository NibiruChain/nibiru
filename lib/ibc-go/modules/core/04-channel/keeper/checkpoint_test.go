package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/core/04-channel/types"
	ibctesting "github.com/NibiruChain/nibiru/v2/lib/ibc-go/testing"
)

func TestConnectionCheckpointRestore(t *testing.T) {
	checkpoint := newConnectionCheckpoint(t)
	checkpoint.AssertConnectionState(t)
}

func TestConnectionCheckpointRestoreIsolation(t *testing.T) {
	checkpoint := newConnectionCheckpoint(t)

	_, path := checkpoint.Restore(t)
	channel := types.NewChannel(
		types.INIT,
		types.UNORDERED,
		types.NewCounterparty(path.EndpointB.ChannelConfig.PortID, ""),
		[]string{path.EndpointA.ConnectionID},
		path.EndpointA.ChannelConfig.Version,
	)
	path.EndpointA.Chain.App.GetIBCKeeper().ChannelKeeper.SetChannel(
		path.EndpointA.Chain.GetContext(),
		path.EndpointA.ChannelConfig.PortID,
		ibctesting.FirstChannelID,
		channel,
	)
	_, found := path.EndpointA.Chain.App.GetIBCKeeper().ChannelKeeper.GetChannel(
		path.EndpointA.Chain.GetContext(),
		path.EndpointA.ChannelConfig.PortID,
		ibctesting.FirstChannelID,
	)
	require.True(t, found)

	_, restoredPath := checkpoint.Restore(t)
	_, found = restoredPath.EndpointA.Chain.App.GetIBCKeeper().ChannelKeeper.GetChannel(
		restoredPath.EndpointA.Chain.GetContext(),
		restoredPath.EndpointA.ChannelConfig.PortID,
		ibctesting.FirstChannelID,
	)
	require.False(t, found)
}

func newConnectionCheckpoint(t *testing.T) *ibctesting.ConnectionCheckpoint {
	t.Helper()

	coordinator := ibctesting.NewCoordinator(t, 2)
	chainA := coordinator.GetChain(ibctesting.GetChainID(1))
	chainB := coordinator.GetChain(ibctesting.GetChainID(2))
	path := ibctesting.NewPath(chainA, chainB)
	coordinator.SetupConnections(path)
	chainA.CreatePortCapability(chainA.GetSimApp().ScopedIBCMockKeeper, ibctesting.MockPort)
	chainB.CreatePortCapability(chainB.GetSimApp().ScopedIBCMockKeeper, ibctesting.MockPort)

	return ibctesting.CaptureConnectionCheckpoint(t, coordinator, path)
}
