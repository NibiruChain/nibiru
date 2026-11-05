package mock

import (
	"github.com/NibiruChain/nibiru/v2/bft/libs/log"
	"github.com/NibiruChain/nibiru/v2/bft/p2p"
	"github.com/NibiruChain/nibiru/v2/bft/p2p/conn"
)

type Reactor struct {
	p2p.BaseReactor

	Channels []*conn.ChannelDescriptor
}

func NewReactor() *Reactor {
	r := &Reactor{}
	r.BaseReactor = *p2p.NewBaseReactor("Mock-PEX", r)
	r.SetLogger(log.TestingLogger())
	return r
}

func (r *Reactor) GetChannels() []*conn.ChannelDescriptor       { return r.Channels }
func (r *Reactor) AddPeer(peer p2p.Peer)                        {}
func (r *Reactor) RemovePeer(peer p2p.Peer, reason interface{}) {}
func (r *Reactor) ReceiveEnvelope(e p2p.Envelope)               {}
