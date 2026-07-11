package keeper_test

import (
	"context"
	"testing"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	transfertypes "github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/apps/transfer/types"
	channelkeeper "github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/core/04-channel/keeper"
	"github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/core/04-channel/types"
	"github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/core/exported"
	ibctesting "github.com/NibiruChain/nibiru/v2/lib/ibc-go/testing"
	"github.com/NibiruChain/nibiru/v2/lib/ibc-go/testing/simapp"
)

type directChannelFixture struct {
	ctx    sdk.Context
	keeper channelkeeper.Keeper
}

func newDirectChannelFixture(t *testing.T) directChannelFixture {
	t.Helper()

	key := sdk.NewKVStoreKey(exported.StoreKey)
	tkey := sdk.NewTransientStoreKey("transient_test")
	testCtx := testutil.DefaultContextWithDB(t, key, tkey)
	ctx := testCtx.Ctx.WithBlockHeader(tmproto.Header{
		ChainID: ibctesting.GetChainID(1),
		Height:  1,
	})

	encCfg := simapp.MakeTestEncodingConfig()
	return directChannelFixture{
		ctx:    ctx,
		keeper: channelkeeper.NewKeeper(encCfg.Marshaler, key, nil, nil, nil, nil),
	}
}

func (f directChannelFixture) grpcCtx() context.Context {
	return sdk.WrapSDKContext(f.ctx)
}

func TestDirectGetAllChannelsWithPortPrefix(t *testing.T) {
	const (
		secondChannelID        = "channel-1"
		differentChannelPortID = "different-portid"
	)

	allChannels := []types.IdentifiedChannel{
		types.NewIdentifiedChannel(transfertypes.PortID, ibctesting.FirstChannelID, types.Channel{}),
		types.NewIdentifiedChannel(differentChannelPortID, secondChannelID, types.Channel{}),
	}

	tests := []struct {
		name             string
		prefix           string
		allChannels      []types.IdentifiedChannel
		expectedChannels []types.IdentifiedChannel
	}{
		{
			name:             "transfer channel is retrieved with prefix",
			prefix:           "tra",
			allChannels:      allChannels,
			expectedChannels: []types.IdentifiedChannel{types.NewIdentifiedChannel(transfertypes.PortID, ibctesting.FirstChannelID, types.Channel{})},
		},
		{
			name:             "matches port with full name as prefix",
			prefix:           transfertypes.PortID,
			allChannels:      allChannels,
			expectedChannels: []types.IdentifiedChannel{types.NewIdentifiedChannel(transfertypes.PortID, ibctesting.FirstChannelID, types.Channel{})},
		},
		{
			name:             "no ports match prefix",
			prefix:           "wont-match-anything",
			allChannels:      allChannels,
			expectedChannels: nil,
		},
		{
			name:             "empty prefix matches everything",
			prefix:           "",
			allChannels:      allChannels,
			expectedChannels: allChannels,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fixture := newDirectChannelFixture(t)
			for _, ch := range tc.allChannels {
				fixture.keeper.SetChannel(fixture.ctx, ch.PortId, ch.ChannelId, types.Channel{})
			}

			actualChannels := fixture.keeper.GetAllChannelsWithPortPrefix(fixture.ctx, tc.prefix)
			require.True(t, containsAll(tc.expectedChannels, actualChannels))
		})
	}
}

func TestDirectGetAllChannels(t *testing.T) {
	fixture := newDirectChannelFixture(t)

	channel0 := types.NewChannel(
		types.OPEN, types.UNORDERED,
		types.NewCounterparty(ibctesting.MockPort, "channel-0"),
		[]string{ibctesting.FirstConnectionID}, ibctesting.DefaultChannelVersion,
	)
	channel1 := types.NewChannel(
		types.OPEN, types.ORDERED,
		types.NewCounterparty(ibctesting.MockPort, "channel-1"),
		[]string{ibctesting.FirstConnectionID}, ibctesting.DefaultChannelVersion,
	)
	channel2 := types.NewChannel(
		types.INIT, types.UNORDERED,
		types.NewCounterparty(ibctesting.MockPort, ""),
		[]string{"connection-1"}, ibctesting.DefaultChannelVersion,
	)

	fixture.keeper.SetChannel(fixture.ctx, ibctesting.MockPort, "channel-0", channel0)
	fixture.keeper.SetChannel(fixture.ctx, ibctesting.MockPort, "channel-1", channel1)
	fixture.keeper.SetChannel(fixture.ctx, ibctesting.MockPort, "channel-2", channel2)

	expChannels := []types.IdentifiedChannel{
		types.NewIdentifiedChannel(ibctesting.MockPort, "channel-0", channel0),
		types.NewIdentifiedChannel(ibctesting.MockPort, "channel-1", channel1),
		types.NewIdentifiedChannel(ibctesting.MockPort, "channel-2", channel2),
	}

	channels := fixture.keeper.GetAllChannels(fixture.ctx)
	require.Len(t, channels, len(expChannels))
	require.Equal(t, expChannels, channels)
}

func TestDirectGetAllSequences(t *testing.T) {
	fixture := newDirectChannelFixture(t)

	seq1 := types.NewPacketSequence(ibctesting.MockPort, "channel-0", 1)
	seq2 := types.NewPacketSequence(ibctesting.MockPort, "channel-0", 2)
	seq3 := types.NewPacketSequence(ibctesting.MockPort, "channel-1", 3)
	expSeqs := []types.PacketSequence{seq2, seq3}

	for _, seq := range []types.PacketSequence{seq1, seq2, seq3} {
		fixture.keeper.SetNextSequenceSend(fixture.ctx, seq.PortId, seq.ChannelId, seq.Sequence)
		fixture.keeper.SetNextSequenceRecv(fixture.ctx, seq.PortId, seq.ChannelId, seq.Sequence)
		fixture.keeper.SetNextSequenceAck(fixture.ctx, seq.PortId, seq.ChannelId, seq.Sequence)
	}

	sendSeqs := fixture.keeper.GetAllPacketSendSeqs(fixture.ctx)
	recvSeqs := fixture.keeper.GetAllPacketRecvSeqs(fixture.ctx)
	ackSeqs := fixture.keeper.GetAllPacketAckSeqs(fixture.ctx)
	require.Len(t, sendSeqs, 2)
	require.Len(t, recvSeqs, 2)
	require.Len(t, ackSeqs, 2)

	require.Equal(t, expSeqs, sendSeqs)
	require.Equal(t, expSeqs, recvSeqs)
	require.Equal(t, expSeqs, ackSeqs)
}

func TestDirectGetAllPacketState(t *testing.T) {
	fixture := newDirectChannelFixture(t)

	ack1 := types.NewPacketState(ibctesting.MockPort, "channel-0", 1, []byte("ack"))
	ack2 := types.NewPacketState(ibctesting.MockPort, "channel-0", 2, []byte("ack"))
	ack2dup := types.NewPacketState(ibctesting.MockPort, "channel-0", 2, []byte("ack"))
	ack3 := types.NewPacketState(ibctesting.MockPort, "channel-1", 1, []byte("ack"))

	receipt := string([]byte{byte(1)})
	rec1 := types.NewPacketState(ibctesting.MockPort, "channel-0", 1, []byte(receipt))
	rec2 := types.NewPacketState(ibctesting.MockPort, "channel-0", 2, []byte(receipt))
	rec3 := types.NewPacketState(ibctesting.MockPort, "channel-1", 1, []byte(receipt))
	rec4 := types.NewPacketState(ibctesting.MockPort, "channel-1", 2, []byte(receipt))

	comm1 := types.NewPacketState(ibctesting.MockPort, "channel-0", 1, []byte("hash"))
	comm2 := types.NewPacketState(ibctesting.MockPort, "channel-0", 2, []byte("hash"))
	comm3 := types.NewPacketState(ibctesting.MockPort, "channel-1", 1, []byte("hash"))
	comm4 := types.NewPacketState(ibctesting.MockPort, "channel-1", 2, []byte("hash"))

	expAcks := []types.PacketState{ack1, ack2, ack3}
	expReceipts := []types.PacketState{rec1, rec2, rec3, rec4}
	expCommitments := []types.PacketState{comm1, comm2, comm3, comm4}

	for _, ack := range []types.PacketState{ack1, ack2, ack2dup, ack3} {
		fixture.keeper.SetPacketAcknowledgement(fixture.ctx, ack.PortId, ack.ChannelId, ack.Sequence, ack.Data)
	}
	for _, rec := range expReceipts {
		fixture.keeper.SetPacketReceipt(fixture.ctx, rec.PortId, rec.ChannelId, rec.Sequence)
	}
	for _, comm := range expCommitments {
		fixture.keeper.SetPacketCommitment(fixture.ctx, comm.PortId, comm.ChannelId, comm.Sequence, comm.Data)
	}

	acks := fixture.keeper.GetAllPacketAcks(fixture.ctx)
	receipts := fixture.keeper.GetAllPacketReceipts(fixture.ctx)
	commitments := fixture.keeper.GetAllPacketCommitments(fixture.ctx)

	require.Len(t, acks, len(expAcks))
	require.Len(t, commitments, len(expCommitments))
	require.Len(t, receipts, len(expReceipts))
	require.Equal(t, expAcks, acks)
	require.Equal(t, expReceipts, receipts)
	require.Equal(t, expCommitments, commitments)
}

func TestDirectSetSequence(t *testing.T) {
	fixture := newDirectChannelFixture(t)
	one := uint64(1)

	fixture.keeper.SetNextSequenceSend(fixture.ctx, ibctesting.MockPort, ibctesting.FirstChannelID, one)
	fixture.keeper.SetNextSequenceRecv(fixture.ctx, ibctesting.MockPort, ibctesting.FirstChannelID, one)
	fixture.keeper.SetNextSequenceAck(fixture.ctx, ibctesting.MockPort, ibctesting.FirstChannelID, one)

	seq, found := fixture.keeper.GetNextSequenceSend(fixture.ctx, ibctesting.MockPort, ibctesting.FirstChannelID)
	require.True(t, found)
	require.Equal(t, one, seq)

	seq, found = fixture.keeper.GetNextSequenceRecv(fixture.ctx, ibctesting.MockPort, ibctesting.FirstChannelID)
	require.True(t, found)
	require.Equal(t, one, seq)

	seq, found = fixture.keeper.GetNextSequenceAck(fixture.ctx, ibctesting.MockPort, ibctesting.FirstChannelID)
	require.True(t, found)
	require.Equal(t, one, seq)

	nextSeqSend, nextSeqRecv, nextSeqAck := uint64(10), uint64(11), uint64(12)
	fixture.keeper.SetNextSequenceSend(fixture.ctx, ibctesting.MockPort, ibctesting.FirstChannelID, nextSeqSend)
	fixture.keeper.SetNextSequenceRecv(fixture.ctx, ibctesting.MockPort, ibctesting.FirstChannelID, nextSeqRecv)
	fixture.keeper.SetNextSequenceAck(fixture.ctx, ibctesting.MockPort, ibctesting.FirstChannelID, nextSeqAck)

	storedNextSeqSend, found := fixture.keeper.GetNextSequenceSend(fixture.ctx, ibctesting.MockPort, ibctesting.FirstChannelID)
	require.True(t, found)
	require.Equal(t, nextSeqSend, storedNextSeqSend)

	storedNextSeqRecv, found := fixture.keeper.GetNextSequenceRecv(fixture.ctx, ibctesting.MockPort, ibctesting.FirstChannelID)
	require.True(t, found)
	require.Equal(t, nextSeqRecv, storedNextSeqRecv)

	storedNextSeqAck, found := fixture.keeper.GetNextSequenceAck(fixture.ctx, ibctesting.MockPort, ibctesting.FirstChannelID)
	require.True(t, found)
	require.Equal(t, nextSeqAck, storedNextSeqAck)
}

func TestDirectGetAllPacketCommitmentsAtChannel(t *testing.T) {
	fixture := newDirectChannelFixture(t)

	expectedSeqs := make(map[uint64]bool)
	hash := []byte("commitment")
	seq := uint64(15)
	maxSeq := uint64(25)
	require.Greater(t, maxSeq, seq)

	for i := uint64(1); i < seq; i++ {
		fixture.keeper.SetPacketCommitment(fixture.ctx, ibctesting.MockPort, "channel-0", i, hash)
		expectedSeqs[i] = true
	}
	for i := seq; i < maxSeq; i += 2 {
		fixture.keeper.SetPacketCommitment(fixture.ctx, ibctesting.MockPort, "channel-0", i, hash)
		expectedSeqs[i] = true
	}
	fixture.keeper.SetPacketCommitment(fixture.ctx, ibctesting.MockPort, "channel-1", maxSeq+1, hash)

	commitments := fixture.keeper.GetAllPacketCommitmentsAtChannel(fixture.ctx, ibctesting.MockPort, "channel-0")
	require.Equal(t, len(expectedSeqs), len(commitments))
	require.NotEmpty(t, commitments)

	for _, packet := range commitments {
		require.True(t, expectedSeqs[packet.Sequence])
		require.Equal(t, ibctesting.MockPort, packet.PortId)
		require.Equal(t, "channel-0", packet.ChannelId)
		require.Equal(t, hash, packet.Data)
		expectedSeqs[packet.Sequence] = false
	}
}

func TestDirectSetPacketAcknowledgement(t *testing.T) {
	fixture := newDirectChannelFixture(t)
	seq := uint64(10)

	storedAckHash, found := fixture.keeper.GetPacketAcknowledgement(fixture.ctx, ibctesting.MockPort, ibctesting.FirstChannelID, seq)
	require.False(t, found)
	require.Nil(t, storedAckHash)

	ackHash := []byte("ackhash")
	fixture.keeper.SetPacketAcknowledgement(fixture.ctx, ibctesting.MockPort, ibctesting.FirstChannelID, seq, ackHash)

	storedAckHash, found = fixture.keeper.GetPacketAcknowledgement(fixture.ctx, ibctesting.MockPort, ibctesting.FirstChannelID, seq)
	require.True(t, found)
	require.Equal(t, ackHash, storedAckHash)
	require.True(t, fixture.keeper.HasPacketAcknowledgement(fixture.ctx, ibctesting.MockPort, ibctesting.FirstChannelID, seq))
}

func TestDirectQueryPacketReceipt(t *testing.T) {
	tests := []struct {
		name        string
		req         *types.QueryPacketReceiptRequest
		setReceipt  bool
		expPass     bool
		expReceived bool
	}{
		{
			name:    "empty request",
			req:     nil,
			expPass: false,
		},
		{
			name: "invalid port ID",
			req: &types.QueryPacketReceiptRequest{
				PortId:    "",
				ChannelId: "test-channel-id",
				Sequence:  1,
			},
			expPass: false,
		},
		{
			name: "invalid channel ID",
			req: &types.QueryPacketReceiptRequest{
				PortId:    "test-port-id",
				ChannelId: "",
				Sequence:  1,
			},
			expPass: false,
		},
		{
			name: "invalid sequence",
			req: &types.QueryPacketReceiptRequest{
				PortId:    "test-port-id",
				ChannelId: "test-channel-id",
				Sequence:  0,
			},
			expPass: false,
		},
		{
			name: "success: receipt not found",
			req: &types.QueryPacketReceiptRequest{
				PortId:    ibctesting.MockPort,
				ChannelId: ibctesting.FirstChannelID,
				Sequence:  3,
			},
			setReceipt:  true,
			expPass:     true,
			expReceived: false,
		},
		{
			name: "success: receipt found",
			req: &types.QueryPacketReceiptRequest{
				PortId:    ibctesting.MockPort,
				ChannelId: ibctesting.FirstChannelID,
				Sequence:  1,
			},
			setReceipt:  true,
			expPass:     true,
			expReceived: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fixture := newDirectChannelFixture(t)
			if tc.setReceipt {
				fixture.keeper.SetPacketReceipt(fixture.ctx, ibctesting.MockPort, ibctesting.FirstChannelID, 1)
			}

			res, err := fixture.keeper.PacketReceipt(fixture.grpcCtx(), tc.req)
			if tc.expPass {
				require.NoError(t, err)
				require.NotNil(t, res)
				require.Equal(t, tc.expReceived, res.Received)
			} else {
				require.Error(t, err)
			}
		})
	}
}
