package app_test

import (
	"encoding/json"
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	transfertypes "github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/apps/transfer/types"
	clienttypes "github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/core/02-client/types"
	ibctesting "github.com/NibiruChain/nibiru/v2/lib/ibc-go/testing"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testapp"
)

// init changes the value of 'DefaultTestingAppInit' to use custom initialization.
func init() {
	ibctesting.DefaultTestingAppInit = newNibiruTestApp
}

/*
SetupNibiruTestingApp returns the TestingApp and default genesis state used to

	initialize the testing app.
*/
func newNibiruTestApp() (ibctesting.TestingApp, map[string]json.RawMessage) {
	return testapp.NewNibiruTestApp(app.GenesisState{})
}

// IBCTestSuite is a testing suite to test keeper functions.
type IBCTestSuite struct {
	suite.Suite

	coordinator *ibctesting.Coordinator

	// testing chains used for convenience and readability
	chainA *ibctesting.TestChain
	chainB *ibctesting.TestChain
	chainC *ibctesting.TestChain
}

// TestIBCTestSuite runs all the tests within this package.
func TestIBCTestSuite(t *testing.T) {
	suite.Run(t, new(IBCTestSuite))
}

func TestIBCFeeModuleRemoved(t *testing.T) {
	_, registered := app.ModuleBasics["feeibc"]
	if registered {
		t.Fatal("feeibc module must not be registered")
	}
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

	amount, ok := sdkmath.NewIntFromString("9223372036854775808") // 2^63 (one above int64)
	suite.Require().True(ok)
	coinToSendToB := sdk.NewCoin(sdk.DefaultBondDenom, amount)

	// send from chainA to chainB
	msg := transfertypes.NewMsgTransfer(
		path.EndpointA.ChannelConfig.PortID,
		path.EndpointA.ChannelID,
		coinToSendToB,
		suite.chainA.SenderAccount.GetAddress().String(),
		suite.chainB.SenderAccount.GetAddress().String(),
		suite.chainB.GetTimeoutHeight(),
		0,
		"",
	)
	res, err := suite.chainA.SendMsgs(msg)
	suite.Require().NoError(err) // message committed

	packet, err := ibctesting.ParsePacketFromEvents(res.GetEvents())
	suite.Require().NoError(err)

	// relay send
	err = path.RelayPacket(packet)
	suite.Require().NoError(err) // relay committed

	// check that voucher exists on chain B
	voucherDenomTrace := transfertypes.ParseDenomTrace(transfertypes.GetPrefixedDenom(packet.GetDestPort(), packet.GetDestChannel(), sdk.DefaultBondDenom))
	chainBApp, ok := suite.chainB.App.(*app.NibiruApp)
	suite.Require().True(ok)

	balance := chainBApp.BankKeeper.GetBalance(suite.chainB.GetContext(), suite.chainB.SenderAccount.GetAddress(), voucherDenomTrace.IBCDenom())

	coinSentFromAToB := transfertypes.GetTransferCoin(path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID, sdk.DefaultBondDenom, amount)
	suite.Require().Equal(coinSentFromAToB, balance)

	// setup between chainB to chainC
	// NOTE:
	// pathBtoC.EndpointA = endpoint on chainB
	// pathBtoC.EndpointB = endpoint on chainC
	pathBtoC := NewIBCTestingTransferPath(suite.chainB, suite.chainC)
	suite.coordinator.Setup(pathBtoC)

	// send from chainB to chainC
	msg = transfertypes.NewMsgTransfer(
		pathBtoC.EndpointA.ChannelConfig.PortID,
		pathBtoC.EndpointA.ChannelID,
		coinSentFromAToB,
		suite.chainB.SenderAccount.GetAddress().String(),
		suite.chainC.SenderAccount.GetAddress().String(),
		suite.chainC.GetTimeoutHeight(),
		0,
		"",
	)
	res, err = suite.chainB.SendMsgs(msg)
	suite.Require().NoError(err) // message committed

	packet, err = ibctesting.ParsePacketFromEvents(res.GetEvents())
	suite.Require().NoError(err)

	err = pathBtoC.RelayPacket(packet)
	suite.Require().NoError(err) // relay committed

	// NOTE: fungible token is prefixed with the full trace in order to verify the packet commitment
	fullDenomPath := transfertypes.GetPrefixedDenom(pathBtoC.EndpointB.ChannelConfig.PortID, pathBtoC.EndpointB.ChannelID, voucherDenomTrace.GetFullDenomPath())

	chainCApp, ok := suite.chainC.App.(*app.NibiruApp)
	suite.Require().True(ok)
	coinSentFromBToC := sdk.NewCoin(transfertypes.ParseDenomTrace(fullDenomPath).IBCDenom(), amount)
	balance = chainCApp.BankKeeper.GetBalance(suite.chainC.GetContext(), suite.chainC.SenderAccount.GetAddress(), coinSentFromBToC.Denom)

	// check that the balance is updated on chainC
	suite.Require().Equal(coinSentFromBToC, balance)

	// check that balance on chain B is empty
	balance = chainBApp.BankKeeper.GetBalance(suite.chainB.GetContext(), suite.chainB.SenderAccount.GetAddress(), coinSentFromBToC.Denom)
	suite.Require().Zero(balance.Amount.Int64())

	// send from chainC back to chainB
	msg = transfertypes.NewMsgTransfer(
		pathBtoC.EndpointB.ChannelConfig.PortID,
		pathBtoC.EndpointB.ChannelID,
		coinSentFromBToC,
		suite.chainC.SenderAccount.GetAddress().String(),
		suite.chainB.SenderAccount.GetAddress().String(),
		suite.chainB.GetTimeoutHeight(),
		0,
		"",
	)
	res, err = suite.chainC.SendMsgs(msg)
	suite.Require().NoError(err) // message committed

	packet, err = ibctesting.ParsePacketFromEvents(res.GetEvents())
	suite.Require().NoError(err)

	err = pathBtoC.RelayPacket(packet)
	suite.Require().NoError(err) // relay committed

	balance = chainBApp.BankKeeper.GetBalance(suite.chainB.GetContext(), suite.chainB.SenderAccount.GetAddress(), coinSentFromAToB.Denom)

	// check that the balance on chainA returned back to the original state
	suite.Require().Equal(coinSentFromAToB, balance)

	// check that module account escrow address is empty
	escrowAddress := transfertypes.GetEscrowAddress(packet.GetDestPort(), packet.GetDestChannel())
	balance = chainBApp.BankKeeper.GetBalance(suite.chainB.GetContext(), escrowAddress, sdk.DefaultBondDenom)
	suite.Require().Equal(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.ZeroInt()), balance)

	// check that balance on chain B is empty
	balance = chainCApp.BankKeeper.GetBalance(suite.chainC.GetContext(), suite.chainC.SenderAccount.GetAddress(), voucherDenomTrace.IBCDenom())
	suite.Require().Zero(balance.Amount.Int64())
}

func (suite *IBCTestSuite) TestPermissionlessPacketRelay() {
	path := NewIBCTestingTransferPath(suite.chainA, suite.chainB)
	suite.coordinator.Setup(path)

	sender := suite.chainA.SenderAccount.GetAddress()
	receiver := suite.chainB.SenderAccount.GetAddress()
	coin := sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(1_000_000))

	msg := transfertypes.NewMsgTransfer(
		path.EndpointA.ChannelConfig.PortID,
		path.EndpointA.ChannelID,
		coin,
		sender.String(),
		receiver.String(),
		suite.chainB.GetTimeoutHeight(),
		0,
		"",
	)
	res, err := suite.chainA.SendMsgs(msg)
	suite.Require().NoError(err)

	packet, err := ibctesting.ParsePacketFromEvents(res.GetEvents())
	suite.Require().NoError(err)

	// Relay receive and acknowledgement transactions with accounts unrelated to
	// the packet sender or receiver.
	relayA := suite.chainA.SenderAccounts[1]
	suite.chainA.SenderAccount = relayA.SenderAccount
	suite.chainA.SenderPrivKey = relayA.SenderPrivKey
	relayB := suite.chainB.SenderAccounts[1]
	suite.chainB.SenderAccount = relayB.SenderAccount
	suite.chainB.SenderPrivKey = relayB.SenderPrivKey

	suite.Require().NoError(path.RelayPacket(packet))

	commitment := suite.chainA.App.GetIBCKeeper().ChannelKeeper.GetPacketCommitment(
		suite.chainA.GetContext(),
		packet.GetSourcePort(),
		packet.GetSourceChannel(),
		packet.GetSequence(),
	)
	suite.Require().Empty(commitment)

	denomTrace := transfertypes.ParseDenomTrace(
		transfertypes.GetPrefixedDenom(
			packet.GetDestPort(),
			packet.GetDestChannel(),
			sdk.DefaultBondDenom,
		),
	)
	chainBApp, ok := suite.chainB.App.(*app.NibiruApp)
	suite.Require().True(ok)
	suite.Require().Equal(
		coin.Amount,
		chainBApp.BankKeeper.GetBalance(
			suite.chainB.GetContext(),
			receiver,
			denomTrace.IBCDenom(),
		).Amount,
	)
}

func (suite *IBCTestSuite) TestPermissionlessTimeoutRelay() {
	path := NewIBCTestingTransferPath(suite.chainA, suite.chainB)
	suite.coordinator.Setup(path)

	sender := suite.chainA.SenderAccount.GetAddress()
	coin := sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(1_000_000))
	timeoutHeight := clienttypes.GetSelfHeight(suite.chainB.GetContext())
	chainAApp, ok := suite.chainA.App.(*app.NibiruApp)
	suite.Require().True(ok)
	startingBalance := chainAApp.BankKeeper.GetBalance(
		suite.chainA.GetContext(),
		sender,
		sdk.DefaultBondDenom,
	)

	msg := transfertypes.NewMsgTransfer(
		path.EndpointA.ChannelConfig.PortID,
		path.EndpointA.ChannelID,
		coin,
		sender.String(),
		suite.chainB.SenderAccount.GetAddress().String(),
		timeoutHeight,
		0,
		"",
	)
	res, err := suite.chainA.SendMsgs(msg)
	suite.Require().NoError(err)

	packet, err := ibctesting.ParsePacketFromEvents(res.GetEvents())
	suite.Require().NoError(err)
	suite.Require().NoError(path.EndpointA.UpdateClient())

	relayer := suite.chainA.SenderAccounts[1]
	suite.chainA.SenderAccount = relayer.SenderAccount
	suite.chainA.SenderPrivKey = relayer.SenderPrivKey
	suite.Require().NoError(path.EndpointA.TimeoutPacket(packet))

	commitment := suite.chainA.App.GetIBCKeeper().ChannelKeeper.GetPacketCommitment(
		suite.chainA.GetContext(),
		packet.GetSourcePort(),
		packet.GetSourceChannel(),
		packet.GetSequence(),
	)
	suite.Require().Empty(commitment)

	suite.Require().Equal(
		startingBalance,
		chainAApp.BankKeeper.GetBalance(
			suite.chainA.GetContext(),
			sender,
			sdk.DefaultBondDenom,
		),
	)
}
