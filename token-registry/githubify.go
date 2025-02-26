package tokenregistry

import "strings"

// GitHubify replaces local image paths with GitHub raw URLs.
//
// Example: "./img/token.png" →  "https://raw.githubusercontent.com/NibiruChain/nibiru/main/token-registry/img/token.png"
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

// LocalImageToGitHub converts a path to a local image into a GitHub download
// link in the NibiruChain/nibiru repository.
func LocalImageToGitHub(local string) string {
	prefix := "./img/"
	trimmed := strings.TrimPrefix(local, prefix)
	return "https://raw.githubusercontent.com/NibiruChain/nibiru/main/token-registry/img/" + trimmed
}

// IsLocalImage returns true if an image URI is meant for a local file in
// the "token-registry/img" directory.
func IsLocalImage(maybeLocal *string) bool {
	if maybeLocal == nil {
		return false
	}
	return strings.HasPrefix(*maybeLocal, "./img")
}

// GitHubify replaces local image paths with GitHub raw URLs.
//
// Example: "./img/token.png" →  "https://raw.githubusercontent.com/NibiruChain/nibiru/main/token-registry/img/token.png"
func (logouris LogoURIs) GitHubify() *LogoURIs {
	out := new(LogoURIs)
	out.Png, out.Svg = logouris.Png, logouris.Svg
	if logouris.Png != nil && IsLocalImage(logouris.Png) {
		out.Png = some(LocalImageToGitHub(*logouris.Png))
	}
	if logouris.Svg != nil && IsLocalImage(logouris.Svg) {
		out.Svg = some(LocalImageToGitHub(*logouris.Svg))
	}
	return out
}

// GitHubify replaces local image paths with GitHub raw URLs.
//
// Example: "./img/token.png" →  "https://raw.githubusercontent.com/NibiruChain/nibiru/main/token-registry/img/token.png"
func (ai AssetImage) GitHubify() AssetImage {
	out := AssetImage{}
	out.Png, out.Svg = ai.Png, ai.Svg
	if ai.Png != nil && IsLocalImage(ai.Png) {
		out.Png = some(LocalImageToGitHub(*ai.Png))
	}
	if ai.Svg != nil && IsLocalImage(ai.Svg) {
		out.Svg = some(LocalImageToGitHub(*ai.Svg))
	}
	if ai.Theme != nil {
		out.Theme = ai.Theme
	}
	if ai.ImageSync != nil {
		out.ImageSync = ai.ImageSync
	}
	return out
}

// GitHubImageToLocal converts a GitHub raw image URL into a local image path.
func GitHubImageToLocal(githubURL string) string {
	prefix := "https://raw.githubusercontent.com/NibiruChain/nibiru/main/token-registry/img/"
	trimmed := strings.TrimPrefix(githubURL, prefix)
	return "./img/" + trimmed
}

// IsGitHubImage returns true if an image URI is a GitHub raw URL from the
// NibiruChain/nibiru repository.
func IsGitHubImage(maybeGitHub *string) bool {
	if maybeGitHub == nil {
		return false
	}
	prefix := "https://raw.githubusercontent.com/NibiruChain/nibiru/main/token-registry/img/"
	return strings.HasPrefix(*maybeGitHub, prefix)
}

// GitHubifyReverse replaces GitHub raw URLs with local image paths.
//
// Example:
// "https://raw.githubusercontent.com/NibiruChain/nibiru/main/token-registry/img/token.png" →  "./img/token.png"
func (logouris LogoURIs) GitHubifyReverse() *LogoURIs {
	out := new(LogoURIs)
	out.Png, out.Svg = logouris.Png, logouris.Svg
	if logouris.Png != nil && IsGitHubImage(logouris.Png) {
		out.Png = some(GitHubImageToLocal(*logouris.Png))
	}
	if logouris.Svg != nil && IsGitHubImage(logouris.Svg) {
		out.Svg = some(GitHubImageToLocal(*logouris.Svg))
	}
	return out
}

// GitHubifyReverse replaces GitHub raw URLs with local image paths.
//
// Example:
// "https://raw.githubusercontent.com/NibiruChain/nibiru/main/token-registry/img/token.png" →  "./img/token.png"
func (ai AssetImage) GitHubifyReverse() AssetImage {
	out := AssetImage{}
	out.Png, out.Svg = ai.Png, ai.Svg
	if ai.Png != nil && IsGitHubImage(ai.Png) {
		out.Png = some(GitHubImageToLocal(*ai.Png))
	}
	if ai.Svg != nil && IsGitHubImage(ai.Svg) {
		out.Svg = some(GitHubImageToLocal(*ai.Svg))
	}
	if ai.Theme != nil {
		out.Theme = ai.Theme
	}
	if ai.ImageSync != nil {
		out.ImageSync = ai.ImageSync
	}
	return out
}

// GitHubifyReverse replaces GitHub raw URLs with local image paths.
//
// Example:
// "https://raw.githubusercontent.com/NibiruChain/nibiru/main/token-registry/img/token.png" →  "./img/token.png"
func (t Token) GitHubifyReverse() Token {
	out := t

	if out.LogoURIs != nil {
		out.LogoURIs = out.LogoURIs.GitHubifyReverse()
	}

	for imgIdx, img := range out.Images {
		out.Images[imgIdx] = img.GitHubifyReverse()
	}

	return out
}
