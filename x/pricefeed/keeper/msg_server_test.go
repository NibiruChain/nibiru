package keeper_test

import (
	"testing"
	"time"

	"github.com/NibiruChain/nibiru/x/pricefeed/keeper"
	"github.com/NibiruChain/nibiru/x/pricefeed/types"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
	"github.com/NibiruChain/nibiru/x/testutil/testkeeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestPostPrice(t *testing.T) {
	k, ctx := testkeeper.PricefeedKeeper(t)
	msgSrv := keeper.NewMsgServerImpl(k)

	_, addrs := sample.PrivKeyAddressPairs(4)
	authorizedOracles := addrs[:2]
	unauthorizedAddrs := addrs[2:]

	mp := types.Params{
		Markets: []types.Market{
			{MarketID: "tstusd", BaseAsset: "tst", QuoteAsset: "usd", Oracles: authorizedOracles, Active: true},
		},
	}
	k.SetParams(ctx, mp)

	tests := []struct {
		giveMsg      string
		giveOracle   sdk.AccAddress
		giveMarketId string
		giveExpiry   time.Time
		wantAccepted bool
		errorKind    error
	}{
		{"authorized", authorizedOracles[0], "tstusd", ctx.BlockTime().UTC().Add(time.Hour * 1), true, nil},
		{"expired", authorizedOracles[0], "tstusd", ctx.BlockTime().UTC().Add(-time.Hour * 1), false, types.ErrExpired},
		{"invalid", authorizedOracles[0], "invalid", ctx.BlockTime().UTC().Add(time.Hour * 1), false, types.ErrInvalidMarket},
		{"unauthorized", unauthorizedAddrs[0], "tstusd", ctx.BlockTime().UTC().Add(time.Hour * 1), false, types.ErrInvalidOracle},
	}

	for _, tt := range tests {
		t.Run(tt.giveMsg, func(t *testing.T) {
			// Use MsgServer over keeper methods directly to tests against valid oracles
			msg := types.NewMsgPostPrice(tt.giveOracle.String(), tt.giveMarketId, sdk.MustNewDecFromStr("0.5"), tt.giveExpiry)
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
