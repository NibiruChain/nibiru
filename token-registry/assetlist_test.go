package tokenregistry_test

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	tokenregistry "github.com/NibiruChain/nibiru/v2/token-registry"
)

type Suite struct {
	suite.Suite
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(Suite))
}

func (s *Suite) TestImagesPresent() {
	assetList := tokenregistry.NibiruAssetList()
	for _, token := range assetList.Assets {
		s.Run(token.Name, func() {
			token = token.GitHubify()

			// Make sure all local images in Token.LogoURIs exist
			if token.LogoURIs != nil {
				png, svg := token.LogoURIs.Png, token.LogoURIs.Svg
				if png != nil && strings.Contains(*png, "token-registry/img/") {
					_, err := os.Stat(*png)
					s.NoError(err)
				}
				if svg != nil && strings.Contains(*svg, "token-registry/img/") {
					_, err := os.Stat(*svg)
					s.NoError(err)
				}
			}

			// Make sure all local images in Token.Images exist
			for _, img := range token.Images {
				png, svg := img.Png, img.Svg
				if png != nil && strings.Contains(*png, "token-registry/img/") {
					_, err := os.Stat(*png)
					s.NoError(err)
				}
				if svg != nil && strings.Contains(*svg, "token-registry/img/") {
					_, err := os.Stat(*svg)
					s.NoError(err)
				}
			}
		})
	}
}
