package tokenregistry

import (
	"encoding/json"
	"os/exec"
	"strings"
)

type TokenOfficial struct {
	ContractAddr string     `json:"contractAddr"`
	DisplayName  string     `json:"displayName"`
	Symbol       string     `json:"symbol"`
	LogoSrc      string     `json:"logoSrc"`
	PriceInfo    *PriceInfo `json:"priceInfo,omitempty"`
}

type PriceInfo struct {
	// "source" identifies where to get the USD price
	Source string `json:"source"`
	// "priceId" is key used to fetch the price data from the given source.
	PriceId string `json:"priceId"`
}

// FindRootPath returns the absolute path of the repository root
// This is retrievable with: go list -m -f {{.Dir}}
func FindRootPath() (string, error) {
	// rootPath, _ := exec.Command("go list -m -f {{.Dir}}").Output()
	// This returns the path to the root of the project.
	rootPathBz, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}").Output()
	if err != nil {
		return "", err
	}
	rootPath := strings.Trim(string(rootPathBz), "\n")
	return rootPath, nil
}

func (to *TokenOfficial) GitHubify() *TokenOfficial {
	if IsLocalImage(&to.LogoSrc) {
		to.LogoSrc = LocalImageToGitHub(to.LogoSrc)
	}
	return to
}

func ParseOfficialSaveBz(tokens []TokenOfficial) ([]byte, error) {
	parsedTokens := make([]TokenOfficial, len(tokens))
	for i, token := range tokens {
		parsedTokens[i] = *(&token).GitHubify()
	}
	return json.MarshalIndent(parsedTokens, "", "  ")
}

var ERC20S []TokenOfficial = []TokenOfficial{
	{
		ContractAddr: "0x0CaCF669f8446BeCA826913a3c6B96aCD4b02a97",
		DisplayName:  "Wrapped Nibiru",
		Symbol:       "WNIBI",
		LogoSrc:      "./img/000_nibiru-evm.png",
		PriceInfo: &PriceInfo{
			Source:  "bybit",
			PriceId: "NIBIUSDT",
		},
	},
	{
		ContractAddr: "0xcA0a9Fb5FBF692fa12fD13c0A900EC56Bb3f0a7b",
		DisplayName:  "Liquid Staked Nibiru (Wrapped)",
		Symbol:       "stNIBI",
		LogoSrc:      "./img/001_stnibi-evm.png",
	},
	{
		ContractAddr: "0x7168634Dd1ee48b1C5cC32b27fD8Fc84E12D00E6",
		DisplayName:  "Astrovault (Wrapped)",
		Symbol:       "AXV",
		LogoSrc:      "./img/003_astrovault-axv.png",
	},
}

var BANK_COINS []TokenOfficial = []TokenOfficial{
	{
		ContractAddr: "unibi",
		DisplayName:  "Nibiru",
		Symbol:       "NIBI",
		LogoSrc:      "./img/000_nibiru.png",
		PriceInfo: &PriceInfo{
			Source:  "bybit",
			PriceId: "NIBIUSDT",
		},
	},
	{
		ContractAddr: "tf/nibi1udqqx30cw8nwjxtl4l28ym9hhrp933zlq8dqxfjzcdhvl8y24zcqpzmh8m/ampNIBI",
		DisplayName:  "Liquid Staked Nibiru",
		Symbol:       "stNIBI",
		LogoSrc:      "./img/001_stnibi-bank.png",
	},
	{
		ContractAddr: "tf/nibi1vetfuua65frvf6f458xgtjerf0ra7wwjykrdpuyn0jur5x07awxsfka0ga/axv",
		DisplayName:  "Astrovault",
		Symbol:       "AXV",
		LogoSrc:      "./img/003_astrovault-axv-bank.png",
	},
	{
		ContractAddr: "ibc/F082B65C88E4B6D5EF1DB243CDA1D331D002759E938A0F5CD3FFDC5D53B3E349",
		DisplayName:  "Noble USDC",
		Symbol:       "USDC.noble",
		LogoSrc:      "./img/002_usdc-noble.png",
		PriceInfo: &PriceInfo{
			Source:  "bybit",
			PriceId: "USDCUSDT",
		},
	},
}
