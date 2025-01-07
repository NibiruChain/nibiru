package tokenregistry

import "strings"

func (t Token) GitHubify() Token {
	out := t

	if out.LogoURIs != nil {
		out.LogoURIs = out.LogoURIs.GitHubify()
	}

	for imgIdx, img := range out.Images {
		out.Images[imgIdx] = img.GitHubify()
	}

	return out
}

func localImageToGitHub(local string) string {
	trimmed := strings.TrimPrefix(local, "./img/")
	return "https://raw.githubusercontent.com/NibiruChain/nibiru/main/token-registry/img/" + trimmed
}

func isLocalImage(maybeLocal *string) bool {
	return strings.HasPrefix(*maybeLocal, "./img")
}

func (logouris LogoURIs) GitHubify() *LogoURIs {
	out := new(LogoURIs)
	if logouris.Png != nil && isLocalImage(logouris.Png) {
		out.Png = some(localImageToGitHub(*logouris.Png))
	}
	if logouris.Svg != nil && isLocalImage(logouris.Svg) {
		out.Svg = some(localImageToGitHub(*logouris.Svg))
	}
	return out
}

func (ai AssetImage) GitHubify() AssetImage {
	out := AssetImage{}
	if ai.Png != nil && isLocalImage(ai.Png) {
		out.Png = some(localImageToGitHub(*ai.Png))
	}
	if ai.Svg != nil && isLocalImage(ai.Svg) {
		out.Svg = some(localImageToGitHub(*ai.Svg))
	}
	if ai.Theme != nil {
		out.Theme = ai.Theme
	}
	if ai.ImageSync != nil {
		out.ImageSync = ai.ImageSync
	}
	return out
}
