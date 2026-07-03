package evmstate

import (
	"encoding/json"
	"testing"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/x/nutil/set"
)

// TestAccessListMarshalJSON tests the JSON marshaling of access lists
func TestAccessListMarshalJSON(t *testing.T) {
	tests := []struct {
		name       string
		accessList accessList
		wantJSON   string
	}{
		{
			name:       "empty access list",
			accessList: make(accessList),
			wantJSON:   "{}",
		},
		{
			name: "single address with no slots",
			accessList: accessList{
				gethcommon.HexToAddress("0x1234567890123456789012345678901234567890"): set.New[gethcommon.Hash](),
			},
			wantJSON: `{"0x1234567890123456789012345678901234567890":null}`,
		},
		{
			name: "single address with multiple slots",
			accessList: accessList{
				gethcommon.HexToAddress("0x1234567890123456789012345678901234567890"): set.New(
					gethcommon.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001"),
					gethcommon.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000002"),
				),
			},
			wantJSON: `{"0x1234567890123456789012345678901234567890":["0x0000000000000000000000000000000000000000000000000000000000000001","0x0000000000000000000000000000000000000000000000000000000000000002"]}`,
		},
	}

	gotLists := make([]accessList, len(tests))
	for idx, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			jsonData, err := tc.accessList.MarshalJSON()
			require.NoError(t, err)

			var got accessList
			err = json.Unmarshal(jsonData, &got)
			require.NoError(t, err, "expect unmarshaling to work for the marshaled data")
			gotLists[idx] = got

			var want accessList
			err = json.Unmarshal([]byte(tc.wantJSON), &want)
			require.NoError(t, err, "expect unmarshaling to work tc.wantJSON")

			isEqual, ineqReason := want.Equals(got)
			assert.Empty(t, ineqReason)
			require.Truef(t, isEqual,
				"want: %s,\n got: %s", tc.wantJSON, jsonData)
		})
	}

	t.Run("accestList.Equals works", func(t *testing.T) {
		isEqual, ineqReason := gotLists[1].Equals(gotLists[0])
		require.False(t, isEqual)
		require.Contains(t, ineqReason, "mismatch in number of keys")

		isEqual, ineqReason = gotLists[1].Equals(gotLists[2])
		require.False(t, isEqual)
		require.Contains(t, ineqReason, "slots mismatch")
	})
}

// TestAccessListUnmarshalJSON tests the JSON unmarshaling of access lists
func TestAccessListUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
		wantErr  bool
	}{
		{
			name:     "empty JSON object",
			jsonData: "{}",
			wantErr:  false,
		},
		{
			name:     "single address with empty slots",
			jsonData: `{"0x1234567890123456789012345678901234567890":[]}`,
			wantErr:  false,
		},
		{
			name:     "single address with slots",
			jsonData: `{"0x1234567890123456789012345678901234567890":["0x0000000000000000000000000000000000000000000000000000000000000001","0x0000000000000000000000000000000000000000000000000000000000000002"]}`,
			wantErr:  false,
		},
		{
			name:     "invalid JSON",
			jsonData: `{"invalid": json}`,
			wantErr:  true,
		},
		{
			name:     "invalid address format",
			jsonData: `{"invalid_address": []}`,
			wantErr:  true,
		},
		{
			name:     "invalid hash format",
			jsonData: `{"0x1234567890123456789012345678901234567890": ["invalid_hash"]}`,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var al accessList
			err := json.Unmarshal([]byte(tt.jsonData), &al)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestAccessListRoundTrip tests marshaling and unmarshaling round trip
func TestAccessListRoundTrip(t *testing.T) {
	original := accessList{
		gethcommon.HexToAddress("0x1234567890123456789012345678901234567890"): set.New(
			gethcommon.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001"),
			gethcommon.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000002"),
		),
		gethcommon.HexToAddress("0xabcdefabcdefabcdefabcdefabcdefabcdefabcd"): set.New(
			gethcommon.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000003"),
		),
	}

	// Marshal to JSON
	jsonData, err := original.MarshalJSON()
	require.NoError(t, err)

	// Unmarshal back
	var unmarshaled accessList
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err)

	// Should be equal
	require.Equal(t, original, unmarshaled)
}

// TestAccessListAddAddress tests the AddAddress method
func TestAccessListAddAddress(t *testing.T) {
	al := make(accessList)
	addr := gethcommon.HexToAddress("0x1234567890123456789012345678901234567890")

	// First addition should return true (address added)
	added := al.AddAddress(addr)
	require.True(t, added)
	require.True(t, al[addr] != nil)

	// Second addition should return false (address already present)
	added = al.AddAddress(addr)
	require.False(t, added)
}

// TestAccessListAddSlot tests the AddSlot method
func TestAccessListAddSlot(t *testing.T) {
	al := make(accessList)
	addr := gethcommon.HexToAddress("0x1234567890123456789012345678901234567890")
	slot := gethcommon.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001")

	// Add slot to non-existent address
	addrChange, slotChange := al.AddSlot(addr, slot)
	require.True(t, addrChange) // address was added
	require.True(t, slotChange) // slot was added
	require.True(t, al[addr].Has(slot))

	// Add same slot again
	addrChange, slotChange = al.AddSlot(addr, slot)
	require.False(t, addrChange) // address already present
	require.False(t, slotChange) // slot already present

	// Add new slot to existing address
	newSlot := gethcommon.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000002")
	addrChange, slotChange = al.AddSlot(addr, newSlot)
	require.False(t, addrChange) // address already present
	require.True(t, slotChange)  // new slot added
	require.True(t, al[addr].Has(newSlot))
}
