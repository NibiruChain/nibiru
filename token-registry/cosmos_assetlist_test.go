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
					localImgPath := "./img/" + strings.Split(*png, "/img/")[1]
					_, err := os.Stat(localImgPath)
					s.NoError(err)
				}
				if svg != nil && strings.Contains(*svg, "token-registry/img/") {
					localImgPath := "./img/" + strings.Split(*svg, "/img/")[1]
					_, err := os.Stat(localImgPath)
					s.NoError(err)
				}
			}

			// Make sure all local images in Token.Images exist
			for _, img := range token.Images {
				png, svg := img.Png, img.Svg
				if png != nil && strings.Contains(*png, "token-registry/img/") {
					localImgPath := "./img/" + strings.Split(*png, "/img/")[1]
					_, err := os.Stat(localImgPath)
					s.NoError(err)
				}
				if svg != nil && strings.Contains(*svg, "token-registry/img/") {
					localImgPath := "./img/" + strings.Split(*svg, "/img/")[1]
					_, err := os.Stat(localImgPath)
					s.NoError(err)
				}
			}
		})
	}
}

func some(s string) *string {
	return &s
}

func (s *Suite) TestLogoURIsGitHubify() {
	tests := []struct {
		name     string
		input    tokenregistry.LogoURIs
		expected tokenregistry.LogoURIs
	}{
		{
			name: "Png and Svg are local paths",
			input: tokenregistry.LogoURIs{
				Png: some("./img/logo.png"),
				Svg: some("./img/logo.svg"),
			},
			expected: tokenregistry.LogoURIs{
				Png: some("https://raw.githubusercontent.com/NibiruChain/nibiru/main/token-registry/img/logo.png"),
				Svg: some("https://raw.githubusercontent.com/NibiruChain/nibiru/main/token-registry/img/logo.svg"),
			},
		},
		{
			name: "Png is local, Svg is external",
			input: tokenregistry.LogoURIs{
				Png: some("./img/logo.png"),
				Svg: some("https://example.com/logo.svg"),
			},
			expected: tokenregistry.LogoURIs{
				Png: some("https://raw.githubusercontent.com/NibiruChain/nibiru/main/token-registry/img/logo.png"),
				Svg: some("https://example.com/logo.svg"),
			},
		},
		{
			name: "Both Png and Svg are external URLs",
			input: tokenregistry.LogoURIs{
				Png: some("https://example.com/logo.png"),
				Svg: some("https://example.com/logo.svg"),
			},
			expected: tokenregistry.LogoURIs{
				Png: some("https://example.com/logo.png"),
				Svg: some("https://example.com/logo.svg"),
			},
		},
		{
			name: "Both Png and Svg are nil",
			input: tokenregistry.LogoURIs{
				Png: nil,
				Svg: nil,
			},
			expected: tokenregistry.LogoURIs{
				Png: nil,
				Svg: nil,
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result := tt.input.GitHubify()
			s.Equal(tt.expected, *result)
		})
	}
}

func (s *Suite) TestAssetImageGitHubify() {
	tests := []struct {
		name     string
		input    tokenregistry.AssetImage
		expected tokenregistry.AssetImage
	}{
		{
			name: "Png and Svg are local paths",
			input: tokenregistry.AssetImage{
				Png: some("./img/asset.png"),
				Svg: some("./img/asset.svg"),
			},
			expected: tokenregistry.AssetImage{
				Png: some("https://raw.githubusercontent.com/NibiruChain/nibiru/main/token-registry/img/asset.png"),
				Svg: some("https://raw.githubusercontent.com/NibiruChain/nibiru/main/token-registry/img/asset.svg"),
			},
		},
		{
			name: "Png is local, Svg is external",
			input: tokenregistry.AssetImage{
				Png: some("./img/asset.png"),
				Svg: some("https://example.com/asset.svg"),
			},
			expected: tokenregistry.AssetImage{
				Png: some("https://raw.githubusercontent.com/NibiruChain/nibiru/main/token-registry/img/asset.png"),
				Svg: some("https://example.com/asset.svg"),
			},
		},
		{
			name: "Both Png and Svg are external URLs",
			input: tokenregistry.AssetImage{
				Png: some("https://example.com/asset.png"),
				Svg: some("https://example.com/asset.svg"),
			},
			expected: tokenregistry.AssetImage{
				Png: some("https://example.com/asset.png"),
				Svg: some("https://example.com/asset.svg"),
			},
		},
		{
			name: "Both Png and Svg are nil",
			input: tokenregistry.AssetImage{
				Png: nil,
				Svg: nil,
			},
			expected: tokenregistry.AssetImage{
				Png: nil,
				Svg: nil,
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			result := tt.input.GitHubify()
			s.Equal(tt.expected, result)
		})
	}
}
