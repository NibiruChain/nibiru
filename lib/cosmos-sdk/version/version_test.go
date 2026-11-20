package version_test

import (
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"

	cmtcli "github.com/NibiruChain/nibiru/v2/bft/libs/cli"

	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/testutil"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/version"
)

func TestNewInfo(t *testing.T) {
	info := version.NewInfo()
	want := fmt.Sprintf(`: 
git commit: 
build tags: 
%s`, fmt.Sprintf("go version %s %s/%s", runtime.Version(), runtime.GOOS, runtime.GOARCH))
	require.Equal(t, want, info.String())
}

func TestInfo_String(t *testing.T) {
	info := version.Info{
		Name:             "testapp",
		AppName:          "testappd",
		Version:          "1.0.0",
		GitCommit:        "1b78457135a4104bc3af97f20654d49e2ea87454",
		BuildTags:        "netgo,ledger",
		GoVersion:        "go version go1.14 linux/amd64",
		CosmosSdkVersion: "0.42.5",
	}
	want := `testapp: 1.0.0
git commit: 1b78457135a4104bc3af97f20654d49e2ea87454
build tags: netgo,ledger
go version go1.14 linux/amd64`
	require.Equal(t, want, info.String())
}

func Test_runVersionCmd(t *testing.T) {
	cmd := version.NewVersionCommand()
	_, mockOut := testutil.ApplyMockIO(cmd)

	cmd.SetArgs([]string{
		fmt.Sprintf("--%s=''", cmtcli.OutputFlag),
		"--long=false",
	})

	require.NoError(t, cmd.Execute())
	assert.Equal(t, "\n", mockOut.String())
	mockOut.Reset()

	cmd.SetArgs([]string{
		fmt.Sprintf("--%s=json", cmtcli.OutputFlag), "--long=true",
	})

	info := version.NewInfo()
	stringInfo, err := json.Marshal(info)
	require.NoError(t, err)
	require.NoError(t, cmd.Execute())

	// The JSON form keeps every existing field and order, and appends the
	// path report as the last field.
	out := strings.TrimSuffix(mockOut.String(), "\n")
	body, pathsJSON, found := strings.Cut(out, `,"binary_paths":`)
	require.True(t, found, out)
	require.Equal(t, string(stringInfo), body+"}")
	var paths version.PathReport
	require.NoError(t, json.Unmarshal([]byte(strings.TrimSuffix(pathsJSON, "}")), &paths))
	require.NotEmpty(t, paths.RunningFrom)
	require.Equal(t, "nibiru", paths.Names[0].Name)
	require.Equal(t, "nibid", paths.Names[1].Name)
	require.Contains(t, mockOut.String(), `"cosmos_sdk_version":"","binary_paths":{`)
	mockOut.Reset()

	// Text output is one YAML document that holds the same report as the
	// JSON form under binary_paths. Strict parsing rejects duplicate keys.
	cmd.SetArgs([]string{"--long=true", fmt.Sprintf("--%s=text", cmtcli.OutputFlag)})
	require.NoError(t, cmd.Execute())
	text := mockOut.String()
	require.NotContains(t, text, "also on PATH")
	require.NotContains(t, text, "\nrunning from: ")
	// Drop the stderr hint, which the mock writer shares with stdout.
	doc, _, _ := strings.Cut(text, "note: ")
	var parsed map[string]any
	require.NoError(t, yaml.UnmarshalStrict([]byte(doc), &parsed), doc)
	bp, ok := parsed["binary_paths"].(map[string]any)
	require.True(t, ok, doc)
	require.NotEmpty(t, bp["running_from"])
}
