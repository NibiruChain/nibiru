package keeper

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common/asset"
	"github.com/NibiruChain/nibiru/x/common/denoms"
	perpammtypes "github.com/NibiruChain/nibiru/x/perp/amm/types"
	"github.com/NibiruChain/nibiru/x/perp/types"
)

func TestEditPoolPegMultiplier(t *testing.T) {
	tests := []struct {
		name string

		initialPeg sdk.Dec
		newPeg     sdk.Dec
		bias       sdk.Dec

		expectedErr   error
		expectedCost  sdk.Int
		expectedEvent types.PegMultiplierUpdate
	}{
		{
			name:         "happy path - zero cost because no bias",
			initialPeg:   sdk.OneDec(),
			newPeg:       sdk.NewDec(2),
			bias:         sdk.ZeroDec(),
			expectedCost: sdk.ZeroInt(),
		},
		{
			name:         "happy path - zero cost because no repeg",
			initialPeg:   sdk.OneDec(),
			newPeg:       sdk.OneDec(),
			bias:         sdk.NewDec(100),
			expectedCost: sdk.ZeroInt(),
		},
		{
			name:         "happy path - simple math with positive bias",
			initialPeg:   sdk.OneDec(),
			newPeg:       sdk.NewDec(2),
			bias:         sdk.NewDec(25), // Bias in quote should be 20
			expectedCost: sdk.NewInt(20), // 20 * (2 - 1)
		},
		{
			name:         "happy path - simple math with negative bias",
			initialPeg:   sdk.OneDec(),
			newPeg:       sdk.NewDec(2),
			bias:         sdk.NewDec(-20), // Bias in quote should be 20
			expectedCost: sdk.NewInt(-25), // 20 * (2 - 1)
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			perpKeeper, mocks, ctx := getKeeper(t)
			pair := asset.Registry.Pair(denoms.BTC, denoms.NUSD)

			ammPool := perpammtypes.NewMarket(
				perpammtypes.ArgsNewMarket{
					Pair:          pair,
					BaseReserves:  sdk.NewDec(100),
					QuoteReserves: sdk.NewDec(100),
					Config:        perpammtypes.DefaultMarketConfig(),
					Bias:          tc.bias,
					PegMultiplier: tc.initialPeg,
				},
			)

			mocks.mockPerpAmmKeeper.EXPECT().GetPool(ctx, pair).Return(ammPool, nil)
			if tc.expectedErr == nil {
				if tc.expectedCost.IsPositive() {
					mocks.mockBankKeeper.EXPECT().
						SendCoinsFromModuleToModule(
							ctx,
							types.PerpEFModuleAccount,
							types.VaultModuleAccount,
							sdk.NewCoins(
								sdk.NewCoin(pair.QuoteDenom(), tc.expectedCost),
							),
						).Return(nil)
				} else if tc.expectedCost.IsNegative() {
					mocks.mockBankKeeper.EXPECT().
						SendCoinsFromModuleToModule(
							ctx,
							types.VaultModuleAccount,
							types.PerpEFModuleAccount,
							sdk.NewCoins(
								sdk.NewCoin(pair.QuoteDenom(), tc.expectedCost.Neg()),
							),
						).Return(nil)
				}
			}

			err := perpKeeper.EditPoolPegMultiplier(ctx, sdk.AccAddress{}, pair, tc.newPeg)

			if tc.expectedErr != nil {
				require.EqualError(t, err, tc.expectedErr.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}
