// Abstractions for keeping track of token names and their corresponding
// decimal precision exponents.
package tokens

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	// sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	// "github.com/osmosis-labs/osmosis/v043_temp/address"
)

type Native struct {
	HumanDisplay        string // "mtrx"
	HumanExponent       int64  // 18
	Base                string // "amtrx"
	BaseDisplay         string // "amtrx"
	BaseExponent        int64  // 0
	DefaultBondDenom    string
	Bech32PrefixAccAddr string
}

func NewNativeToken(humanDisplay string, humanExponent int64, base string, baseDisplay string) Native {
	token := new(Native)
	token.HumanDisplay = humanDisplay
	token.HumanExponent = humanExponent
	token.Base = base
	token.BaseDisplay = baseDisplay
	token.BaseExponent = 0
	token.DefaultBondDenom = base
	// Bech32PrefixAccAddr defines the Bech32 prefix of an account's address
	token.Bech32PrefixAccAddr = humanDisplay
	return *token
}

// Metadata for non-native assets used in the chain.
type IBCToken struct {
	HumanDisplay     string // "atom"
	HumanExponent    int64  // 6
	Base             string // "ibc/27394FB092D2ECCD56123C74F36E4C1F926001CEADA9CA97EA622B25F41E5EB2"
	BaseDisplay      string // "uatom"
	BaseExponent     int64  // 0
	DefaultBondDenom string
}

func NewIBCToken(humanDisplay string, humanExponent int64, baseDisplay string, base string) IBCToken {
	token := new(IBCToken)
	token.HumanDisplay = humanDisplay
	token.HumanExponent = humanExponent
	token.Base = base
	token.BaseDisplay = baseDisplay
	token.BaseExponent = 0
	token.DefaultBondDenom = base
	return *token
}

const ibcPlaceholder string = "ibc/..."

var (
	NATIVE_MAP = map[string]Native{
		"mtrx": NewNativeToken("mtrx", 18, "amtrx", "amtrx"),
	}
	// TODO Use IBC addresses from https://docs.osmosis.zone/developing/assets/asset-info.html
	IBC_MAP = map[string]IBCToken{
		"usdm": NewIBCToken("usdm", 18, "ausdm", "ausdm"),
		"osmo": NewIBCToken("osmo", 6, "uosmo", ibcPlaceholder),
		"ust":  NewIBCToken("ust", 6, "uust", ibcPlaceholder),
		"ion":  NewIBCToken("ion", 6, "uion", ibcPlaceholder),
		"atom": NewIBCToken(
			"atom",  /* HumanDisplay */
			6,       /* HumanExponent */
			"uatom", /* BaseDisplay */
			"ibc/27394FB092D2ECCD56123C74F36E4C1F926001CEADA9CA97EA622B25F41E5EB2"),
		"juno": NewIBCToken(
			"juno", 6, "ujuno",
			"ibc/46B44899322F3CD854D2D46DEEF881958467CDD4B3B10086DA49296BBED94BED"),
		"luna": NewIBCToken(
			"luna", 6, "uluna",
			"ibc/0EF15DF2F02480ADE0BB6E85D9EBB5DAEA2836D3860E9F97F9AADE4F57A31AA0"),
	}
)

func RegisterDenoms() {
	for denom, tokenRegistry := range NATIVE_MAP {
		err := sdk.RegisterDenom(denom, sdk.OneDec())
		if err != nil {
			panic(err)
		}
		err = sdk.RegisterDenom(
			tokenRegistry.Base,
			sdk.NewDecWithPrec(1, tokenRegistry.HumanExponent))
		if err != nil {
			panic(err)
		}
	}
}
