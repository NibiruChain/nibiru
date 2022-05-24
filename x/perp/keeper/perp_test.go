package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/NibiruChain/nibiru/x/common"

	"github.com/NibiruChain/nibiru/x/perp/types"
	"github.com/NibiruChain/nibiru/x/testutil"
	"github.com/NibiruChain/nibiru/x/testutil/sample"

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
				trader := sample.AccAddress()
				nibiruApp, ctx := testutil.NewNibiruApp(true)

				_, err := nibiruApp.PerpKeeper.GetPosition(
					ctx, "osmo:nusd", trader)
				require.Error(t, err)
				require.ErrorContains(t, err, types.ErrPositionNotFound.Error())
			},
		},
		{
			name: "set - creating position with set works and shows up in get",
			test: func() {
				vpoolPair, err := common.NewTokenPairFromStr("osmo:nusd")
				require.NoError(t, err)

				traderAddr := sample.AccAddress()
				nibiruApp, ctx := testutil.NewNibiruApp(true)

				_, err = nibiruApp.PerpKeeper.GetPosition(
					ctx, vpoolPair, traderAddr)
				require.Error(t, err)
				require.ErrorContains(t, err, types.ErrPositionNotFound.Error())

				dummyPosition := &types.Position{
					TraderAddress: traderAddr,
					Pair:          vpoolPair.String(),
					Size_:         sdk.OneDec(),
					Margin:        sdk.OneDec(),
				}
				nibiruApp.PerpKeeper.SetPosition(
					ctx, vpoolPair, traderAddr, dummyPosition)
				outPosition, err := nibiruApp.PerpKeeper.GetPosition(
					ctx, vpoolPair, traderAddr)
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
				vpoolPair, err := common.NewTokenPairFromStr("osmo:nusd")
				require.NoError(t, err)

				traders := []sdk.AccAddress{
					sample.AccAddress(), sample.AccAddress(),
				}
				nibiruApp, ctx := testutil.NewNibiruApp(true)

				t.Log("vpool contains no positions to start")
				for _, trader := range traders {
					_, err := nibiruApp.PerpKeeper.GetPosition(
						ctx, vpoolPair, trader)
					require.Error(t, err)
					require.ErrorContains(t, err, types.ErrPositionNotFound.Error())
				}

				var dummyPositions []*types.Position
				for _, traderAddr := range traders {
					dummyPosition := &types.Position{
						TraderAddress: traderAddr,
						Pair:          vpoolPair.String(),
						Size_:         sdk.OneDec(),
						Margin:        sdk.OneDec(),
					}
					nibiruApp.PerpKeeper.SetPosition(
						ctx, vpoolPair, traderAddr, dummyPosition)
					outPosition, err := nibiruApp.PerpKeeper.GetPosition(
						ctx, vpoolPair, traderAddr)
					require.NoError(t, err)
					require.EqualValues(t, dummyPosition, outPosition)
					t.Logf("position created successfully on vpool, %v, for trader %v",
						vpoolPair, traderAddr.String())
					dummyPositions = append(dummyPositions, dummyPosition)
				}

				t.Log("attempt to clear all positions")

				require.NoError(t,
					nibiruApp.PerpKeeper.ClearPosition(
						ctx, vpoolPair, traders[0]),
				)

				outPosition, err := nibiruApp.PerpKeeper.GetPosition(
					ctx, vpoolPair, traders[0])
				require.NoError(t, err)
				require.EqualValues(t,
					types.ZeroPosition(ctx, vpoolPair, traders[0]),
					outPosition,
				)

				outPosition, err = nibiruApp.PerpKeeper.GetPosition(
					ctx, vpoolPair, traders[1])
				require.NoError(t, err)
				require.EqualValues(t, dummyPositions[1], outPosition)
				t.Log("trader 1 has a position and trader 0 does not.")

				t.Log("clearing position of trader 1...")
				require.NoError(t,
					nibiruApp.PerpKeeper.ClearPosition(
						ctx, vpoolPair, traders[1]),
				)
				outPosition, err = nibiruApp.PerpKeeper.GetPosition(
					ctx, vpoolPair, traders[1])
				require.NoError(t, err)
				require.EqualValues(t,
					types.ZeroPosition(ctx, vpoolPair, traders[1]),
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
