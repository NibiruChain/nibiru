package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateEpochIdentifierInterface(t *testing.T) {
	testCases := []struct {
		name       string
		id         interface{}
		expectPass bool
	}{
		{
			"invalid - blank identifier",
			"",
			false,
		},
		{
			"invalid - blank identifier with spaces",
			"   ",
			false,
		},
		{
			"invalid - non-string",
			3,
			false,
		},
		{
			"pass",
			WeekEpochID,
			true,
		},
	}

	for _, tc := range testCases {
		err := ValidateEpochIdentifierInterface(tc.id)

		if tc.expectPass {
			require.NoError(t, err)
		} else {
			require.Error(t, err)
		}
	}
}
