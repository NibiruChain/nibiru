package keeper_test

import (
	"fmt"
	"testing"

	"github.com/NibiruChain/nibiru/x/testutil"
	"github.com/NibiruChain/nibiru/x/testutil/mock"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
	"github.com/golang/mock/gomock"

	"github.com/stretchr/testify/require"
)

func TestGetAndSetPosition(t *testing.T) {
	testCases := []struct {
		name string
		test func()
	}{
		{
			"no positions raises vpool not found error",
			func() {
				mockCtrl := gomock.NewController(t)
				defer mockCtrl.Finish()
				vpoolMock := mock.NewMockIVirtualPool(mockCtrl)

				trader := sample.AccAddress()
				nibiruApp, ctx := testutil.NewNibiruApp(true)

				vpoolMock.EXPECT().Pair().Return("osmo:nusd").Times(1)
				_, err := nibiruApp.PerpKeeper.GetPosition(
					ctx, vpoolMock, trader.String())
				require.Error(t, err)
				require.ErrorContains(t, err, fmt.Errorf("not found").Error())
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

func TestClearPosition(t *testing.T) {

}
