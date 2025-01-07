package tokenregistry

import (
	"encoding/json"
)

// some: Helper to create pointers for literals
func some[T any](v T) *T {
	return &v
}

type Token struct {
	// A short description of the asset
	Description string `json:"description"`
	// An extended, detailed description of the asset (optional)
	ExtendedDescription *string `json:"extended_description,omitempty"`
	// Links to social platforms and official websites (optional)
	Socials *SocialLinks `json:"socials,omitempty"`
	// Units of denomination for the asset
	DenomUnits []DenomUnit `json:"denom_units"`
	// The base denomination for the asset (canonical name)
	Base string `json:"base"`
	// The human-readable name of the asset
	Name string `json:"name"`
	// The display name or symbol used in UI for the asset
	Display string `json:"display"`
	// The ticker or symbol of the asset
	Symbol string `json:"symbol"`
	// URIs for the asset's logo in different formats (optional)
	LogoURIs *LogoURIs `json:"logo_URIs,omitempty"`
	// Unique identifier for the asset on Coingecko (optional)
	CoingeckoID *string `json:"coingecko_id,omitempty"`
	// An array of image representations for the asset (optional)
	Images []AssetImage `json:"images,omitempty"`
	// Type of the asset (e.g., "sdk.coin", "ics20", "erc20")
	TypeAsset TypeAsset `json:"type_asset"`
	// Trace data for the asset (optional, for cross-chain or liquid staking assets)
	Traces []Trace `json:"traces,omitempty"`
}

type AssetList struct {
	Schema    string  `json:"$schema"`
	ChainName string  `json:"chain_name"`
	Assets    []Token `json:"assets"`
}

// String returns a "pretty" JSON version of the [AssetList].
func (a AssetList) String() string {
	jsonBz, _ := json.MarshalIndent(a, "", "  ")
	return string(jsonBz)
}

type SocialLinks struct {
	Website *string `json:"website,omitempty"`
	Twitter *string `json:"twitter,omitempty"`
}

type DenomUnit struct {
	Denom    string   `json:"denom"`
	Exponent int      `json:"exponent"`
	Aliases  []string `json:"aliases,omitempty"`
}

type LogoURIs struct {
	Png *string `json:"png,omitempty"`
	Svg *string `json:"svg,omitempty"`
}

type AssetImage struct {
	Png       *string     `json:"png,omitempty"`
	Svg       *string     `json:"svg,omitempty"`
	Theme     *ImageTheme `json:"theme,omitempty"`
	ImageSync *ImageSync  `json:"image_sync,omitempty"`
}

// ImageTheme represents theme customization for an image.
type ImageTheme struct {
	// Whether the image should appear in a circular format (optional)
	Circle *bool `json:"circle,omitempty"`
	// Primary color in hexadecimal format (optional)
	PrimaryColorHex *string `json:"primary_color_hex,omitempty"`
}

// ImageSync represents synchronization details for cross-chain assets.
type ImageSync struct {
	// Name of the chain associated with the image
	ChainName string `json:"chain_name"`
	// Base denomination of the synced asset
	BaseDenom string `json:"base_denom"`
}

// TypeAsset is an enum type for "type_asset". Valid values include "sdk.coin",
// "ics20", and "erc20".
type TypeAsset string

const (
	TypeAsset_SDKCoin TypeAsset = "sdk.coin"
	TypeAsset_ICS20   TypeAsset = "ics20"
	TypeAsset_ERC20   TypeAsset = "erc20"
)

// Trace represents trace data for cross-chain or liquid staking assets.
type Trace struct {
	// Type of trace (e.g., "ibc", "liquid-stake", "wrapped", "bridge")
	Type TraceType `json:"type"`
	// Counterparty information for the trace
	Counterparty Counterparty `json:"counterparty"`
	// Provider of the asset for liquid staking or cross-chain trace (optional)
	Provider *string `json:"provider,omitempty"`
	// Additional chain-level details (optional)
	Chain *TraceChainInfo `json:"chain,omitempty"`
}

// TraceType is an enum type for "trace.type" (e.g. "ibc", "liquid-stake",
// "wrapped", "bridge")
type TraceType string

const (
	TraceType_IBC         TraceType = "ibc"
	TraceType_LiquidStake TraceType = "liquid-stake"
	TraceType_Wrapped     TraceType = "wrapped"
	TraceType_Bridge      TraceType = "bridge"
)

// Counterparty represents the counterparty information for an asset trace.
type Counterparty struct {
	// Name of the counterparty chain
	ChainName string `json:"chain_name"`
	// Base denomination on the counterparty chain
	BaseDenom string `json:"base_denom"`
	// Channel ID used for communication (optional)
	ChannelID *string `json:"channel_id,omitempty"`
}

// TraceChainInfo represents additional chain-level details for an asset trace.
type TraceChainInfo struct {
	// Channel ID on the chain
	ChannelID string `json:"channel_id"`
	// Path used for asset transfer on the chain
	Path string `json:"path"`
}
