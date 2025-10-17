package nutil_test

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/x/nutil"
)

func TestParseNibiBalance_Table(t *testing.T) {
	W := nutil.WeiPerUnibi   // 10^12
	Wm1 := W.SubRaw(1)       // 10^12 - 1
	twoW := W.MulRaw(2)      // 2*10^12
	twoWm1 := twoW.SubRaw(1) // 2*10^12 - 1
	cases := []struct {
		name     string
		inputWei sdkmath.Int
		expU     sdkmath.Int
		expW     sdkmath.Int
	}{
		{"zero", sdkmath.ZeroInt(), sdkmath.ZeroInt(), sdkmath.ZeroInt()},
		{"one", sdkmath.OneInt(), sdkmath.ZeroInt(), sdkmath.OneInt()},
		{"W-1", Wm1, sdkmath.ZeroInt(), Wm1},
		{"W", W, sdkmath.OneInt(), sdkmath.ZeroInt()},
		{"W+1", W.AddRaw(1), sdkmath.OneInt(), sdkmath.OneInt()},
		{"2W-1", twoWm1, sdkmath.NewInt(1), sdkmath.NewInt(0).Add(W).SubRaw(1)},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			u, w := nutil.ParseNibiBalance(tc.inputWei)

			// Exact expected split (for explicit cases)
			require.True(t, u.Equal(tc.expU), "expU=%s got=%s", tc.expU, u)
			require.True(t, w.Equal(tc.expW), "expW=%s got=%s", tc.expW, w)

			// Invariant: 0 <= w < W
			require.False(t, w.IsNegative(), "amtWei must be non-negative")
			require.True(t, w.LT(W), "amtWei must be < WeiPerUnibi")

			// Invariant: u*W + w == input
			reconstructed := u.Mul(W).Add(w)
			require.True(t, reconstructed.Equal(tc.inputWei),
				"reconstruct failed: got=%s want=%s", reconstructed, tc.inputWei)
		})
	}
}

func TestParseNibiBalanceFromParts_Normalizes(t *testing.T) {
	W := nutil.WeiPerUnibi // 10^12

	type tc struct {
		name  string
		unibi sdkmath.Int
		wei   sdkmath.Int
		expU  sdkmath.Int
		expW  sdkmath.Int
	}
	cases := []tc{
		{"zero,zero", sdkmath.ZeroInt(), sdkmath.ZeroInt(), sdkmath.ZeroInt(), sdkmath.ZeroInt()},
		{"oneUnibi_noWei", sdkmath.OneInt(), sdkmath.ZeroInt(), sdkmath.OneInt(), sdkmath.ZeroInt()},
		{"oneUnibi_W-1", sdkmath.OneInt(), W.SubRaw(1), sdkmath.OneInt(), W.SubRaw(1)},
		{"oneUnibi_W", sdkmath.OneInt(), W, sdkmath.NewInt(2), sdkmath.ZeroInt()},                          // carry
		{"fiveUnibi_2W+3", sdkmath.NewInt(5), W.MulRaw(2).AddRaw(3), sdkmath.NewInt(7), sdkmath.NewInt(3)}, // 5 + 2 carry = 7, remainder 3
		{"zero_123W+456", sdkmath.ZeroInt(), W.MulRaw(123).AddRaw(456), sdkmath.NewInt(123), sdkmath.NewInt(456)},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			u, w := nutil.ParseNibiBalanceFromParts(c.unibi, c.wei)

			// Exact expected split
			require.True(t, u.Equal(c.expU), "expU=%s got=%s", c.expU, u)
			require.True(t, w.Equal(c.expW), "expW=%s got=%s", c.expW, w)

			// Invariant: 0 <= w < W
			require.False(t, w.IsNegative(), "amtWei must be non-negative")
			require.True(t, w.LT(W), "amtWei must be < WeiPerUnibi")

			// Invariant: u*W + w == unibi*W + wei  (i.e., inputs normalized)
			left := u.Mul(W).Add(w)
			right := c.unibi.Mul(W).Add(c.wei)
			require.True(t, left.Equal(right), "normalized total mismatch: got=%s want=%s", left, right)
		})
	}
}
