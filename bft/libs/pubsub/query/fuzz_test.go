package query_test

import (
	"testing"

	"github.com/NibiruChain/nibiru/v2/bft/libs/pubsub/query"
)

// FuzzPubsubQuery parses arbitrary event queries, as sent to the subscribe
// and search RPC endpoints. A query that parses must print to a string that
// parses again and prints the same way. Seeds live in
// testdata/fuzz/FuzzPubsubQuery.
func FuzzPubsubQuery(f *testing.F) {
	f.Fuzz(func(t *testing.T, data string) {
		q0, err := query.New(data)
		if err != nil {
			return
		}

		sdata1 := q0.String()
		q1, err := query.New(sdata1)
		if err != nil {
			t.Fatalf("reparse %q (from %q): %v", sdata1, data, err)
		}

		sdata2 := q1.String()
		if sdata1 != sdata2 {
			t.Fatalf("query changed on reparse: %q -> %q", sdata1, sdata2)
		}
	})
}
