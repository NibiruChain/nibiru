package addr

import (
	"encoding/json"
	"math/rand"
	"testing"

	"github.com/NibiruChain/nibiru/v2/bft/libs/log"
	"github.com/NibiruChain/nibiru/v2/bft/p2p"
	"github.com/NibiruChain/nibiru/v2/bft/p2p/pex"
)

// FuzzP2PAddrBook decodes a JSON p2p.NetAddress, as a peer would send it,
// and adds it to an address book with AddrBook.AddAddress. After a
// successful add, PickAddress must return a non-nil address. Seeds live in
// testdata/fuzz/FuzzP2PAddrBook.
func FuzzP2PAddrBook(f *testing.F) {
	// The book is never started, so it never reads or writes this file.
	addrBook := pex.NewAddrBook("./testdata/addrbook.json", true)
	addrBook.SetLogger(log.NewNopLogger())

	f.Fuzz(func(t *testing.T, data []byte) {
		addr := new(p2p.NetAddress)
		if err := json.Unmarshal(data, addr); err != nil {
			return
		}

		if err := addrBook.AddAddress(addr, addr); err != nil {
			return
		}

		bias := rand.Intn(100) //nolint:gosec
		if p := addrBook.PickAddress(bias); p == nil {
			t.Fatalf("picked a nil address (bias: %d, addrBook size: %v)",
				bias, addrBook.Size())
		}
	})
}
