package evm

import "strings"

const (
	// MainnetSaiUSDC is the Sai USDC ERC20 contract on Nibiru mainnet.
	MainnetSaiUSDC = "0x357c97796aA8e2A6b5A64b1Fc19d5baB9668E190"
	// MainnetLegacyUSDC is the legacy LayerZero Stargate USDC.e contract.
	MainnetLegacyUSDC = "0x0829F361A05D993d5CEb035cA6DF3446b060970b"
)

// ResolveUSDCAlias resolves mainnet USDC convenience aliases. Explicit addresses,
// bank denoms, and other inputs pass through unchanged. Callers on other networks
// should supply the deployed token address or denom instead of these aliases.
func ResolveUSDCAlias(token string) string {
	switch {
	case strings.EqualFold(token, "USDC"):
		return MainnetSaiUSDC
	case strings.EqualFold(token, "USDC.e"):
		return MainnetLegacyUSDC
	default:
		return token
	}
}
