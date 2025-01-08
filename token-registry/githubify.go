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

// localImageToGitHub converts a path to a local image into a GitHub download
// link in the NibiruChain/nibiru repository.
func localImageToGitHub(local string) string {
	trimmed := strings.TrimPrefix(local, "./img/")
	return "https://raw.githubusercontent.com/NibiruChain/nibiru/main/token-registry/img/" + trimmed
}

// isLocalImage returns true if an image URI is meant for a local file in
// the "token-registry/img" directory.
func isLocalImage(maybeLocal *string) bool {
	if maybeLocal == nil {
		return false
	}
	return strings.HasPrefix(*maybeLocal, "./img")
}

func (logouris LogoURIs) GitHubify() *LogoURIs {
	out := new(LogoURIs)
	out.Png, out.Svg = logouris.Png, logouris.Svg
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
	out.Png, out.Svg = ai.Png, ai.Svg
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
