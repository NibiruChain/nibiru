package ibcnibi_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/NibiruChain/nibiru/app"
	// "github.com/NibiruChain/nibiru/x/ibcnibi"
	pricefeedtypes "github.com/NibiruChain/nibiru/x/pricefeed/types"
	stabletypes "github.com/NibiruChain/nibiru/x/stablecoin/types"
	"github.com/NibiruChain/nibiru/x/testutil"
	"github.com/NibiruChain/nibiru/x/testutil/sample"

	"github.com/cosmos/cosmos-sdk/simapp"
	sdk "github.com/cosmos/cosmos-sdk/types"

	transfertypes "github.com/cosmos/ibc-go/v3/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v3/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v3/modules/core/04-channel/types"

	ibctesting "github.com/cosmos/ibc-go/v3/testing"
	ibcmock "github.com/cosmos/ibc-go/v3/testing/mock"
	"github.com/stretchr/testify/suite"
)

/* SetupTestingApp returns the TestingApp and default genesis state used to
   initialize the testing app. */
func SetupNibiruTestingApp() (
	testingApp ibctesting.TestingApp,
	defaultGenesis map[string]json.RawMessage,
) {
	nibiruApp, ctx := testutil.NewNibiruApp(true)
	encCdc := app.MakeTestEncodingConfig()
	token0, token1 := "uatom", "unibi"
	oracle := sample.AccAddress()

	nibiruApp.PriceKeeper.SetParams(ctx, pricefeedtypes.Params{
		Pairs: []pricefeedtypes.Pair{
			{Token0: token0, Token1: token1,
				Oracles: []sdk.AccAddress{oracle}, Active: true},
		},
	})
	nibiruApp.PriceKeeper.SetPrice(
		ctx, oracle, token0, token1, sdk.OneDec(),
		ctx.BlockTime().Add(time.Hour),
	)
	nibiruApp.PriceKeeper.SetCurrentPrices(ctx, token0, token1)

	return nibiruApp, app.NewDefaultGenesisState(encCdc.Marshaler)
}

// init changes the value of 'DefaultTestingAppInit' to use custom initialization.
func init() {
	ibctesting.DefaultTestingAppInit = SetupNibiruTestingApp
}

// KeeperTestSuite is a testing suite to test keeper functions.
type KeeperTestSuite struct {
	suite.Suite

	coordinator *ibctesting.Coordinator

	// testing chains used for convenience and readability
	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain
}

// TestKeeperTestSuite runs all the tests within this package.
func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}

// SetupTest creates a coordinator with 2 test chains.
func (suite *KeeperTestSuite) SetupTest() {
	// initializes 2 test chains
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 2)
	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(1))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(2))
}

func (suite KeeperTestSuite) TestIBC() {
	suite.SetupTest()

	timeoutHeight := clienttypes.NewHeight(1000, 1000)
	ack := ibcmock.MockAcknowledgement

	path := ibctesting.NewPath(suite.chainA, suite.chainB) // clientID, connectionID, channelID empty
	suite.coordinator.Setup(path)                          // clientID, connectionID, channelID filled
	suite.Require().Equal("07-tendermint-0", path.EndpointA.ClientID)
	suite.Require().Equal("connection-0", path.EndpointA.ClientID)
	suite.Require().Equal("channel-0", path.EndpointA.ClientID)

	sender := sample.AccAddress().String()
	receiver := sample.AccAddress().String()
	transfer := transfertypes.NewFungibleTokenPacketData(
		"unibi", "1000", sender, receiver)
	bz := transfertypes.ModuleCdc.MustMarshalJSON(&transfer)

	// create packet 1
	packet1 := channeltypes.Packet{
		Data:               bz,
		Sequence:           1,
		SourcePort:         path.EndpointA.ChannelConfig.PortID,
		SourceChannel:      path.EndpointA.ChannelID,
		DestinationPort:    path.EndpointB.ChannelConfig.PortID,
		DestinationChannel: path.EndpointB.ChannelID,
		TimeoutHeight:      timeoutHeight,
		TimeoutTimestamp:   0}

	// send on endpointA
	err := path.EndpointA.SendPacket(packet1)
	suite.Require().NoError(err)

	// receive on endpointB
	err = path.EndpointB.RecvPacket(packet1)
	suite.Require().NoError(err)

	// acknowledge the receipt of the packet
	err = path.EndpointA.AcknowledgePacket(packet1, ack.Acknowledgement())
	suite.Require().NoError(err)

	err = simapp.FundModuleAccount(
		/* bankKeeper */ suite.chainA.App.(*app.NibiruApp).BankKeeper,
		/* ctx */ suite.chainB.GetContext(),
		/* recipientModule */ stabletypes.ModuleName,
		/* coins */ sdk.NewCoins(sdk.NewInt64Coin("uatom", 100)),
	)
	suite.Require().NoError(err)

	// // we can also relay
	// packet2 := channeltypes.NewPacket()

	// path.EndpointA.SendPacket(packet2)

	// path.Relay(packet2, expectedAck)

	// // if needed we can update our clients
	// path.EndpointB.UpdateClient()
}
