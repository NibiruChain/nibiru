package types

import (
	fmt "fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestCalculateEpochMintProvision(t *testing.T) {
	testCases := []struct {
		name              string
		period            uint64
		expEpochProvision sdk.Dec
	}{
		{
			"pass - initial period",
			0,
			sdk.MustNewDecFromStr("1110672624143.835616438356000000"),
		},
		{
			"pass - period 1",
			1,
			sdk.MustNewDecFromStr("555878103595.890410958904000000"),
		},
		{
			"pass - period 2",
			2,
			sdk.MustNewDecFromStr("278480843321.917808219178000000"),
		},
		{
			"pass - period 3",
			3,
			sdk.MustNewDecFromStr("139782213184.931506849315000000"),
		},
		{
			"pass - period 4",
			4,
			sdk.MustNewDecFromStr("70432898116.438356164383000000"),
		},
		{
			"pass - period 5",
			5,
			sdk.MustNewDecFromStr("35758240582.191780821917000000"),
		},
		{
			"pass - period 6",
			6,
			sdk.MustNewDecFromStr("18420911815.068493150684000000"),
		},
		{
			"pass - period 7",
			7,
			sdk.MustNewDecFromStr("9752247431.506849315068000000"),
		},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Case %s", tc.name), func(t *testing.T) {
			epochMintProvisions := CalculateEpochMintProvision(
				DefaultParams(),
				tc.period,
			)

			require.Equal(t, tc.expEpochProvision, epochMintProvisions)
		})
	}
}
