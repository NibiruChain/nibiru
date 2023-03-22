package app_test

import (
	"encoding/json"
	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/x/common/testutil/testapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	transfertypes "github.com/cosmos/ibc-go/v4/modules/apps/transfer/types"
	"github.com/cosmos/ibc-go/v4/modules/core/02-client/types"
	ibctesting "github.com/cosmos/ibc-go/v4/testing"
	"github.com/stretchr/testify/suite"
	"testing"
)

// init changes the value of 'DefaultTestingAppInit' to use custom initialization.
func init() {
	ibctesting.DefaultTestingAppInit = SetupNibiruTestingApp
}

/*
SetupTestingApp returns the TestingApp and default genesis state used to

	initialize the testing app.
*/
func SetupNibiruTestingApp() (
	testingApp ibctesting.TestingApp,
	defaultGenesis map[string]json.RawMessage,
) {
	// create testing app
	nibiruApp, _ := testapp.NewNibiruTestAppAndContext(true)

	// Create genesis state
	encCdc := app.MakeTestEncodingConfig()
	genesisState := app.NewDefaultGenesisState(encCdc.Marshaler)

	return nibiruApp, genesisState
}

// IBCTestSuite is a testing suite to test keeper functions.
type IBCTestSuite struct {
	suite.Suite

	coordinator *ibctesting.Coordinator

	// testing chains used for convenience and readability
	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain
	chainC *ibctesting.TestChain

	path *ibctesting.Path // chainA <---> chainB
}

// TestIBCTestSuite runs all the tests within this package.
func TestIBCTestSuite(t *testing.T) {
	suite.Run(t, new(IBCTestSuite))
}

/*
NewIBCTestingTransferPath returns a "path" for testing.

	A path contains two endpoints, 'EndpointA' and 'EndpointB' that correspond
	to the order of the chains passed into the ibctesting.NewPath function.
	A path is a pointer, and its values will be filled in as necessary during
	the setup portion of testing.
*/
func NewIBCTestingTransferPath(
	chainA, chainB *ibctesting.TestChain,
) *ibctesting.Path {
	path := ibctesting.NewPath(chainA, chainB)
	path.EndpointA.ChannelConfig.PortID = ibctesting.TransferPort
	path.EndpointB.ChannelConfig.PortID = ibctesting.TransferPort
	path.EndpointA.ChannelConfig.Version = transfertypes.Version // "ics20-1"
	path.EndpointB.ChannelConfig.Version = transfertypes.Version // "ics20-1"

	return path
}

// SetupTest creates a coordinator with 2 test chains.
func (suite *IBCTestSuite) SetupTest() {
	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 3)
	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(1))
	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(2))
	suite.chainC = suite.coordinator.GetChain(ibctesting.GetChainID(3))
}

// constructs a send from chainA to chainB on the established channel/connection
// and sends the same coin back from chainB to chainA.
func (suite *IBCTestSuite) TestHandleMsgTransfer() {
	// setup between chainA and chainB
	path := NewIBCTestingTransferPath(suite.chainA, suite.chainB)
	suite.coordinator.Setup(path)

	timeoutHeight := types.NewHeight(0, 110)

	amount, ok := sdk.NewIntFromString("9223372036854775808") // 2^63 (one above int64)
	suite.Require().True(ok)
	coinToSendToB := sdk.NewCoin(app.BondDenom, amount)

	// send from chainA to chainB
	msg := transfertypes.NewMsgTransfer(path.EndpointA.ChannelConfig.PortID, path.EndpointA.ChannelID, coinToSendToB, suite.chainA.SenderAccount.GetAddress().String(), suite.chainB.SenderAccount.GetAddress().String(), timeoutHeight, 0)
	res, err := suite.chainA.SendMsgs(msg)
	suite.Require().NoError(err) // message committed

	packet, err := ibctesting.ParsePacketFromEvents(res.GetEvents())
	suite.Require().NoError(err)

	// relay send
	err = path.RelayPacket(packet)
	suite.Require().NoError(err) // relay committed

	// check that voucher exists on chain B
	voucherDenomTrace := transfertypes.ParseDenomTrace(transfertypes.GetPrefixedDenom(packet.GetDestPort(), packet.GetDestChannel(), sdk.DefaultBondDenom))
	balance := suite.chainB.GetSimApp().BankKeeper.GetBalance(suite.chainB.GetContext(), suite.chainB.SenderAccount.GetAddress(), voucherDenomTrace.IBCDenom())

	coinSentFromAToB := transfertypes.GetTransferCoin(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, sdk.DefaultBondDenom, amount)
	suite.Require().Equal(coinSentFromAToB, balance)

	// setup between chainB to chainC
	// NOTE:
	// pathBtoC.EndpointA = endpoint on chainB
	// pathBtoC.EndpointB = endpoint on chainC
	pathBtoC := NewIBCTestingTransferPath(suite.chainB, suite.chainC)
	suite.coordinator.Setup(pathBtoC)

	// send from chainB to chainC
	msg = transfertypes.NewMsgTransfer(pathBtoC.EndpointA.ChannelConfig.PortID, pathBtoC.EndpointA.ChannelID, coinSentFromAToB, suite.chainB.SenderAccount.GetAddress().String(), suite.chainC.SenderAccount.GetAddress().String(), timeoutHeight, 0)
	res, err = suite.chainB.SendMsgs(msg)
	suite.Require().NoError(err) // message committed

	packet, err = ibctesting.ParsePacketFromEvents(res.GetEvents())
	suite.Require().NoError(err)

	err = pathBtoC.RelayPacket(packet)
	suite.Require().NoError(err) // relay committed

	// NOTE: fungible token is prefixed with the full trace in order to verify the packet commitment
	fullDenomPath := transfertypes.GetPrefixedDenom(pathBtoC.EndpointB.ChannelConfig.PortID, pathBtoC.EndpointB.ChannelID, voucherDenomTrace.GetFullDenomPath())

	coinSentFromBToC := sdk.NewCoin(transfertypes.ParseDenomTrace(fullDenomPath).IBCDenom(), amount)
	balance = suite.chainC.GetSimApp().BankKeeper.GetBalance(suite.chainC.GetContext(), suite.chainC.SenderAccount.GetAddress(), coinSentFromBToC.Denom)

	// check that the balance is updated on chainC
	suite.Require().Equal(coinSentFromBToC, balance)

	// check that balance on chain B is empty
	balance = suite.chainB.GetSimApp().BankKeeper.GetBalance(suite.chainB.GetContext(), suite.chainB.SenderAccount.GetAddress(), coinSentFromBToC.Denom)
	suite.Require().Zero(balance.Amount.Int64())

	// send from chainC back to chainB
	msg = transfertypes.NewMsgTransfer(pathBtoC.EndpointB.ChannelConfig.PortID, pathBtoC.EndpointB.ChannelID, coinSentFromBToC, suite.chainC.SenderAccount.GetAddress().String(), suite.chainB.SenderAccount.GetAddress().String(), timeoutHeight, 0)
	res, err = suite.chainC.SendMsgs(msg)
	suite.Require().NoError(err) // message committed

	packet, err = ibctesting.ParsePacketFromEvents(res.GetEvents())
	suite.Require().NoError(err)

	err = pathBtoC.RelayPacket(packet)
	suite.Require().NoError(err) // relay committed

	balance = suite.chainB.GetSimApp().BankKeeper.GetBalance(suite.chainB.GetContext(), suite.chainB.SenderAccount.GetAddress(), coinSentFromAToB.Denom)

	// check that the balance on chainA returned back to the original state
	suite.Require().Equal(coinSentFromAToB, balance)

	// check that module account escrow address is empty
	escrowAddress := transfertypes.GetEscrowAddress(packet.GetDestPort(), packet.GetDestChannel())
	balance = suite.chainB.GetSimApp().BankKeeper.GetBalance(suite.chainB.GetContext(), escrowAddress, sdk.DefaultBondDenom)
	suite.Require().Equal(sdk.NewCoin(sdk.DefaultBondDenom, sdk.ZeroInt()), balance)

	// check that balance on chain B is empty
	balance = suite.chainC.GetSimApp().BankKeeper.GetBalance(suite.chainC.GetContext(), suite.chainC.SenderAccount.GetAddress(), voucherDenomTrace.IBCDenom())
	suite.Require().Zero(balance.Amount.Int64())
}

//func (suite IBCTestSuite) TestClientAndConnectionSetup() {
//	suite.T().Log("initializes 2 test chains")
//	suite.coordinator = ibctesting.NewCoordinator(suite.T(), 2)
//	suite.chainA = suite.coordinator.GetChain(ibctesting.GetChainID(1))
//	suite.chainB = suite.coordinator.GetChain(ibctesting.GetChainID(2))
//
//	suite.T().Log("clientID, connectionID, channelID empty")
//	suite.path = NewIBCTestingTransferPath(suite.chainA, suite.chainB)
//	suite.coordinator.CommitNBlocks(suite.chainA, 2)
//	suite.coordinator.CommitNBlocks(suite.chainB, 2)
//
//	suite.coordinator.SetupClients(suite.path)
//	suite.Assert().Equal("07-tendermint-0", suite.path.EndpointA.ClientID)
//	suite.Assert().Equal("07-tendermint-0", suite.path.EndpointB.ClientID)
//
//	suite.coordinator.SetupConnections(suite.path)
//	suite.Assert().Equal("connection-0", suite.path.EndpointA.ConnectionID)
//	suite.Assert().Equal("connection-0", suite.path.EndpointB.ConnectionID)
//
//	suite.T().Log("After connections are set up, client IDs should increment.")
//	suite.Assert().Equal("07-tendermint-1", suite.path.EndpointA.ClientID)
//	suite.Assert().Equal("07-tendermint-1", suite.path.EndpointB.ClientID)
//
//	err := suite.coordinator.ChanOpenInitOnBothChains(suite.path)
//	suite.Assert().Equal("channel-0", suite.path.EndpointA.ChannelID)
//	suite.Assert().Equal("channel-0", suite.path.EndpointB.ChannelID)
//	suite.Require().NoError(err)
//	suite.T().Log("clientID, connectionID, channelID filled")
//}
//
//func (suite IBCTestSuite) TestInitialization() {
//	suite.SetupTest()
//
//	var err error = suite.coordinator.ConnOpenInitOnBothChains(suite.path)
//	suite.Assert().Equal("channel-0", suite.path.EndpointA.ChannelID)
//	suite.Assert().Equal("07-tendermint-0", suite.path.EndpointA.ClientID)
//	suite.Assert().Equal("07-tendermint-0", suite.path.EndpointB.ClientID)
//	suite.Require().NoError(err)
//}
//
//func (suite IBCTestSuite) TestClient_BeginBlocker() {
//	// set localhost client
//	setLocalHostClient := func() {
//		revision := ibcclienttypes.ParseChainID(suite.chainA.GetContext().ChainID())
//		localHostClient := localhosttypes.NewClientState(
//			suite.chainA.GetContext().ChainID(),
//			ibcclienttypes.NewHeight(revision, uint64(suite.chainA.GetContext().BlockHeight())),
//		)
//		suite.chainA.App.GetIBCKeeper().ClientKeeper.SetClientState(
//			suite.chainA.GetContext(), ibcexported.Localhost, localHostClient)
//	}
//	setLocalHostClient()
//
//	prevHeight := ibcclienttypes.GetSelfHeight(suite.chainA.GetContext())
//
//	localHostClient := suite.chainA.GetClientState(ibcexported.Localhost)
//	suite.Require().Equal(prevHeight, localHostClient.GetLatestHeight())
//
//	for i := 0; i < 10; i++ {
//		// increment height
//		suite.coordinator.CommitBlock(suite.chainA, suite.chainB)
//
//		suite.Require().NotPanics(func() {
//			ibcclient.BeginBlocker(
//				suite.chainA.GetContext(), suite.chainA.App.GetIBCKeeper().ClientKeeper)
//		}, "BeginBlocker shouldn't panic")
//
//		localHostClient = suite.chainA.GetClientState(ibcexported.Localhost)
//		suite.Require().Equal(prevHeight.Increment(), localHostClient.GetLatestHeight())
//		prevHeight = localHostClient.GetLatestHeight().(ibcclienttypes.Height)
//	}
//}
//
//func NewPacket(
//	path *ibctesting.Path,
//	sender string, receiver string,
//	coin sdk.Coin,
//	timeoutHeight ibcclienttypes.Height,
//) channeltypes.Packet {
//	transfer := transfertypes.NewFungibleTokenPacketData(
//		coin.Denom, coin.Amount.String(), sender, receiver)
//	bz := transfertypes.ModuleCdc.MustMarshalJSON(&transfer)
//	packet := channeltypes.Packet{
//		Data:               bz,
//		Sequence:           1,
//		SourcePort:         path.EndpointA.ChannelConfig.PortID,
//		SourceChannel:      path.EndpointA.ChannelID,
//		DestinationPort:    path.EndpointB.ChannelConfig.PortID,
//		DestinationChannel: path.EndpointB.ChannelID,
//		TimeoutHeight:      timeoutHeight,
//		TimeoutTimestamp:   0}
//	return packet
//}
//
//func (suite IBCTestSuite) TestSendPacketRecvPacket() {
//	t := suite.T()
//	suite.SetupTest()
//
//	t.Log("create packet")
//	sender := testutil.AccAddress().String()
//	receiver := testutil.AccAddress().String()
//	coin := sdk.NewInt64Coin("unibi", 1000)
//	timeoutHeight := ibcclienttypes.NewHeight(1000, 1000)
//	path := suite.path
//	packet1 := NewPacket(path, sender, receiver, coin, timeoutHeight)
//
//	var err error
//
//	t.Log("Send packet from A to B")
//	err = path.EndpointA.SendPacket(packet1)
//	suite.Assert().NoError(err)
//
//	t.Log("receive on endpointB")
//	err = path.EndpointB.RecvPacket(packet1)
//	suite.Assert().NoError(err)
//
//	t.Log("acknowledge the receipt of the packet")
//	ack := ibcmock.MockAcknowledgement
//	err = path.EndpointB.AcknowledgePacket(packet1, ack.Acknowledgement())
//	suite.Assert().NoError(err)
//
//	t.Log("updating the client should not cause any problems.")
//	err = path.EndpointB.UpdateClient()
//	suite.Assert().NoError(err)
//}
//
//func (suite IBCTestSuite) TestConsensusAfterClientUpgrade() {
//	// TODO test: https://github.com/NibiruChain/nibiru/issues/581
//}
