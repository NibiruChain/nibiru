package keeper_test

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/core/04-channel/types"
	ibctesting "github.com/NibiruChain/nibiru/v2/lib/ibc-go/testing"
	ibcmock "github.com/NibiruChain/nibiru/v2/lib/ibc-go/testing/mock"
)

// KeeperTestSuite is a testing suite to test keeper functions.
type KeeperTestSuite struct {
	suite.Suite

	coordinator          *ibctesting.Coordinator
	connectionCheckpoint *ibctesting.ConnectionCheckpoint

	// testing chains used for convenience and readability
	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain
}

// TestKeeperTestSuite runs all the tests within this package.
func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}

func (suite *KeeperTestSuite) SetupSuite() {
	suite.connectionCheckpoint = newConnectionCheckpoint(suite.T())
}

// SetupTest creates a coordinator with 2 test chains.
func (suite *KeeperTestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 2)
	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(1))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(2))
	// commit some blocks so that QueryProof returns valid proof (cannot return valid query if height <= 1)
	suite.coordinator.CommitNBlocks(suite.chainA, 2)
	suite.coordinator.CommitNBlocks(suite.chainB, 2)
}

func (suite *KeeperTestSuite) RestoreConnectionCheckpoint() *ibctesting.Path {
	coordinator, path := suite.connectionCheckpoint.Restore(suite.T())
	suite.coordinator = coordinator
	suite.chainA = path.EndpointA.Chain
	suite.chainB = path.EndpointB.Chain
	return path
}

// TestSetChannel create clients and connections on both chains. It tests for the non-existence
// and existence of a channel in INIT on chainA.
func (suite *KeeperTestSuite) TestSetChannel() {
	// create client and connections on both chains
	path := ibctesting.NewPath(suite.chainA, suite.chainB)
	suite.coordinator.SetupConnections(path)

	// check for channel to be created on chainA
	found := suite.chainA.App.GetIBCKeeper().ChannelKeeper.HasChannel(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
	suite.False(found)

	path.SetChannelOrdered()

	// init channel
	err := path.EndpointA.ChanOpenInit()
	suite.NoError(err)

	storedChannel, found := suite.chainA.App.GetIBCKeeper().ChannelKeeper.GetChannel(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
	// counterparty channel id is empty after open init
	expectedCounterparty := types.NewCounterparty(path.EndpointB.ChannelConfig.PortID, "")

	suite.True(found)
	suite.Equal(types.INIT, storedChannel.State)
	suite.Equal(types.ORDERED, storedChannel.Ordering)
	suite.Equal(expectedCounterparty, storedChannel.Counterparty)
}

func (suite *KeeperTestSuite) TestGetAppVersion() {
	// create client and connections on both chains
	path := ibctesting.NewPath(suite.chainA, suite.chainB)
	suite.coordinator.SetupConnections(path)

	version, found := suite.chainA.App.GetIBCKeeper().ChannelKeeper.GetAppVersion(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
	suite.Require().False(found)
	suite.Require().Empty(version)

	// init channel
	err := path.EndpointA.ChanOpenInit()
	suite.NoError(err)

	channelVersion, found := suite.chainA.App.GetIBCKeeper().ChannelKeeper.GetAppVersion(suite.chainA.GetContext(), path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID)
	suite.Require().True(found)
	suite.Require().Equal(ibcmock.Version, channelVersion)
}

// containsAll verifies if all elements in the expected slice exist in the actual slice
// independent of order.
func containsAll(expected, actual []types.IdentifiedChannel) bool {
	for _, expectedChannel := range expected {
		foundMatch := false
		for _, actualChannel := range actual {
			if reflect.DeepEqual(actualChannel, expectedChannel) {
				foundMatch = true
				break
			}
		}
		if !foundMatch {
			return false
		}
	}
	return true
}
