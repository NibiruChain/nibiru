package plan

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDockerReferenceValidation(t *testing.T) {
	digest := "sha256:" + strings.Repeat("a", 64)
	for _, tc := range []struct {
		name  string
		image string
		err   string
	}{
		{"tag", "ghcr.io/nibiruchain/nibiru:2.21.0", ""},
		{"digest", "ghcr.io/nibiruchain/nibiru@" + digest, ""},
		{"tag and digest", "ghcr.io/nibiruchain/nibiru:2.21.0@" + digest, ""},
		{"registry port", "localhost:5000/nibiru:2.21.0", ""},
		{"short name", "nibiru:2.21.0", ""},
		{"missing version", "ghcr.io/nibiruchain/nibiru", "explicit tag or digest"},
		{"empty", "", "invalid docker image reference"},
		{"URL", "https://ghcr.io/nibiruchain/nibiru:2.21.0", "invalid docker image reference"},
		{"uppercase repository", "ghcr.io/NibiruChain/nibiru:2.21.0", "invalid docker image reference"},
		{"whitespace", " ghcr.io/nibiruchain/nibiru:2.21.0", "invalid docker image reference"},
		{"empty tag", "ghcr.io/nibiruchain/nibiru:", "invalid docker image reference"},
		{"short digest", "ghcr.io/nibiruchain/nibiru@sha256:abc", "invalid docker image reference"},
		{"unsupported digest", "ghcr.io/nibiruchain/nibiru@unknown:" + strings.Repeat("a", 64), "invalid docker image reference"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			binaries := BinaryDownloadURLMap{
				"any":    "https://example.com/nibid?checksum=sha256:" + strings.Repeat("a", 64),
				"docker": tc.image,
			}
			err := binaries.ValidateBasic()
			if tc.err == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.err)
				require.ErrorContains(t, err, "binaries[docker]")
			}
		})
	}

	t.Run("Docker alone is insufficient", func(t *testing.T) {
		err := (BinaryDownloadURLMap{"docker": "ghcr.io/nibiruchain/nibiru:2.21.0"}).ValidateBasic()
		require.ErrorContains(t, err, "no platform binary entries")
	})
}

func TestValidateFullWithDocker(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "nibid.zip")
	require.NoError(t, NewTestZip(NewTestFile("bin/nibid", "binary fixture"), NewTestFile("readme.txt", "release")).SaveAs(archive))
	goodURL := makeFileURL(t, archive)
	missingBinary := filepath.Join(dir, "missing-binary.zip")
	require.NoError(t, NewTestZip(NewTestFile("readme.txt", "no executable"), NewTestFile("license.txt", "license")).SaveAs(missingBinary))

	for _, tc := range []struct {
		name string
		url  string
		err  string
	}{
		{"valid archive", goodURL, ""},
		{"missing checksum", "file://" + archive, "missing checksum"},
		{"malformed checksum", "file://" + archive + "?checksum=sha256:abc", "checksum"},
		{"incorrect checksum", "file://" + archive + "?checksum=sha256:" + strings.Repeat("a", 64), "Checksums did not match"},
		{"failed download", "file://" + filepath.Join(dir, "absent.zip") + "?checksum=sha256:" + strings.Repeat("a", 64), "error downloading binary"},
		{"missing executable", makeFileURL(t, missingBinary), "does not contain a bin/nibid or nibid file"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			info := Info{Binaries: BinaryDownloadURLMap{
				"linux/amd64": tc.url,
				"linux/arm64": goodURL,
				// An unreachable registry also proves validation makes no image download.
				"docker": "registry.invalid/nibiruchain/nibiru:2.21.0",
			}}
			err := info.ValidateFull("nibid")
			if tc.err == "" {
				require.NoError(t, err)
				require.NoError(t, info.Binaries.CheckURLs("nibid"))
			} else {
				require.ErrorContains(t, err, tc.err)
			}
		})
	}
}
