package keeper_test

import (
	"bytes"
	"github.com/NibiruChain/nibiru/x/testutil/sample"
	"github.com/NibiruChain/nibiru/x/testutil/testapp"
	"github.com/NibiruChain/nibiru/x/vpool/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	"sort"
	"testing"
)

func TestWhitelist(t *testing.T) {
	app := testapp.NewNibiruApp(false)
	ctx, _ := app.NewContext(false, tmproto.Header{}).CacheContext()
	whitelist := app.VpoolKeeper.Whitelist(ctx)

	t.Run("Add - success", func(t *testing.T) {
		addr := sample.AccAddress()
		require.NoError(t, whitelist.Add(addr))
		require.True(t, whitelist.IsWhitelisted(addr))
	})

	t.Run("Add - fail already exists", func(t *testing.T) {
		addr := sample.AccAddress()
		require.NoError(t, whitelist.Add(addr))
		require.ErrorIs(t, whitelist.Add(addr), types.ErrAlreadyInWhitelist)
	})

	t.Run("Remove - success", func(t *testing.T) {
		addr := sample.AccAddress()
		require.NoError(t, whitelist.Add(addr))
		require.NoError(t, whitelist.Remove(addr))
		require.False(t, whitelist.IsWhitelisted(addr))
	})

	t.Run("Remove - fail not exists", func(t *testing.T) {
		addr := sample.AccAddress()
		require.ErrorIs(t, whitelist.Remove(addr), types.ErrNotInWhitelist)
	})

	t.Run("Iterate", func(t *testing.T) {
		// we need to instantiate again the ctx
		// and hence the whitelist because of tests running before this
		ctx, _ := app.NewContext(false, tmproto.Header{}).CacheContext()
		whitelist := app.VpoolKeeper.Whitelist(ctx)
		var expected []sdk.AccAddress
		for i := 0; i < 10; i++ {
			addr := sample.AccAddress()
			expected = append(expected, addr)

			require.NoError(t, whitelist.Add(addr))
		}
		// sorting is needed because KVStore will store bytes in an ordered way.
		sort.Slice(expected, func(i, j int) bool {
			return bytes.Compare(expected[i], expected[j]) < 0
		})

		var got []sdk.AccAddress
		whitelist.Iterate(func(addr sdk.AccAddress) (stop bool) {
			got = append(got, addr)
			return false
		})

		require.Equal(t, expected, got)

	})

}
