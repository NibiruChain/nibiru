package nutil

import (
	"fmt"
	"math/big"

	sdkmath "cosmossdk.io/math"
	"github.com/holiman/uint256"
)

// WeiPerUnibi is a big.Int for 10^{12}. Each "unibi" (micronibi) is 10^{12}
// wei because 1 NIBI = 10^{18} wei.
var WeiPerUnibi = sdkmath.NewIntFromBigInt(
	new(big.Int).Exp(big.NewInt(10), big.NewInt(12), nil),
)

// WeiPerUnibiU256 is a uint256.Int for 10^{12}. Each "unibi" (micronibi) is
// 10^{12} wei because 1 NIBI = 10^{18} wei.
func WeiPerUnibiU256() *uint256.Int {
	return uint256.MustFromBig(WeiPerUnibi.BigInt())
}

// ParseNibiBalance splits a NIBI amount in wei into unibi (bank units) and wei
// remainder. Used by keeper to normalize the dual-balance model at the 10^12
// boundary.
//   - This is the inverse of GetWeiBalance aggregation: (unibi × 10^12) + wei.
//   - Example: ParseNibiBalance(2×10^12 + 3) → (2, 3)
//
// ### Returns:
//   - amtUnibi: bank balance in unibi (micro-NIBI; 10^{-6} NIBI)
//   - amtWei: remainder in wei, where 0 ≤ amtWei < 10^{12}
func ParseNibiBalance(wei sdkmath.Int) (amtUnibi, amtWei sdkmath.Int) {
	return wei.Quo(WeiPerUnibi), wei.Mod(WeiPerUnibi)
}

// ParseNibiBalanceFromParts normalizes (unibi, wei) into canonical form.
// Computes total wei as (unibi × 10^{12}) + wei, then splits into normalized parts.
//
// Returns:
//   - amtUnibi: normalized unibi (micro-NIBI; 10^-6 NIBI)
//   - amtWei: normalized remainder, where 0 ≤ amtWei < 10^{12}
//
// Used by keeper to carry/borrow across the 10^{12} boundary, keeping bank and
// wei-store synchronized.
//
// Example: ParseNibiBalanceFromParts(5, 2×10^{12} + 3) → (7, 3)
func ParseNibiBalanceFromParts(unibi, wei sdkmath.Int) (amtUnibi, amtWei sdkmath.Int) {
	unibiPartInWei := unibi.Mul(WeiPerUnibi)
	totalWei := unibiPartInWei.Add(wei)
	return ParseNibiBalance(totalWei)
}

// U256SafeFromBig converts a big.Int to uint256.Int, returning
// an error if x < 0 or x.BitLen() > 256. Nil or zero is treated as zero.
func U256SafeFromBig(x *big.Int) (*uint256.Int, error) {
	switch {
	case x == nil:
		return nil, nil
	case x.Sign() < 0:
		return nil, fmt.Errorf("BigToU256 Error: negtive number cannot be a uint256 { num: %s }", x)
	case x.BitLen() > 256:
		return nil, fmt.Errorf("BigToU256 Error: overflow (>256 bits) { num: %s }", x)
	case x.IsUint64():
		u := new(uint256.Int)
		u.SetUint64(x.Uint64())
		return u, nil
	}

	num, isOverflow := uint256.FromBig(x)
	if isOverflow {
		return nil, fmt.Errorf("BigToU256 Error: overflow (uint256.FromBig) { num: %s }", x)
	}
	return num, nil
}

// U256SafeFromSdkInt converts [sdkmath.Int] to [*uint256.Int] safely.
// Returns for non-positive value or overflow.
func U256SafeFromSdkInt(x sdkmath.Int) (*uint256.Int, error) {
	return U256SafeFromBig(x.BigInt())
}
