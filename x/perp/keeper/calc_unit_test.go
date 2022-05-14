package keeper

import (
	"testing"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/perp/types"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
)

func Test_calcFreeCollateral(t *testing.T) {

	testCases := []struct {
		name string
		test func()
	}{
		{
			name: "invalid token pair - fail",
			test: func() {
				k, _, ctx := getKeeper(t)
				fundingPayment := sdk.ZeroDec()
				the3pool := "dai:usdc:usdt"
				alice := sample.AccAddress()
				pos := types.ZeroPosition(ctx, common.TokenPair(the3pool), alice.String())
				_, err := k.calcFreeCollateral(ctx, *pos, fundingPayment)
				assert.Error(t, err)
				assert.ErrorContains(t, err, common.ErrInvalidTokenPair.Error())
			},
		},
		{
			name: "token pair not found - fail",
			test: func() {
				k, _, ctx := getKeeper(t)
				k, mocks, ctx := getKeeper(t)

				fundingPayment := sdk.ZeroDec()
				validPair := common.TokenPair("xxx:yyy")
				alice := sample.AccAddress()
				pos := types.ZeroPosition(ctx, validPair, alice.String())
				mocks.mockVpoolKeeper.EXPECT().ExistsPool(ctx, validPair).
					Return(false)
				_, err := k.calcFreeCollateral(ctx, *pos, fundingPayment)
				assert.Error(t, err)
				assert.ErrorContains(t, err, types.ErrPairNotFound.Error())
			},
		},
	}

	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			tc.test()
		})
	}
}
