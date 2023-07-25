package types

import (
	"bytes"
	"testing"
)

func TestChangeReason(t *testing.T) {
	testCases := []struct {
		name   string
		reason ChangeReason
		data   []byte
	}{
		{"MarketOrder", ChangeReason_MarketOrder, []byte("market_order")},
		{"ClosePosition", ChangeReason_ClosePosition, []byte("close_position")},
		// add other cases here...
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test Marshal
			result, err := tc.reason.Marshal()
			if err != nil {
				t.Errorf("unexpected error on Marshal: %v", err)
			}

			if !bytes.Equal(result, tc.data) {
				t.Errorf("expected %s, got %s on Marshal", tc.data, result)
			}

			// Test Unmarshal
			var reason ChangeReason
			err = reason.Unmarshal(tc.data)
			if err != nil {
				t.Errorf("unexpected error on Unmarshal: %v", err)
			}

			if reason != tc.reason {
				t.Errorf("expected %s, got %s on Unmarshal", tc.reason, reason)
			}

			// Test MarshalJSON and UnmarshalJSON
			jsonData, err := tc.reason.MarshalJSON()
			if err != nil {
				t.Errorf("unexpected error on MarshalJSON: %v", err)
			}

			var jsonResult ChangeReason
			err = jsonResult.UnmarshalJSON(jsonData)
			if err != nil {
				t.Errorf("unexpected error on UnmarshalJSON: %v", err)
			}

			if jsonResult != tc.reason {
				t.Errorf("expected %s, got %s on UnmarshalJSON", tc.reason, jsonResult)
			}
		})
	}
}
