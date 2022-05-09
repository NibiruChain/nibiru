package keeper_test

import (
	"fmt"
	"testing"

	perptypes "github.com/NibiruChain/nibiru/x/perp/types/v1"
	"github.com/NibiruChain/nibiru/x/testutil"
	"github.com/NibiruChain/nibiru/x/testutil/mock"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
	"github.com/golang/mock/gomock"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/stretchr/testify/require"
)

func TestGetLatestCumulativePremiumFraction(t *testing.T) {
	testCases := []struct {
		name string
		test func()
	}{
		{
			name: "uninitialized vpool has no metadata",
			test: func() {
				nibiruApp, ctx := testutil.NewNibiruApp(true)
				vpool := "xxx:yyy"
				lcpf, err := nibiruApp.PerpKeeper.GetLatestCumulativePremiumFraction(
					ctx, vpool)
				require.Error(t, err)
				require.EqualValues(t, sdk.Int{}, lcpf)
			},
		},
		{
			name: "get - no positions set raises vpool not found error",
			test: func() {
				mockCtrl := gomock.NewController(t)
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
		{
			name: "set - creating position with set works and shows up in get",
			test: func() {
				mockCtrl := gomock.NewController(t)
				vpoolMock := mock.NewMockIVirtualPool(mockCtrl)
				vpoolPair := "osmo:nusd"

				trader := sample.AccAddress()
				nibiruApp, ctx := testutil.NewNibiruApp(true)

				vpoolMock.EXPECT().Pair().Return(vpoolPair).Times(1)
				_, err := nibiruApp.PerpKeeper.GetPosition(
					ctx, vpoolMock, trader.String())
				require.Error(t, err)
				require.ErrorContains(t, err, fmt.Errorf("not found").Error())

				dummyPosition := &perptypes.Position{
					Address: trader.String(),
					Pair:    vpoolPair,
					Size_:   sdk.OneInt(),
					Margin:  sdk.OneInt(),
				}
				vpoolMock.EXPECT().Pair().Return(vpoolPair).Times(2)
				nibiruApp.PerpKeeper.SetPosition(
					ctx, vpoolMock, trader.String(), dummyPosition)
				outPosition, err := nibiruApp.PerpKeeper.GetPosition(
					ctx, vpoolMock, trader.String())
				require.NoError(t, err)
				require.EqualValues(t, dummyPosition, outPosition)
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

func TestCalcRemainMarginWithFundingPayment(t *testing.T) {
	testCases := []struct {
		name string
		test func()
	}{
		{
			name: "get - no positions set raises vpool not found error",
			test: func() {
				mockCtrl := gomock.NewController(t)
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
		{
			name: "set - creating position with set works and shows up in get",
			test: func() {
				mockCtrl := gomock.NewController(t)
				vpoolMock := mock.NewMockIVirtualPool(mockCtrl)
				vpoolPair := "osmo:nusd"

				trader := sample.AccAddress()
				nibiruApp, ctx := testutil.NewNibiruApp(true)

				vpoolMock.EXPECT().Pair().Return(vpoolPair).Times(1)
				_, err := nibiruApp.PerpKeeper.GetPosition(
					ctx, vpoolMock, trader.String())
				require.Error(t, err)
				require.ErrorContains(t, err, fmt.Errorf("not found").Error())

				dummyPosition := &perptypes.Position{
					Address: trader.String(),
					Pair:    vpoolPair,
					Size_:   sdk.OneInt(),
					Margin:  sdk.OneInt(),
				}
				vpoolMock.EXPECT().Pair().Return(vpoolPair).Times(2)
				nibiruApp.PerpKeeper.SetPosition(
					ctx, vpoolMock, trader.String(), dummyPosition)
				outPosition, err := nibiruApp.PerpKeeper.GetPosition(
					ctx, vpoolMock, trader.String())
				require.NoError(t, err)
				require.EqualValues(t, dummyPosition, outPosition)
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
