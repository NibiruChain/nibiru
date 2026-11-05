package types_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/upgrade/types"
)

func TestScheduleRequestSeparatesTimeTargetsFromExecutablePlans(t *testing.T) {
	timestamp := time.Date(2027, 1, 2, 3, 4, 5, 123456789, time.UTC)
	for _, tc := range []struct {
		name              string
		plan              types.Plan
		requestOK, planOK bool
	}{
		{"height", types.Plan{Name: "next", Height: 100}, true, true},
		{"time", types.Plan{Name: "next", Time: timestamp}, true, false},
		{"ambiguous", types.Plan{Name: "next", Height: 100, Time: timestamp}, false, false},
		{"missing", types.Plan{Name: "next"}, false, false},
		{"negative", types.Plan{Name: "next", Height: -1}, false, false},
		{"missing name", types.Plan{Time: timestamp}, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.requestOK, tc.plan.ValidateScheduleRequest() == nil)
			require.Equal(t, tc.planOK, tc.plan.ValidateBasic() == nil)
		})
	}
}
