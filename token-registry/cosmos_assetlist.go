package tokenregistry

import "bytes"

func NibiruAssetList() AssetList {
	var tokens = TOKENS()
	for idx, token := range tokens {
		tokens[idx] = token.GitHubify()
	}

	return AssetList{
		Schema:    "../assetlist.schema.json",
		ChainName: "nibiru",
		Assets:    tokens,
	}
}

func PointImagesToCosmosChainRegistry(assetListJson []byte) (out []byte) {
	oldImgPath := []byte("raw.githubusercontent.com/NibiruChain/nibiru/main/token-registry/img")
	newImgPath := []byte("raw.githubusercontent.com/cosmos/chain-registry/master/nibiru/images")
	return bytes.ReplaceAll(assetListJson, oldImgPath, newImgPath)
}

func TOKENS() []Token {
	return []Token{
		{
			Name:                "Nibiru",
			Description:         "The native token of Nibiru blockchain",
			ExtendedDescription: some("Nibiru is a smart contract ecosystem with a high-performance, EVM-equivalent execution layer. Nibiru is engineered to meet the growing demand for versatile, scalable, and easy-to-use Web3 applications."),
			Socials: &SocialLinks{
				Website: some("https://nibiru.fi"),
				Twitter: some("https://twitter.com/nibiruchain"),
			},
			DenomUnits: []DenomUnit{
				{Denom: "unibi", Exponent: 0},
				{Denom: "nibi", Exponent: 6},
				{Denom: "attonibi", Exponent: 18},
			},
			Base:    "unibi",
			Display: "nibi",
			Symbol:  "NIBI",
			LogoURIs: &LogoURIs{
				Png: some("https://raw.githubusercontent.com/cosmos/chain-registry/master/nibiru/images/nibiru.png"),
				Svg: some("https://raw.githubusercontent.com/cosmos/chain-registry/master/nibiru/images/nibiru.svg"),
			},
			CoingeckoID: some("nibiru"),
			Images: []AssetImage{
				{
					Png: some("https://raw.githubusercontent.com/cosmos/chain-registry/master/nibiru/images/nibiru.png"),
					Svg: some("https://raw.githubusercontent.com/cosmos/chain-registry/master/nibiru/images/nibiru.svg"),
					Theme: &ImageTheme{
						PrimaryColorHex: some("#14c0ce"),
					},
				},
			},
			TypeAsset: TypeAsset_SDKCoin,
		},
		{
			Name:                "Liquid Staked Nibiru (Eris)",
			Description:         "Liquid Staked Nibiru (Eris)",
			ExtendedDescription: some("Liquid Staked Nibiru, powered by Eris Protocol's amplifier contracts. Nibiru is a smart contract ecosystem with a high-performance, EVM-equivalent execution layer. Nibiru is engineered to meet the growing demand for versatile, scalable, and easy-to-use Web3 applications."),
			Socials: &SocialLinks{
				Website: some("https://nibiru.fi/docs/learn/liquid-stake/"),
				Twitter: some("https://x.com/eris_protocol"),
			},
			DenomUnits: []DenomUnit{
				{Denom: "tf/nibi1udqqx30cw8nwjxtl4l28ym9hhrp933zlq8dqxfjzcdhvl8y24zcqpzmh8m/ampNIBI", Exponent: 0},
				{Denom: "stNIBI", Exponent: 6},
			},
			Base:    "tf/nibi1udqqx30cw8nwjxtl4l28ym9hhrp933zlq8dqxfjzcdhvl8y24zcqpzmh8m/ampNIBI",
			Display: "stNIBI",
			Symbol:  "stNIBI",
			LogoURIs: &LogoURIs{
				Png: some("https://raw.githubusercontent.com/cosmos/chain-registry/master/nibiru/images/stnibi-logo-circle-500x500.png"),
				Svg: some("https://raw.githubusercontent.com/cosmos/chain-registry/master/nibiru/images/stnibi-logo-circle-500x500.svg"),
			},
			Images: []AssetImage{
				{
					Png: some("https://raw.githubusercontent.com/cosmos/chain-registry/master/nibiru/images/stnibi-logo-circle-500x500.png"),
					Svg: some("https://raw.githubusercontent.com/cosmos/chain-registry/master/nibiru/images/stnibi-logo-circle-500x500.svg"),
					Theme: &ImageTheme{
						PrimaryColorHex: some("#14c0ce"),
					},
				},
			},
			Traces: []Trace{
				{
					Type: "liquid-stake",
					Counterparty: Counterparty{
						ChainName: "nibiru",
						BaseDenom: "unibi",
					},
					Provider: some("Eris Protocol"),
				},
			},
			TypeAsset: TypeAsset_SDKCoin,
		},
		{
			Name:        "Noble USDC",
			Description: "Noble USDC on Nibiru",
			DenomUnits: []DenomUnit{
				{Denom: "ibc/F082B65C88E4B6D5EF1DB243CDA1D331D002759E938A0F5CD3FFDC5D53B3E349", Exponent: 0},
				{Denom: "usdc", Exponent: 6},
			},
			Base:    "ibc/F082B65C88E4B6D5EF1DB243CDA1D331D002759E938A0F5CD3FFDC5D53B3E349",
			Display: "usdc",
			Symbol:  "USDC",
			Traces: []Trace{
				{
					Type: TraceType_IBC,
					Counterparty: Counterparty{
						ChainName: "noble",
						BaseDenom: "uusdc",
						ChannelID: some("channel-67"),
					},
					Chain: &TraceChainInfo{
						ChannelID: "channel-2",
						Path:      "transfer/channel-2/uusdc",
					},
				},
			},
			Images: []AssetImage{
				{
					ImageSync: &ImageSync{
						ChainName: "noble",
						BaseDenom: "uusdc",
					},
					Png: some("https://raw.githubusercontent.com/cosmos/chain-registry/master/_non-cosmos/ethereum/images/usdc.png"),
					Svg: some("https://raw.githubusercontent.com/cosmos/chain-registry/master/_non-cosmos/ethereum/images/usdc.svg"),
					Theme: &ImageTheme{
						Circle:          some(true),
						PrimaryColorHex: some("#2775CA"),
					},
				},
			},
			LogoURIs: &LogoURIs{
				Png: some("https://raw.githubusercontent.com/cosmos/chain-registry/master/_non-cosmos/ethereum/images/usdc.png"),
				Svg: some("https://raw.githubusercontent.com/cosmos/chain-registry/master/_non-cosmos/ethereum/images/usdc.svg"),
			},
			TypeAsset: TypeAsset_ICS20,
		},

		{
			Name:                "Astrovault token",
			Description:         "AXV",
			ExtendedDescription: some("AXV is the Astrovault token."),
			Socials: &SocialLinks{
				Website: some("https://astrovault.io/"),
				Twitter: some("https://x.com/axvdex"),
			},
			DenomUnits: []DenomUnit{
				{Denom: "tf/nibi1vetfuua65frvf6f458xgtjerf0ra7wwjykrdpuyn0jur5x07awxsfka0ga/axv", Exponent: 0},
				{Denom: "AXV", Exponent: 6},
			},
			Base:    "tf/nibi1vetfuua65frvf6f458xgtjerf0ra7wwjykrdpuyn0jur5x07awxsfka0ga/axv",
			Display: "AXV",
			Symbol:  "AXV",
			LogoURIs: &LogoURIs{
				Png: some("https://raw.githubusercontent.com/cosmos/chain-registry/master/neutron/images/axv.png"),
				Svg: some("https://raw.githubusercontent.com/cosmos/chain-registry/master/neutron/images/axv.svg"),
			},
			Images: []AssetImage{
				{
					ImageSync: &ImageSync{
						ChainName: "neutron",
						BaseDenom: "cw20:neutron10dxyft3nv4vpxh5vrpn0xw8geej8dw3g39g7nqp8mrm307ypssksau29af",
					},
					Png: some("https://raw.githubusercontent.com/cosmos/chain-registry/master/neutron/images/axv.png"),
					Svg: some("https://raw.githubusercontent.com/cosmos/chain-registry/master/neutron/images/axv.svg"),
				},
			},
			TypeAsset: TypeAsset_SDKCoin,
		},

		{
			Name:                "Astrovault Nibiru LST (xNIBI)",
			Description:         "Astrovault Nibiru LST (xNIBI)",
			TypeAsset:           TypeAsset_CW20,
			Address:             some("nibi1cehpv50vl90g9qkwwny8mw7txw79zs6f7wsfe8ey7dgp238gpy4qhdqjhm"),
			ExtendedDescription: some("xNIBI is a liquid staking derivative for NIBI created by Astrovault."),
			Socials: &SocialLinks{
				Website: some("https://astrovault.io/"),
				Twitter: some("https://x.com/axvdex"),
			},
			DenomUnits: []DenomUnit{
				{Denom: "cw20:nibi1cehpv50vl90g9qkwwny8mw7txw79zs6f7wsfe8ey7dgp238gpy4qhdqjhm", Exponent: 0},
				{Denom: "xNIBI", Exponent: 6},
			},
			Base:    "cw20:nibi1cehpv50vl90g9qkwwny8mw7txw79zs6f7wsfe8ey7dgp238gpy4qhdqjhm",
			Display: "xNIBI",
			Symbol:  "xNIBI",
			LogoURIs: &LogoURIs{
				Svg: some("./img/004_astrovault-xnibi.svg"),
			},
			Images: []AssetImage{
				{
					Svg: some("./img/004_astrovault-xnibi.svg"),
				},
			},
		},

		{
			Description: "uoprek",
			DenomUnits: []DenomUnit{
				{Denom: "tf/nibi149m52kn7nvsg5nftvv4fh85scsavpdfxp5nr7zasz97dum89dp5qkyhy0t/uoprek", Exponent: 0},
			},
			Base:      "tf/nibi149m52kn7nvsg5nftvv4fh85scsavpdfxp5nr7zasz97dum89dp5qkyhy0t/uoprek",
			Name:      "uoprek",
			Display:   "tf/nibi149m52kn7nvsg5nftvv4fh85scsavpdfxp5nr7zasz97dum89dp5qkyhy0t/uoprek",
			Symbol:    "UOPREK",
			TypeAsset: TypeAsset_SDKCoin,
		},
		{
			Description: "utestate",
			DenomUnits: []DenomUnit{
				{Denom: "tf/nibi1lp28kx3gz0prsztl024z730ufkg3alahaq3e7a6gae22nk0dqdvsyrrgqw/utestate", Exponent: 0},
			},
			Base:      "tf/nibi1lp28kx3gz0prsztl024z730ufkg3alahaq3e7a6gae22nk0dqdvsyrrgqw/utestate",
			Name:      "utestate",
			Display:   "tf/nibi1lp28kx3gz0prsztl024z730ufkg3alahaq3e7a6gae22nk0dqdvsyrrgqw/utestate",
			Symbol:    "UTESTATE",
			TypeAsset: TypeAsset_SDKCoin,
		},
		{
			Description: "npp",
			DenomUnits: []DenomUnit{
				{Denom: "tf/nibi1xpp7yn0tce62ffattws3gpd6v0tah0mlevef3ej3r4pnfvsehcgqk3jvxq/NPP", Exponent: 0},
			},
			Base:      "tf/nibi1xpp7yn0tce62ffattws3gpd6v0tah0mlevef3ej3r4pnfvsehcgqk3jvxq/NPP",
			Name:      "npp",
			Display:   "tf/nibi1xpp7yn0tce62ffattws3gpd6v0tah0mlevef3ej3r4pnfvsehcgqk3jvxq/NPP",
			Symbol:    "NPP",
			TypeAsset: TypeAsset_SDKCoin,
		},
	}
}
