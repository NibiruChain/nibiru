package pex

import (
	"net"
	"testing"

	"github.com/cosmos/gogoproto/proto"

	"github.com/NibiruChain/nibiru/v2/bft/config"
	"github.com/NibiruChain/nibiru/v2/bft/crypto/ed25519"
	"github.com/NibiruChain/nibiru/v2/bft/libs/log"
	"github.com/NibiruChain/nibiru/v2/bft/libs/service"
	"github.com/NibiruChain/nibiru/v2/bft/p2p"
	"github.com/NibiruChain/nibiru/v2/bft/p2p/pex"
	tmp2p "github.com/NibiruChain/nibiru/v2/bft/proto/tendermint/p2p"
	"github.com/NibiruChain/nibiru/v2/bft/version"
)

// FuzzP2PPex decodes bytes as a PEX channel message, the same way
// p2p.Peer does for bytes off the wire, and hands the result to
// pex.Reactor.ReceiveEnvelope. Seeds live in testdata/fuzz/FuzzP2PPex.
func FuzzP2PPex(f *testing.F) {
	// The book is never started, so it never reads or writes this file.
	addrB := pex.NewAddrBook("./testdata/addrbook1", false)
	addrB.SetLogger(log.NewNopLogger())
	pexR := pex.NewReactor(addrB, &pex.ReactorConfig{SeedMode: false})
	pexR.SetLogger(log.NewNopLogger())

	cfg := config.DefaultP2PConfig()
	cfg.PexReactor = true
	sw := p2p.MakeSwitch(cfg, 0, "127.0.0.1", "123.123.123",
		func(i int, sw *p2p.Switch) *p2p.Switch { return sw })
	pexR.SetSwitch(sw)

	peer := newFuzzPeer()
	pexR.AddPeer(peer)

	f.Fuzz(func(t *testing.T, data []byte) {
		// p2p.Peer panics on these errors, and the connection recovers and
		// drops the peer, so they never reach the reactor.
		var wrapped tmp2p.Message
		if err := proto.Unmarshal(data, &wrapped); err != nil {
			return
		}
		msg, err := wrapped.Unwrap()
		if err != nil {
			return
		}

		pexR.ReceiveEnvelope(p2p.Envelope{
			ChannelID: pex.PexChannel,
			Src:       peer,
			Message:   msg,
		})
	})
}

type fuzzPeer struct {
	*service.BaseService
	m map[string]interface{}
}

var _ p2p.Peer = (*fuzzPeer)(nil)

func newFuzzPeer() *fuzzPeer {
	fp := &fuzzPeer{m: make(map[string]interface{})}
	fp.BaseService = service.NewBaseService(nil, "fuzzPeer", fp)
	return fp
}

var (
	privKey         = ed25519.GenPrivKey()
	nodeID          = p2p.PubKeyToID(privKey.PubKey())
	defaultNodeInfo = p2p.DefaultNodeInfo{
		ProtocolVersion: p2p.NewProtocolVersion(
			version.P2PProtocol,
			version.BlockProtocol,
			0,
		),
		DefaultNodeID: nodeID,
		ListenAddr:    "0.0.0.0:98992",
		Moniker:       "foo1",
	}
)

func (fp *fuzzPeer) FlushStop()       {}
func (fp *fuzzPeer) ID() p2p.ID       { return nodeID }
func (fp *fuzzPeer) RemoteIP() net.IP { return net.IPv4(0, 0, 0, 0) }
func (fp *fuzzPeer) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: fp.RemoteIP(), Port: 98991, Zone: ""}
}
func (fp *fuzzPeer) IsOutbound() bool       { return false }
func (fp *fuzzPeer) IsPersistent() bool     { return false }
func (fp *fuzzPeer) CloseConn() error       { return nil }
func (fp *fuzzPeer) NodeInfo() p2p.NodeInfo { return defaultNodeInfo }
func (fp *fuzzPeer) Status() p2p.ConnectionStatus {
	return p2p.ConnectionStatus{}
}
func (fp *fuzzPeer) SocketAddr() *p2p.NetAddress {
	return p2p.NewNetAddress(fp.ID(), fp.RemoteAddr())
}
func (fp *fuzzPeer) SendEnvelope(e p2p.Envelope) bool    { return true }
func (fp *fuzzPeer) TrySendEnvelope(e p2p.Envelope) bool { return true }
func (fp *fuzzPeer) Send(_ byte, _ []byte) bool          { return true }
func (fp *fuzzPeer) TrySend(_ byte, _ []byte) bool       { return true }
func (fp *fuzzPeer) Set(key string, value interface{})   { fp.m[key] = value }
func (fp *fuzzPeer) Get(key string) interface{}          { return fp.m[key] }
func (fp *fuzzPeer) GetRemovalFailed() bool              { return false }
func (fp *fuzzPeer) SetRemovalFailed()                   {}
