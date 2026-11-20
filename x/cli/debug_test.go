package main_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/app"
	svrcmd "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/server/cmd"
	nibid "github.com/NibiruChain/nibiru/v2/x/cli"
)

const (
	testAcct = "nibi1ah8gqrtjllhc5ld4rxgl4uglvwl93ag0sh6e6v"
	testMsg  = `{"@type":"/cosmos.bank.v1beta1.MsgSend","from_address":"` +
		testAcct + `","to_address":"` + testAcct +
		`","amount":[{"denom":"unibi","amount":"1"}]}`
)

// runRoot runs the real root command with args and returns stdout and the
// error. stdin is optional.
func runRoot(t *testing.T, stdin string, args ...string) (string, error) {
	t.Helper()
	out, _, err := runRootBoth(t, stdin, args...)
	return out, err
}

// runRootBoth is runRoot that also returns stderr.
func runRootBoth(
	t *testing.T, stdin string, args ...string,
) (stdout, stderr string, err error) {
	t.Helper()
	rootCmd, _ := nibid.NewRootCmd()
	out := new(bytes.Buffer)
	rootCmd.SetOut(out)
	errBuf := new(bytes.Buffer)
	rootCmd.SetErr(errBuf)
	if stdin != "" {
		rootCmd.SetIn(strings.NewReader(stdin))
	}
	rootCmd.SetArgs(append([]string{"--home", t.TempDir()}, args...))
	err = svrcmd.Execute(rootCmd, "", app.DefaultNodeHome)
	return out.String(), errBuf.String(), err
}

func TestDebugStargateMatchesLegacyBuildStargateWasm(t *testing.T) {
	newOut, err := runRoot(t, "", "debug", "stargate", testMsg)
	require.NoError(t, err)
	legacyOut, err := runRoot(t, "", "build-stargate-wasm", testMsg)
	require.NoError(t, err)

	require.NotEmpty(t, newOut)
	require.Equal(t, legacyOut, newOut)
	require.True(t, strings.HasPrefix(newOut,
		`{"stargate":{"type_url":"/cosmos.bank.v1beta1.MsgSend","value":"`), newOut)
	require.True(t, strings.HasSuffix(newOut, "\"}}\n"))
}

func TestDebugStargateRejectsBadInput(t *testing.T) {
	_, err := runRoot(t, "", "debug", "stargate", `{"nope":1}`)
	require.Error(t, err)
	_, err = runRoot(t, "", "debug", "stargate")
	require.Error(t, err)
}

func TestDebugAddressSmartAndExplicit(t *testing.T) {
	out, err := runRoot(t, "", "debug", testAcct)
	require.NoError(t, err)
	require.Contains(t, out, `"detected": "addr"`)
	require.Contains(t, out, `"role": "account"`)

	// The explicit SDK subcommand keeps its plain-text format.
	out, err = runRoot(t, "", "debug", "addr", testAcct)
	require.NoError(t, err)
	_ = out // printed via cmd.Println (stderr by cobra convention)
}

func TestDebugReadsStdin(t *testing.T) {
	for _, args := range [][]string{{"debug", "-"}} {
		out, err := runRoot(t, testAcct+"\n", args...)
		require.NoError(t, err)
		require.Contains(t, out, `"detected": "addr"`)
	}
}

func TestDebugAbciQueryJSONRPC(t *testing.T) {
	body := `{"jsonrpc":"2.0","id":7,"method":"abci_query","params":{"path":"/cosmos.auth.v1beta1.Query/Account","data":"0a2b6e69626931616838677172746a6c6c6863356c64347278676c3475676c76776c3933616730736836653676"}}`
	out, err := runRoot(t, "", "debug", body)
	require.NoError(t, err)
	require.Contains(t, out, `"detected": "abci-query"`)
	require.Contains(t, out, testAcct)

	out, err = runRoot(t, "", "debug", "abci-query", "--path",
		"/cosmos.auth.v1beta1.Query/Account", "--data",
		"0a2b6e69626931616838677172746a6c6c6863356c64347278676c3475676c76776c3933616730736836653676")
	require.NoError(t, err)
	require.Contains(t, out, testAcct)
}

// base64-decode is a hidden legacy command that `ud cw3` calls. It must not
// panic without an argument, and must stay hidden.
func TestBase64DecodeLegacy(t *testing.T) {
	_, err := runRoot(t, "", "base64-decode")
	require.Error(t, err)

	rootCmd, _ := nibid.NewRootCmd()
	for _, name := range []string{"base64-decode", "build-stargate-wasm"} {
		c, _, err := rootCmd.Find([]string{name})
		require.NoError(t, err)
		require.Equal(t, name, c.Name())
		require.True(t, c.Hidden, name)
	}
}

const undelegateWrapped = `{"stargate":{"type_url":"/cosmos.staking.v1beta1.MsgUndelegate","value":"Cj9uaWJpMTdwOXJ6d25uZnhjanAzMnVuOXVnN3loaHpndGtodmw5amZrc3p0Z3c1dWg2OXdhYzJwZ3N5bjcwbmoSMm5pYml2YWxvcGVyMXdqNWtma25qa3BjNmpkMzByeHRtOHRweGZqZjd4cWx3eDM4YzdwGgwKBXVuaWJpEgMxMTE="}}`

const undelegateDecoded = `[
  {
    "type_url": "/cosmos.staking.v1beta1.MsgUndelegate",
    "value": {
      "amount": {
        "amount": "111",
        "denom": "unibi"
      },
      "delegator_address": "nibi17p9rzwnnfxcjp32un9ug7yhhzgtkhvl9jfksztgw5uh69wac2pgsyn70nj",
      "validator_address": "nibivaloper1wj5kfknjkpc6jd30rxtm8tpxfjf7xqlwx38c7p"
    }
  }
]
`

// The hidden `base64-decode` output is pinned: `ud cw3` parses it.
func TestBase64DecodeOutputPinned(t *testing.T) {
	// cmd.Println writes to the command's out writer, which the test sets.
	legacy, err := runRoot(t, "", "base64-decode", undelegateWrapped)
	require.NoError(t, err)
	require.Equal(t, undelegateDecoded, legacy)

	// `debug msg` shares the decoder and prints the same JSON on stdout.
	out, err := runRoot(t, "", "debug", "msg", undelegateWrapped)
	require.NoError(t, err)
	require.Equal(t, undelegateDecoded, out)
}

// --as accepts addr, and address as an undocumented alias.
func TestDebugAsAddrAndAlias(t *testing.T) {
	in := strings.Repeat("ab", 32)
	for _, as := range []string{"addr", "address"} {
		out, err := runRoot(t, "", "debug", "--as", as, in)
		require.NoError(t, err, as)
		require.Contains(t, out, `"detected": "addr"`, as)
		require.Contains(t, out, `"role": "wasm-contract"`, as)
	}
	// The help names addr, not address, as the kind.
	out, err := runRoot(t, "", "debug", "--help")
	require.NoError(t, err)
	require.Contains(t, out, "abci-query, addr, pubkey")
	require.NotContains(t, out, "--as address")
}
