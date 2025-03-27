package tokenregistry_test

import (
	// The `_ "embed"` import adds access to files embedded in the running Go
	// program (smart contracts).
	_ "embed"
	"encoding/json"
	"os"
	"os/exec"
	"path"

	"github.com/stretchr/testify/suite"

	tokenregistry "github.com/NibiruChain/nibiru/v2/token-registry"
)

var (
	//go:embed official_erc20s.json
	officialErc20sJson []byte

	//go:embed official_bank_coins.json
	officialBankCoinsJson []byte
)

var _ suite.SetupAllSuite = (*Suite)(nil)

func (s *Suite) SetupSuite() {
	rootPath, err := tokenregistry.FindRootPath()
	s.Require().NoError(err)

	// go run token-registry/main/main.go
	absPath := path.Join(rootPath, "token-registry/main/main.go")
	_, err = exec.Command("go", "run", absPath).Output()
	s.Require().NoError(err)
}

// TestErc20Images ensures that the embedded JSON file correctly unmarshals into the Erc20 struct.
func (s *Suite) TestErc20Images() {
	var tokens []tokenregistry.TokenOfficial
	err := json.Unmarshal(officialErc20sJson, &tokens)
	s.NoError(err, "Failed to unmarshal official_erc20s.json")
	s.NotEmpty(tokens, "Expected at least one token in the list")

	var bankCoins []tokenregistry.TokenOfficial
	err = json.Unmarshal(officialBankCoinsJson, &bankCoins)
	s.NoError(err, "Failed to unmarshal official_bank_coins.json")
	s.NotEmpty(tokens, "Expected at least one token in the list")

	tokens = append(tokens, bankCoins...)
	for _, token := range tokens {
		// Example: Validate first token fields
		s.NotEmpty(token.ContractAddr, "Contract address should not be empty")
		s.NotEmpty(token.DisplayName, "Display name should not be empty")
		s.NotEmpty(token.Symbol, "Symbol should not be empty")

		s.NotEmpty(token.LogoSrc, "Logo source should not be empty")
		s.Require().Truef(
			tokenregistry.IsGitHubImage(some(token.LogoSrc)),
			"Invalid image URL for token %+v", token,
		)

		localImgPath := tokenregistry.GitHubImageToLocal(token.LogoSrc)
		_, err := os.Stat(localImgPath)
		s.NoErrorf(err, "Local image file does not exist for token %s at path: %s", token.Symbol, localImgPath)
	}
}
