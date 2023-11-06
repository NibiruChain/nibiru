package types

import (
	fmt "fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
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
			sdk.MustNewDecFromStr("1111495344589.041095890410000000"),
		},
		{
			"pass - period 1",
			1,
			sdk.MustNewDecFromStr("556289865136.986301369863000000"),
		},
		{
			"pass - period 2",
			2,
			sdk.MustNewDecFromStr("278687125410.958904109589000000"),
		},
		{
			"pass - period 3",
			3,
			sdk.MustNewDecFromStr("139885755547.945205479452000000"),
		},
		{
			"pass - period 4",
			4,
			sdk.MustNewDecFromStr("70485070616.438356164383000000"),
		},
		{
			"pass - period 5",
			5,
			sdk.MustNewDecFromStr("35784728150.684931506849000000"),
		},
		{
			"pass - period 6",
			6,
			sdk.MustNewDecFromStr("18434556917.808219178082000000"),
		},
		{
			"pass - period 7",
			7,
			sdk.MustNewDecFromStr("9759471301.369863013698000000"),
		},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Case %s", tc.name), func(t *testing.T) {
			epochMintProvisions := CalculateEpochMintProvision(
				DefaultParams(),
				tc.period,
			)

			assert.EqualValues(t, tc.expEpochProvision, epochMintProvisions)
		})
	}
}

func TestCalculateEpochMintProvision_ZeroEpochs(t *testing.T) {
	params := DefaultParams()
	params.EpochsPerPeriod = 0

	epochMintProvisions := CalculateEpochMintProvision(params, 1)
	assert.EqualValues(t, epochMintProvisions, sdk.ZeroDec())
}
