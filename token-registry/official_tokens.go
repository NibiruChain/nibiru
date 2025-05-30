package tokenregistry

import (
	"encoding/json"
	"os"
	"os/exec"
	"path"
	"strings"
)

type TokenOfficial struct {
	ContractAddr string     `json:"contractAddr"`
	DisplayName  string     `json:"displayName"`
	Symbol       string     `json:"symbol"`
	LogoSrc      string     `json:"logoSrc"`
	PriceInfo    *PriceInfo `json:"priceInfo,omitempty"`
}

const (
	SAVE_PATH_OFFICIAL_ERC20S     = "token-registry/official_erc20s.json"
	SAVE_PATH_OFFICIAL_BANK_COINS = "token-registry/official_bank_coins.json"
)

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

func LoadERC20s() (tokens []TokenOfficial, err error) {
	rootPath, err := FindRootPath()
	if err != nil {
		return tokens, err
	}

	fpath := path.Join(rootPath, SAVE_PATH_OFFICIAL_ERC20S)
	bz, err := os.ReadFile(fpath)
	if err != nil {
		return tokens, err
	}
	err = json.Unmarshal(bz, &tokens)
	return tokens, err
}

func LoadBankCoins() ([]TokenOfficial, error) {
	rootPath, err := FindRootPath()
	if err != nil {
		return nil, err
	}

	fpath := path.Join(rootPath, SAVE_PATH_OFFICIAL_BANK_COINS)
	bz, err := os.ReadFile(fpath)
	if err != nil {
		return nil, err
	}

	var tokens []TokenOfficial
	err = json.Unmarshal(bz, &tokens)
	return tokens, err
}
