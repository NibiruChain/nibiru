package wasm

import (
	"testing"

	abci "github.com/NibiruChain/nibiru/v2/bft/abci/types"
	cmtrand "github.com/NibiruChain/nibiru/v2/bft/libs/rand"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/lib/wasmvm/wvm"

	clienttypes "github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/core/02-client/types"
	channeltypes "github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/core/04-channel/types"
	ibcexported "github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/core/exported"

	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/address"
	capabilitytypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/capability/types"

	"github.com/NibiruChain/nibiru/v2/x/wasm/keeper"
	"github.com/NibiruChain/nibiru/v2/x/wasm/types"
)

func TestOnRecvPacket(t *testing.T) {
	anyRelayerAddr := sdk.AccAddress(cmtrand.Bytes(address.Len))
	anyContractIBCPkg := IBCPacketFixture(func(p *channeltypes.Packet) {
		p.DestinationPort = "wasm.nibi1w09vr7rpe2agu0kg2zlpkdckce865l3zps8mxjurxthfh3m7035qu7fze6"
	})
	myCustomEvent := sdk.NewEvent("testing")
	specs := map[string]struct {
		ibcPkg               channeltypes.Packet
		contractRsp          ibcexported.Acknowledgement
		contractOkMsgExecErr error
		expEvents            sdk.Events
		expPanic             bool
		expAck               ibcexported.Acknowledgement
	}{
		"contract returns success response": {
			ibcPkg:      anyContractIBCPkg,
			contractRsp: keeper.ContractConfirmStateAck([]byte{1}),
			expAck:      keeper.ContractConfirmStateAck([]byte{1}),
			expEvents: sdk.Events{
				myCustomEvent,
				{
					Type: "ibc_packet_received",
					Attributes: []abci.EventAttribute{
						{Key: "module", Value: "wasm"},
						{Key: "_contract_address", Value: "nibi1w09vr7rpe2agu0kg2zlpkdckce865l3zps8mxjurxthfh3m7035qu7fze6"},
						{Key: "success", Value: "true"},
					},
				},
			},
		},
		"contract returns err response": {
			ibcPkg:      anyContractIBCPkg,
			contractRsp: channeltypes.NewErrorAcknowledgement(types.ErrInvalid.Wrap("testing")),
			expAck:      channeltypes.NewErrorAcknowledgement(types.ErrInvalid.Wrap("testing")),
			expEvents: sdk.Events{
				{
					Type: "ibc_packet_received",
					Attributes: []abci.EventAttribute{
						{Key: "module", Value: "wasm"},
						{Key: "_contract_address", Value: "nibi1w09vr7rpe2agu0kg2zlpkdckce865l3zps8mxjurxthfh3m7035qu7fze6"},
						{Key: "success", Value: "false"},
					},
				},
			},
		},
		"nil considered success response": { // regression only
			ibcPkg: anyContractIBCPkg,
			expEvents: sdk.Events{
				myCustomEvent,
				{
					Type: "ibc_packet_received",
					Attributes: []abci.EventAttribute{
						{Key: "module", Value: "wasm"},
						{Key: "_contract_address", Value: "nibi1w09vr7rpe2agu0kg2zlpkdckce865l3zps8mxjurxthfh3m7035qu7fze6"},
						{Key: "success", Value: "true"},
					},
				},
			},
		},
		"unknown contract port": {
			ibcPkg: IBCPacketFixture(func(p *channeltypes.Packet) {
				p.DestinationPort = "not-a-contract-port"
			}),
			expPanic: true,
		},
		"returned messages executed with error": {
			ibcPkg:               anyContractIBCPkg,
			contractOkMsgExecErr: types.ErrInvalid.Wrap("testing"),
			expAck:               channeltypes.NewErrorAcknowledgement(types.ErrInvalid.Wrap("testing")),
			expEvents: sdk.Events{{
				Type: "ibc_packet_received",
				Attributes: []abci.EventAttribute{
					{Key: "module", Value: "wasm"},
					{Key: "_contract_address", Value: "nibi1w09vr7rpe2agu0kg2zlpkdckce865l3zps8mxjurxthfh3m7035qu7fze6"},
					{Key: "success", Value: "false"},
					{Key: "error", Value: "testing: invalid"}, // not redacted
				},
			}},
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			mock := IBCContractKeeperMock{
				OnRecvPacketFn: func(ctx sdk.Context, contractAddr sdk.AccAddress, msg wvm.IBCPacketReceiveMsg) (ibcexported.Acknowledgement, error) {
					// additional custom event to confirm event handling on state commit/ rollback
					ctx.EventManager().EmitEvent(myCustomEvent)
					return spec.contractRsp, spec.contractOkMsgExecErr
				},
			}
			h := NewIBCHandler(mock, nil, nil)
			em := &sdk.EventManager{}
			ctx := sdk.Context{}.WithEventManager(em)
			if spec.expPanic {
				require.Panics(t, func() {
					_ = h.OnRecvPacket(ctx, spec.ibcPkg, anyRelayerAddr)
				})
				return
			}
			gotAck := h.OnRecvPacket(ctx, spec.ibcPkg, anyRelayerAddr)
			assert.Equal(t, spec.expAck, gotAck)
			assert.Equal(t, spec.expEvents, em.Events())
		})
	}
}

// TestOnChanOpenInitEmptyVersion covers the version fallback in
// OnChanOpenInit without the IBC test harness, which Nibiru skips. See
// TestOnChanOpenInitVersion for the integration variant.
func TestOnChanOpenInitEmptyVersion(t *testing.T) {
	const contractPort = "wasm.nibi1w09vr7rpe2agu0kg2zlpkdckce865l3zps8mxjurxthfh3m7035qu7fze6"
	specs := map[string]struct {
		startVersion    string
		contractVersion string
		expVersion      string
		expErr          bool
	}{
		"contract version wins": {
			startVersion:    "v1",
			contractVersion: "v2",
			expVersion:      "v2",
		},
		"proposed version when contract returns none": {
			startVersion: "v1",
			expVersion:   "v1",
		},
		"contract version when none proposed": {
			contractVersion: "v2",
			expVersion:      "v2",
		},
		"empty versions should fail": {
			expErr: true,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			var claimed bool
			mock := IBCContractKeeperMock{
				OnOpenChannelFn: func(ctx sdk.Context, contractAddr sdk.AccAddress, msg wvm.IBCChannelOpenMsg) (string, error) {
					return spec.contractVersion, nil
				},
				ClaimCapabilityFn: func(ctx sdk.Context, cap *capabilitytypes.Capability, name string) error {
					claimed = true
					return nil
				},
			}
			h := NewIBCHandler(mock, nil, nil)
			gotVersion, gotErr := h.OnChanOpenInit(
				sdk.Context{},
				channeltypes.UNORDERED,
				[]string{"connection-0"},
				contractPort,
				"channel-1",
				&capabilitytypes.Capability{},
				channeltypes.NewCounterparty("transfer", ""),
				spec.startVersion,
			)
			if spec.expErr {
				require.ErrorIs(t, gotErr, types.ErrEmpty)
				assert.False(t, claimed)
				return
			}
			require.NoError(t, gotErr)
			assert.Equal(t, spec.expVersion, gotVersion)
			assert.True(t, claimed)
		})
	}
}

func TestMapToWasmVMIBCPacket(t *testing.T) {
	var myTimestamp uint64 = 1
	specs := map[string]struct {
		src channeltypes.Packet
		exp wvm.IBCPacket
	}{
		"with height timeout": {
			src: IBCPacketFixture(),
			exp: wvm.IBCPacket{
				Data:     []byte("myData"),
				Src:      wvm.IBCEndpoint{PortID: "srcPort", ChannelID: "channel-1"},
				Dest:     wvm.IBCEndpoint{PortID: "destPort", ChannelID: "channel-2"},
				Sequence: 1,
				Timeout:  wvm.IBCTimeout{Block: &wvm.IBCTimeoutBlock{Height: 1, Revision: 2}},
			},
		},
		"with time timeout": {
			src: IBCPacketFixture(func(p *channeltypes.Packet) {
				p.TimeoutTimestamp = myTimestamp
				p.TimeoutHeight = clienttypes.Height{}
			}),
			exp: wvm.IBCPacket{
				Data:     []byte("myData"),
				Src:      wvm.IBCEndpoint{PortID: "srcPort", ChannelID: "channel-1"},
				Dest:     wvm.IBCEndpoint{PortID: "destPort", ChannelID: "channel-2"},
				Sequence: 1,
				Timeout:  wvm.IBCTimeout{Timestamp: myTimestamp},
			},
		}, "with time and height timeout": {
			src: IBCPacketFixture(func(p *channeltypes.Packet) {
				p.TimeoutTimestamp = myTimestamp
			}),
			exp: wvm.IBCPacket{
				Data:     []byte("myData"),
				Src:      wvm.IBCEndpoint{PortID: "srcPort", ChannelID: "channel-1"},
				Dest:     wvm.IBCEndpoint{PortID: "destPort", ChannelID: "channel-2"},
				Sequence: 1,
				Timeout: wvm.IBCTimeout{
					Block:     &wvm.IBCTimeoutBlock{Height: 1, Revision: 2},
					Timestamp: myTimestamp,
				},
			},
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			got := newIBCPacket(spec.src)
			assert.Equal(t, spec.exp, got)
		})
	}
}

func IBCPacketFixture(mutators ...func(p *channeltypes.Packet)) channeltypes.Packet {
	r := channeltypes.Packet{
		Sequence:           1,
		SourcePort:         "srcPort",
		SourceChannel:      "channel-1",
		DestinationPort:    "destPort",
		DestinationChannel: "channel-2",
		Data:               []byte("myData"),
		TimeoutHeight: clienttypes.Height{
			RevisionHeight: 1,
			RevisionNumber: 2,
		},
		TimeoutTimestamp: 0,
	}
	for _, m := range mutators {
		m(&r)
	}
	return r
}

var _ types.IBCContractKeeper = &IBCContractKeeperMock{}

type IBCContractKeeperMock struct {
	types.IBCContractKeeper
	OnRecvPacketFn    func(ctx sdk.Context, contractAddr sdk.AccAddress, msg wvm.IBCPacketReceiveMsg) (ibcexported.Acknowledgement, error)
	OnOpenChannelFn   func(ctx sdk.Context, contractAddr sdk.AccAddress, msg wvm.IBCChannelOpenMsg) (string, error)
	ClaimCapabilityFn func(ctx sdk.Context, cap *capabilitytypes.Capability, name string) error
}

func (m IBCContractKeeperMock) OnOpenChannel(ctx sdk.Context, contractAddr sdk.AccAddress, msg wvm.IBCChannelOpenMsg) (string, error) {
	if m.OnOpenChannelFn == nil {
		panic("not expected to be called")
	}
	return m.OnOpenChannelFn(ctx, contractAddr, msg)
}

func (m IBCContractKeeperMock) ClaimCapability(ctx sdk.Context, cap *capabilitytypes.Capability, name string) error {
	if m.ClaimCapabilityFn == nil {
		panic("not expected to be called")
	}
	return m.ClaimCapabilityFn(ctx, cap, name)
}

func (m IBCContractKeeperMock) OnRecvPacket(ctx sdk.Context, contractAddr sdk.AccAddress, msg wvm.IBCPacketReceiveMsg) (ibcexported.Acknowledgement, error) {
	if m.OnRecvPacketFn == nil {
		panic("not expected to be called")
	}
	return m.OnRecvPacketFn(ctx, contractAddr, msg)
}
