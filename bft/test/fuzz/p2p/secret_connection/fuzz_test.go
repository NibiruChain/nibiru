package secretconnection

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/NibiruChain/nibiru/v2/bft/crypto/ed25519"
	"github.com/NibiruChain/nibiru/v2/bft/libs/async"
	sc "github.com/NibiruChain/nibiru/v2/bft/p2p/conn"
)

// FuzzP2PSecretConnection writes arbitrary bytes into one end of a
// SecretConnection pair and checks that the other end reads the same bytes.
// Seeds live in testdata/fuzz/FuzzP2PSecretConnection.
func FuzzP2PSecretConnection(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		checkRoundTrip(t, data)
	})
}

// checkRoundTrip writes data through a fresh SecretConnection pair and fails
// t unless the reader gets the same bytes back.
func checkRoundTrip(t testing.TB, data []byte) {
	if len(data) == 0 {
		return
	}

	fooConn, barConn, err := makeSecretConnPair()
	if err != nil {
		t.Fatalf("handshake: %v", err)
	}
	defer fooConn.Close()
	defer barConn.Close()

	// io.Pipe has no buffer, so Write blocks until the other side reads.
	// Write from a goroutine and read here.
	writeErr := make(chan error, 1)
	go func() {
		// Write seals frames in place, so give it a copy.
		dataToWrite := make([]byte, len(data))
		copy(dataToWrite, data)
		n, err := fooConn.Write(dataToWrite)
		if err == nil && n < len(data) {
			err = fmt.Errorf("wrote %d of %d bytes", n, len(data))
		}
		writeErr <- err
	}()

	dataRead := make([]byte, len(data))
	totalRead := 0
	for totalRead < len(data) {
		m, err := barConn.Read(dataRead[totalRead:])
		if err != nil {
			t.Fatalf("read after %d of %d bytes: %v", totalRead, len(data), err)
		}
		totalRead += m
	}
	if err := <-writeErr; err != nil {
		t.Fatalf("write: %v", err)
	}

	if !bytes.Equal(data, dataRead) {
		t.Fatalf("bytes written %X != read %X", data, dataRead)
	}
}

type kvstoreConn struct {
	*io.PipeReader
	*io.PipeWriter
}

func (drw kvstoreConn) Close() error {
	err2 := drw.PipeWriter.CloseWithError(io.EOF)
	err1 := drw.PipeReader.Close()
	if err2 != nil {
		return err2
	}
	return err1
}

// makeKVStoreConnPair returns two connected pipes. Each end is akin to a
// net.Conn.
func makeKVStoreConnPair() (fooConn, barConn kvstoreConn) {
	barReader, fooWriter := io.Pipe()
	fooReader, barWriter := io.Pipe()
	return kvstoreConn{fooReader, fooWriter}, kvstoreConn{barReader, barWriter}
}

// makeSecretConnPair runs the SecretConnection handshake on both ends of a
// pipe pair and checks that each side sees the other's public key.
func makeSecretConnPair() (
	fooSecConn, barSecConn *sc.SecretConnection, err error,
) {
	var (
		fooConn, barConn = makeKVStoreConnPair()
		fooPrvKey        = ed25519.GenPrivKey()
		fooPubKey        = fooPrvKey.PubKey()
		barPrvKey        = ed25519.GenPrivKey()
		barPubKey        = barPrvKey.PubKey()
	)

	trs, ok := async.Parallel(
		func(_ int) (val interface{}, abort bool, err error) {
			fooSecConn, err = sc.MakeSecretConnection(fooConn, fooPrvKey)
			if err != nil {
				return nil, true, fmt.Errorf("foo: %w", err)
			}
			if !fooSecConn.RemotePubKey().Equals(barPubKey) {
				return nil, true, fmt.Errorf("foo: remote pubkey %v, want %v",
					fooSecConn.RemotePubKey(), barPubKey)
			}
			return nil, false, nil
		},
		func(_ int) (val interface{}, abort bool, err error) {
			barSecConn, err = sc.MakeSecretConnection(barConn, barPrvKey)
			if err != nil {
				return nil, true, fmt.Errorf("bar: %w", err)
			}
			if !barSecConn.RemotePubKey().Equals(fooPubKey) {
				return nil, true, fmt.Errorf("bar: remote pubkey %v, want %v",
					barSecConn.RemotePubKey(), fooPubKey)
			}
			return nil, false, nil
		},
	)

	if err := trs.FirstError(); err != nil {
		return nil, nil, err
	}
	if !ok {
		return nil, nil, fmt.Errorf("handshake task aborted")
	}
	return fooSecConn, barSecConn, nil
}
