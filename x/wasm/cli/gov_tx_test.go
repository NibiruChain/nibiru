package cli

import (
	"bytes"
	"context"
	"encoding/hex"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/client"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/client/flags"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/codec"
	codectypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/codec/types"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/crypto/keyring"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	authtx "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/auth/tx"
	v1 "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/gov/types/v1"

	"github.com/NibiruChain/nibiru/v2/x/wasm/testdata"
	wasmtestutil "github.com/NibiruChain/nibiru/v2/x/wasm/testutil"
	"github.com/NibiruChain/nibiru/v2/x/wasm/types"
)

func TestParseAccessConfigUpdates(t *testing.T) {
	specs := map[string]struct {
		src    []string
		exp    []types.AccessConfigUpdate
		expErr bool
	}{
		"nobody": {
			src: []string{"1:nobody"},
			exp: []types.AccessConfigUpdate{{
				CodeID:                1,
				InstantiatePermission: types.AccessConfig{Permission: types.AccessTypeNobody},
			}},
		},
		"everybody": {
			src: []string{"1:everybody"},
			exp: []types.AccessConfigUpdate{{
				CodeID:                1,
				InstantiatePermission: types.AccessConfig{Permission: types.AccessTypeEverybody},
			}},
		},
		"any of addresses - single": {
			src: []string{"1:cosmos1vx8knpllrj7n963p9ttd80w47kpacrhuts497x"},
			exp: []types.AccessConfigUpdate{
				{
					CodeID: 1,
					InstantiatePermission: types.AccessConfig{
						Permission: types.AccessTypeAnyOfAddresses,
						Addresses:  []string{"cosmos1vx8knpllrj7n963p9ttd80w47kpacrhuts497x"},
					},
				},
			},
		},
		"any of addresses - multiple": {
			src: []string{"1:cosmos1vx8knpllrj7n963p9ttd80w47kpacrhuts497x,cosmos14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9s4hmalr"},
			exp: []types.AccessConfigUpdate{
				{
					CodeID: 1,
					InstantiatePermission: types.AccessConfig{
						Permission: types.AccessTypeAnyOfAddresses,
						Addresses:  []string{"cosmos1vx8knpllrj7n963p9ttd80w47kpacrhuts497x", "cosmos14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9s4hmalr"},
					},
				},
			},
		},
		"multiple code ids with different permissions": {
			src: []string{"1:cosmos1vx8knpllrj7n963p9ttd80w47kpacrhuts497x,cosmos14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9s4hmalr", "2:nobody"},
			exp: []types.AccessConfigUpdate{
				{
					CodeID: 1,
					InstantiatePermission: types.AccessConfig{
						Permission: types.AccessTypeAnyOfAddresses,
						Addresses:  []string{"cosmos1vx8knpllrj7n963p9ttd80w47kpacrhuts497x", "cosmos14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9s4hmalr"},
					},
				}, {
					CodeID: 2,
					InstantiatePermission: types.AccessConfig{
						Permission: types.AccessTypeNobody,
					},
				},
			},
		},
		"any of addresses - empty list": {
			src:    []string{"1:"},
			expErr: true,
		},
		"any of addresses - invalid address": {
			src:    []string{"1:foo"},
			expErr: true,
		},
		"any of addresses - duplicate address": {
			src:    []string{"1:cosmos1vx8knpllrj7n963p9ttd80w47kpacrhuts497x,cosmos1vx8knpllrj7n963p9ttd80w47kpacrhuts497x"},
			expErr: true,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			got, gotErr := parseAccessConfigUpdates(spec.src)
			if spec.expErr {
				require.Error(t, gotErr)
				return
			}
			require.NoError(t, gotErr)
			assert.Equal(t, spec.exp, got)
		})
	}
}

func TestParseCodeInfoFlags(t *testing.T) {
	correctSource := "https://github.com/CosmWasm/wasmd/blob/main/x/wasm/testdata/hackatom.wasm"
	correctBuilderRef := "cosmwasm/workspace-optimizer:0.12.9"

	wasmBin, err := os.ReadFile(wasmtestutil.FixturePath("testdata", "hackatom.wasm.gzip"))
	require.NoError(t, err)

	checksumStr := strings.ToUpper(testdata.ChecksumHackatom)

	specs := map[string]struct {
		args   []string
		expErr bool
	}{
		"source missing": {
			args:   []string{"--builder=" + correctBuilderRef, "--code-hash=" + checksumStr},
			expErr: true,
		},
		"builder missing": {
			args:   []string{"--code-source-url=" + correctSource, "--code-hash=" + checksumStr},
			expErr: true,
		},
		"code hash missing": {
			args:   []string{"--code-source-url=" + correctSource, "--builder=" + correctBuilderRef},
			expErr: true,
		},
		"source format wrong": {
			args:   []string{"--code-source-url=" + "format_wrong", "--builder=" + correctBuilderRef, "--code-hash=" + checksumStr},
			expErr: true,
		},
		"builder format wrong": {
			args:   []string{"--code-source-url=" + correctSource, "--builder=" + "format//", "--code-hash=" + checksumStr},
			expErr: true,
		},
		"code hash wrong": {
			args:   []string{"--code-source-url=" + correctSource, "--builder=" + correctBuilderRef, "--code-hash=" + "AA"},
			expErr: true,
		},
		"happy path, none set": {
			args:   []string{},
			expErr: false,
		},
		"happy path all set": {
			args:   []string{"--code-source-url=" + correctSource, "--builder=" + correctBuilderRef, "--code-hash=" + checksumStr},
			expErr: false,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			flags := ProposalStoreAndInstantiateContractCmd().Flags()
			require.NoError(t, flags.Parse(spec.args))
			_, _, _, gotErr := parseVerificationFlags(wasmBin, flags)
			if spec.expErr {
				require.Error(t, gotErr)
				return
			}
			require.NoError(t, gotErr)
		})
	}
}

func TestProposalInstantiateContract2Cmd(t *testing.T) {
	registry := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(registry)
	v1.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)
	txConfig := authtx.NewTxConfig(cdc, authtx.DefaultSignModes)

	proposer := sdk.AccAddress(bytes.Repeat([]byte{1}, 20))
	salt := []byte("my salt")
	out := &bytes.Buffer{}
	clientCtx := client.Context{}.
		WithCodec(cdc).
		WithInterfaceRegistry(registry).
		WithTxConfig(txConfig).
		WithKeyring(keyring.NewInMemory(cdc)).
		WithChainID("test-chain").
		WithOutput(out)

	cmd := ProposalInstantiateContract2Cmd()
	cmd.SetArgs([]string{
		"1",
		`{"foo":"bar"}`,
		hex.EncodeToString(salt),
		"--" + flagLabel, "testing",
		"--" + flagNoAdmin,
		"--" + flagFixMsg,
		"--title", "my title",
		"--summary", "my summary",
		"--" + flags.FlagFrom, proposer.String(),
		"--" + flags.FlagGenerateOnly,
	})
	ctx := context.WithValue(context.Background(), client.ClientContextKey, &clientCtx)
	require.NoError(t, cmd.ExecuteContext(ctx))

	gotTx, err := txConfig.TxJSONDecoder()(out.Bytes())
	require.NoError(t, err)
	require.Len(t, gotTx.GetMsgs(), 1)
	proposal, ok := gotTx.GetMsgs()[0].(*v1.MsgSubmitProposal)
	require.True(t, ok, "got %T", gotTx.GetMsgs()[0])
	propMsgs, err := proposal.GetMsgs()
	require.NoError(t, err)
	require.Len(t, propMsgs, 1)
	gotMsg, ok := propMsgs[0].(*types.MsgInstantiateContract2)
	require.True(t, ok, "got %T", propMsgs[0])
	assert.Equal(t, DefaultGovAuthority.String(), gotMsg.Sender)
	assert.Equal(t, uint64(1), gotMsg.CodeID)
	assert.Equal(t, "testing", gotMsg.Label)
	assert.Equal(t, types.RawContractMessage(`{"foo":"bar"}`), gotMsg.Msg)
	assert.Equal(t, salt, gotMsg.Salt)
	assert.True(t, gotMsg.FixMsg)
}
