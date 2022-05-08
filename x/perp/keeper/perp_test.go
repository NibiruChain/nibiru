package keeper_test

import (
	"fmt"
	"testing"

	"github.com/NibiruChain/nibiru/x/perp/types"
	"github.com/NibiruChain/nibiru/x/testutil"
	"github.com/NibiruChain/nibiru/x/testutil/mock"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
	"github.com/golang/mock/gomock"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/stretchr/testify/require"
)

func TestGetAndSetPosition(t *testing.T) {
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

				dummyPosition := &types.Position{
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

func TestClearPosition(t *testing.T) {
	testCases := []struct {
		name string
		test func()
	}{
		{
			name: "set - creating position with set works and shows up in get",
			test: func() {
				mockCtrl := gomock.NewController(t)
				vpoolMock := mock.NewMockIVirtualPool(mockCtrl)
				vpoolPair := "osmo:nusd"

				traders := []sdk.AccAddress{
					sample.AccAddress(), sample.AccAddress(),
				}
				nibiruApp, ctx := testutil.NewNibiruApp(true)

				t.Log("vpool contains no positions to start")
				vpoolMock.EXPECT().Pair().Return(vpoolPair).Times(2)
				for _, trader := range traders {
					_, err := nibiruApp.PerpKeeper.GetPosition(
						ctx, vpoolMock, trader.String())
					require.Error(t, err)
					require.ErrorContains(t, err, fmt.Errorf("not found").Error())
				}

				var dummyPositions []*types.Position
				for _, trader := range traders {
					dummyPosition := &types.Position{
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
					t.Logf("position created successfully on vpool, %v, for trader %v",
						vpoolPair, trader.String())
					dummyPositions = append(dummyPositions, dummyPosition)
				}

				t.Log("attempt to clear all positions")
				vpoolMock.EXPECT().Pair().Return(vpoolPair).Times(3)

				require.NoError(t,
					nibiruApp.PerpKeeper.ClearPosition(
						ctx, vpoolMock, traders[0].String()),
				)

				outPosition, err := nibiruApp.PerpKeeper.GetPosition(
					ctx, vpoolMock, traders[0].String())
				require.NoError(t, err)
				require.EqualValues(t,
					types.ZeroPosition(ctx, vpoolPair, traders[0].String()),
					outPosition,
				)

				outPosition, err = nibiruApp.PerpKeeper.GetPosition(
					ctx, vpoolMock, traders[1].String())
				require.NoError(t, err)
				require.EqualValues(t, dummyPositions[1], outPosition)
				t.Log("trader 1 has a position and trader 0 does not.")

				t.Log("clearing position of trader 1...")
				vpoolMock.EXPECT().Pair().Return(vpoolPair).Times(2)
				require.NoError(t,
					nibiruApp.PerpKeeper.ClearPosition(
						ctx, vpoolMock, traders[1].String()),
				)
				outPosition, err = nibiruApp.PerpKeeper.GetPosition(
					ctx, vpoolMock, traders[1].String())
				require.NoError(t, err)
				require.EqualValues(t,
					types.ZeroPosition(ctx, vpoolPair, traders[1].String()),
					outPosition,
				)
				t.Log("Success, all trader positions have been cleared.")
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
