package ibctesting

import (
	"bytes"
	"testing"
	"time"

	dbm "github.com/cometbft/cometbft-db"
	abci "github.com/cometbft/cometbft/abci/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmttypes "github.com/cometbft/cometbft/types"
	cryptotypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/crypto/types"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	clienttypes "github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/core/02-client/types"
	connectiontypes "github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/core/03-connection/types"
	host "github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/core/24-host"
	"github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/core/exported"
	ibctm "github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/light-clients/07-tendermint"
	"github.com/NibiruChain/nibiru/v2/lib/ibc-go/testing/simapp"
)

type ConnectionCheckpoint struct {
	currentTime time.Time
	chainAID    string
	chainBID    string
	chainA      chainCheckpoint
	chainB      chainCheckpoint
	endpointA   endpointCheckpoint
	endpointB   endpointCheckpoint
}

type chainCheckpoint struct {
	db                 dbm.DB
	chainID            string
	lastHeader         *ibctm.Header
	currentHeader      tmproto.Header
	vals               *cmttypes.ValidatorSet
	nextVals           *cmttypes.ValidatorSet
	signers            map[string]cmttypes.PrivValidator
	senderPrivKeys     []cryptotypes.PrivKey
	senderAccountIndex int
}

type endpointCheckpoint struct {
	clientID     string
	connectionID string
	channelID    string
}

func CaptureConnectionCheckpoint(t *testing.T, coord *Coordinator, path *Path) *ConnectionCheckpoint {
	t.Helper()

	return &ConnectionCheckpoint{
		currentTime: coord.CurrentTime,
		chainAID:    path.EndpointA.Chain.ChainID,
		chainBID:    path.EndpointB.Chain.ChainID,
		chainA:      captureChainCheckpoint(t, path.EndpointA.Chain),
		chainB:      captureChainCheckpoint(t, path.EndpointB.Chain),
		endpointA:   captureEndpointCheckpoint(path.EndpointA),
		endpointB:   captureEndpointCheckpoint(path.EndpointB),
	}
}

func (cp *ConnectionCheckpoint) Restore(t *testing.T) (*Coordinator, *Path) {
	t.Helper()

	coord := &Coordinator{
		T:           t,
		CurrentTime: cp.currentTime,
		Chains:      map[string]*TestChain{},
	}

	chainA := cp.chainA.restore(t, coord)
	chainB := cp.chainB.restore(t, coord)
	coord.Chains[chainA.ChainID] = chainA
	coord.Chains[chainB.ChainID] = chainB

	path := NewPath(chainA, chainB)
	restoreEndpoint(path.EndpointA, cp.endpointA)
	restoreEndpoint(path.EndpointB, cp.endpointB)

	return coord, path
}

func captureChainCheckpoint(t *testing.T, chain *TestChain) chainCheckpoint {
	t.Helper()

	app := chain.GetSimApp()
	senderPrivKeys := make([]cryptotypes.PrivKey, len(chain.SenderAccounts))
	senderAccountIndex := 0
	for i, sender := range chain.SenderAccounts {
		senderPrivKeys[i] = sender.SenderPrivKey
		if bytes.Equal(sender.SenderPrivKey.PubKey().Address(), chain.SenderPrivKey.PubKey().Address()) {
			senderAccountIndex = i
		}
	}

	return chainCheckpoint{
		db:                 cloneDB(t, app.DB()),
		chainID:            chain.ChainID,
		lastHeader:         copyTMHeader(chain.LastHeader),
		currentHeader:      chain.CurrentHeader,
		vals:               chain.Vals.Copy(),
		nextVals:           chain.NextVals.Copy(),
		signers:            copySigners(chain.Signers),
		senderPrivKeys:     senderPrivKeys,
		senderAccountIndex: senderAccountIndex,
	}
}

func (cp chainCheckpoint) restore(t *testing.T, coord *Coordinator) *TestChain {
	t.Helper()

	db := cloneDB(t, cp.db)
	app := setupTestingAppFromDB(db, cp.chainID)
	simApp, ok := app.(*simapp.SimApp)
	require.True(t, ok)

	senderAccounts := make([]SenderAccount, len(cp.senderPrivKeys))
	ctx := app.GetBaseApp().NewUncachedContext(false, cp.currentHeader)
	for i, privKey := range cp.senderPrivKeys {
		addr := sdk.AccAddress(privKey.PubKey().Address())
		account := simApp.AccountKeeper.GetAccount(ctx, addr)
		require.NotNil(t, account)
		senderAccounts[i] = SenderAccount{
			SenderPrivKey: privKey,
			SenderAccount: account,
		}
	}

	chain := &TestChain{
		T:              t,
		Coordinator:    coord,
		ChainID:        cp.chainID,
		App:            app,
		CurrentHeader:  cp.currentHeader,
		LastHeader:     copyTMHeader(cp.lastHeader),
		QueryServer:    app.GetIBCKeeper(),
		TxConfig:       app.GetTxConfig(),
		Codec:          app.AppCodec(),
		Vals:           cp.vals.Copy(),
		NextVals:       cp.nextVals.Copy(),
		Signers:        copySigners(cp.signers),
		SenderPrivKey:  senderAccounts[cp.senderAccountIndex].SenderPrivKey,
		SenderAccount:  senderAccounts[cp.senderAccountIndex].SenderAccount,
		SenderAccounts: senderAccounts,
	}

	chain.App.BeginBlock(abciBeginBlock(chain.CurrentHeader))
	return chain
}

func captureEndpointCheckpoint(endpoint *Endpoint) endpointCheckpoint {
	return endpointCheckpoint{
		clientID:     endpoint.ClientID,
		connectionID: endpoint.ConnectionID,
		channelID:    endpoint.ChannelID,
	}
}

func restoreEndpoint(endpoint *Endpoint, cp endpointCheckpoint) {
	endpoint.ClientID = cp.clientID
	endpoint.ConnectionID = cp.connectionID
	endpoint.ChannelID = cp.channelID
}

func cloneDB(t *testing.T, src dbm.DB) dbm.DB {
	t.Helper()

	dst := dbm.NewMemDB()
	itr, err := src.Iterator(nil, nil)
	require.NoError(t, err)
	defer itr.Close()

	batch := dst.NewBatch()
	defer batch.Close()
	for ; itr.Valid(); itr.Next() {
		require.NoError(t, batch.Set(bytes.Clone(itr.Key()), bytes.Clone(itr.Value())))
	}
	require.NoError(t, itr.Error())
	require.NoError(t, batch.WriteSync())
	return dst
}

func copySigners(signers map[string]cmttypes.PrivValidator) map[string]cmttypes.PrivValidator {
	out := make(map[string]cmttypes.PrivValidator, len(signers))
	for addr, signer := range signers {
		out[addr] = signer
	}
	return out
}

func copyTMHeader(header *ibctm.Header) *ibctm.Header {
	if header == nil {
		return nil
	}
	copied := *header
	return &copied
}

func abciBeginBlock(header tmproto.Header) abci.RequestBeginBlock {
	return abci.RequestBeginBlock{Header: header}
}

func (cp *ConnectionCheckpoint) AssertConnectionState(t *testing.T) {
	t.Helper()

	_, path := cp.Restore(t)
	assertEndpointConnectionState(t, path.EndpointA)
	assertEndpointConnectionState(t, path.EndpointB)

	proof, height := path.EndpointA.QueryProof(host.ConnectionKey(path.EndpointA.ConnectionID))
	require.NotEmpty(t, proof)
	require.NotEqual(t, clienttypes.Height{}, height)
}

func assertEndpointConnectionState(t *testing.T, endpoint *Endpoint) {
	t.Helper()

	connection, found := endpoint.Chain.App.GetIBCKeeper().ConnectionKeeper.GetConnection(endpoint.Chain.GetContext(), endpoint.ConnectionID)
	require.True(t, found)
	require.Equal(t, connectiontypes.OPEN, connection.State)

	clientState, found := endpoint.Chain.App.GetIBCKeeper().ClientKeeper.GetClientState(endpoint.Chain.GetContext(), endpoint.ClientID)
	require.True(t, found)
	require.NotNil(t, clientState)

	require.NotNil(t, endpoint.Chain.GetPortCapability(MockPort))

	require.Equal(t, endpoint.Chain.App.LastBlockHeight()+1, endpoint.Chain.CurrentHeader.Height)
	require.Equal(t, exported.Tendermint, clientState.ClientType())
}
