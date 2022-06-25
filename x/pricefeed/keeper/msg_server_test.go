package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/x/common"
	"github.com/NibiruChain/nibiru/x/pricefeed/keeper"
	"github.com/NibiruChain/nibiru/x/pricefeed/types"
	testutilkeeper "github.com/NibiruChain/nibiru/x/testutil/keeper"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
)

func TestPostPrice(t *testing.T) {
	k, ctx := testutilkeeper.PricefeedKeeper(t)
	msgSrv := keeper.NewMsgServerImpl(k)

	_, addrs := sample.PrivKeyAddressPairs(4)
	authorizedOracles := addrs[:2]
	unauthorizedAddrs := addrs[2:]

	pair := common.MustNewAssetPair("usd:tst")
	params := types.Params{
		Pairs: common.AssetPairs{pair},
	}
	k.SetParams(ctx, params)
	k.WhitelistOraclesForPairs(ctx, authorizedOracles, common.AssetPairs{pair})

	tests := []struct {
		giveMsg      string
		giveOracle   sdk.AccAddress
		giveToken0   string
		giveToken1   string
		giveExpiry   time.Time
		wantAccepted bool
		errorKind    error
	}{
		{"authorized", authorizedOracles[0], "tst", "usd",
			ctx.BlockTime().UTC().Add(time.Hour * 1), true, nil},
		{"expired", authorizedOracles[0], "tst", "usd",
			ctx.BlockTime().UTC().Add(-time.Hour * 1), false, types.ErrExpired},
		{"invalid", authorizedOracles[0], "invalid", "invalid",
			ctx.BlockTime().UTC().Add(time.Hour * 1), false, types.ErrInvalidOracle},
		{"unauthorized", unauthorizedAddrs[0], "tst", "usd",
			ctx.BlockTime().UTC().Add(time.Hour * 1), false, types.ErrInvalidOracle},
	}

	for _, tt := range tests {
		t.Run(tt.giveMsg, func(t *testing.T) {
			// Use MsgServer over keeper methods directly to test against valid oracles
			msg := types.NewMsgPostPrice(
				tt.giveOracle.String(), tt.giveToken0, tt.giveToken1,
				sdk.MustNewDecFromStr("0.5"), tt.giveExpiry)
			_, err := msgSrv.PostPrice(sdk.WrapSDKContext(ctx), msg)

			if tt.wantAccepted {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.ErrorIs(t, tt.errorKind, err)
			}
		})
	}
}
