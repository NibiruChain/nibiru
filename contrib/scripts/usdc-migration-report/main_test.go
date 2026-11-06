package main

import (
	"bytes"
	"encoding/json"
	"math/big"
	"strings"
	"testing"

	tmjson "github.com/NibiruChain/nibiru/v2/bft/libs/json"
	rpctypes "github.com/NibiruChain/nibiru/v2/bft/rpc/core/types"
	"github.com/NibiruChain/nibiru/v2/evm"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/bech32"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	abci "github.com/NibiruChain/nibiru/v2/bft/abci/types"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
)

func TestReadSavedRPCBlockResults(t *testing.T) {
	event := abci.Event(sdk.NewEvent("usdc_migration", sdk.NewAttribute("upgrade", "v2.23.0"),
		sdk.NewAttribute("height", "100"), sdk.NewAttribute("status", "skipped"),
		sdk.NewAttribute("reason", "registry_not_deployed")))
	for _, height := range []any{"100", 100} {
		data, err := json.Marshal(map[string]any{"result": map[string]any{"height": height, "begin_block_events": []abci.Event{event}}})
		require.NoError(t, err)
		var output bytes.Buffer
		require.NoError(t, run(bytes.NewReader(data), &output))
		require.Contains(t, output.String(), `"status": "skipped"`)
		require.Contains(t, output.String(), `"collected_base_units": "0"`)
	}
}

func TestRejectRPCFailureAndWrongBlock(t *testing.T) {
	for _, input := range []string{
		`{"error":{"message":"height not available"}}`,
		`{"result":{"height":"1","begin_block_events":[]}}`,
		`{"result":{"height":"2","begin_block_events":[{"type":"usdc_migration","attributes":[{"key":"upgrade","value":"v2.23.0"},{"key":"height","value":"1"},{"key":"status","value":"skipped"}]}]}}`,
	} {
		var output bytes.Buffer
		require.Error(t, run(strings.NewReader(input), &output))
		require.Empty(t, output.String())
	}
}

// Build actual nibi bank events without importing the app or configuring the
// process-wide SDK prefix. The standalone program starts with SDK defaults.
func TestReadCollectedNibiruBankHolderWithoutAppSetup(t *testing.T) {
	require.Equal(t, "cosmos", sdk.GetConfig().GetBech32AccountAddrPrefix())
	for _, addressLength := range []int{20, 32} {
		source, err := bech32.ConvertAndEncode("nibi", bytes.Repeat([]byte{3}, addressLength))
		require.NoError(t, err)
		token := gethcommon.HexToAddress("0x100")
		safe := gethcommon.HexToAddress("0x200")
		denom := "erc20/" + token.Hex()
		frame := func(status string) abci.Event {
			return abci.Event(sdk.NewEvent("usdc_migration",
				sdk.NewAttribute("upgrade", "v2.23.0"), sdk.NewAttribute("height", "100"),
				sdk.NewAttribute("status", status), sdk.NewAttribute("token", token.Hex()),
				sdk.NewAttribute("denom", denom), sdk.NewAttribute("safe", safe.Hex())))
		}
		conversion, err := sdk.TypedEventToEvent(&evm.EventConvertCoinToEvm{
			Sender: source, Erc20ContractAddress: token.Hex(), ToEthAddr: safe.Hex(),
			BankCoin: sdk.NewInt64Coin(denom, 1234567),
			EvmLogs: []evm.LogLite{{Address: token.Hex(), Topics: []string{
				crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)")).Hex(),
				gethcommon.BytesToHash(evm.EVM_MODULE_ADDRESS.Bytes()).Hex(),
				gethcommon.BytesToHash(safe.Bytes()).Hex(),
			}, Data: big.NewInt(1234567).FillBytes(make([]byte, 32))}},
		})
		require.NoError(t, err)
		account := abci.Event(sdk.NewEvent("usdc_migration_account",
			sdk.NewAttribute("upgrade", "v2.23.0"), sdk.NewAttribute("representation", "bank"),
			sdk.NewAttribute("source", source), sdk.NewAttribute("destination", safe.Hex()),
			sdk.NewAttribute("amount", "1234567"), sdk.NewAttribute("status", "collected"),
			sdk.NewAttribute("disposition", "repayment")))
		eoa := gethcommon.HexToAddress("0x300")
		transfer, err := sdk.TypedEventToEvent(&evm.EventTxLog{Logs: []evm.Log{{
			Address: token.Hex(), Topics: []string{
				crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)")).Hex(),
				gethcommon.BytesToHash(eoa.Bytes()).Hex(), gethcommon.BytesToHash(safe.Bytes()).Hex(),
			}, Data: big.NewInt(7).FillBytes(make([]byte, 32)),
		}}})
		require.NoError(t, err)
		eoaAccount := abci.Event(sdk.NewEvent("usdc_migration_account",
			sdk.NewAttribute("upgrade", "v2.23.0"), sdk.NewAttribute("representation", "erc20"),
			sdk.NewAttribute("source", eoa.Hex()), sdk.NewAttribute("destination", safe.Hex()),
			sdk.NewAttribute("amount", "7"), sdk.NewAttribute("status", "collected"),
			sdk.NewAttribute("disposition", "repayment")))
		result, err := tmjson.Marshal(rpctypes.ResultBlockResults{
			Height: 100, BeginBlockEvents: []abci.Event{frame("started"), abci.Event(conversion), account, abci.Event(transfer), eoaAccount, frame("completed")},
		})
		require.NoError(t, err)
		input, err := json.Marshal(map[string]json.RawMessage{"result": result})
		require.NoError(t, err)
		var output bytes.Buffer
		require.NoError(t, run(bytes.NewReader(input), &output))
		var report struct {
			Repayment string `json:"repayment_base_units"`
			Accounts  []struct {
				Source string `json:"source"`
				Amount string `json:"amount_base_units"`
			} `json:"accounts"`
		}
		require.NoError(t, json.Unmarshal(output.Bytes(), &report))
		require.Equal(t, "1234574", report.Repayment)
		require.Len(t, report.Accounts, 2)
		require.Equal(t, source, report.Accounts[0].Source)
		require.Equal(t, "1234567", report.Accounts[0].Amount)
		require.Equal(t, eoa.Hex(), report.Accounts[1].Source)
		// A valid address from another prefix must fail even when it matches
		// the SDK's process default. Never mutate that default to read events.
		wrongPrefix, err := bech32.ConvertAndEncode("cosmos", bytes.Repeat([]byte{3}, addressLength))
		require.NoError(t, err)
		output.Reset()
		require.ErrorContains(t, run(strings.NewReader(strings.ReplaceAll(string(input), source, wrongPrefix)), &output), "invalid bank holder")
		require.Empty(t, output.String())
		require.Equal(t, "cosmos", sdk.GetConfig().GetBech32AccountAddrPrefix())
	}
}
