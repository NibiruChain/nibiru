package upgrades_test

import (
	"bytes"
	"encoding/json"
	"math/big"
	"testing"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/app/upgrades"
	abci "github.com/NibiruChain/nibiru/v2/bft/abci/types"
	"github.com/NibiruChain/nibiru/v2/evm"
	"github.com/NibiruChain/nibiru/v2/evm/evmtest"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
)

func reportEvent(kind string, attrs ...string) abci.Event {
	attributes := []sdk.Attribute{sdk.NewAttribute("upgrade", "v2.23.0")}
	for i := 0; i < len(attrs); i += 2 {
		attributes = append(attributes, sdk.NewAttribute(attrs[i], attrs[i+1]))
	}
	return abci.Event(sdk.NewEvent(kind, attributes...))
}

func reportTyped(t *testing.T, event interface {
	ProtoMessage()
	Reset()
	String() string
},
) abci.Event {
	t.Helper()
	got, err := sdk.TypedEventToEvent(event)
	require.NoError(t, err)
	return abci.Event(got)
}

func reportFixture(t *testing.T) []abci.Event {
	t.Helper()
	token, safe := gethcommon.HexToAddress("0x100"), gethcommon.HexToAddress("0x200")
	bankUser := sdk.AccAddress(bytes.Repeat([]byte{3}, 20)).String()
	bankCore := sdk.AccAddress(bytes.Repeat([]byte{4}, 32)).String()
	eoa, sai := gethcommon.HexToAddress("0x300"), gethcommon.HexToAddress("0x400")
	denom := "erc20/" + token.Hex()
	frame := func(status string) abci.Event {
		return reportEvent("usdc_migration", "status", status, "height", "49130384", "registry", "registry", "token", token.Hex(), "denom", denom, "safe", safe.Hex())
	}
	conversion := func(sender string, amount int64) abci.Event {
		return reportTyped(t, &evm.EventConvertCoinToEvm{
			Sender: sender, Erc20ContractAddress: token.Hex(), ToEthAddr: safe.Hex(),
			BankCoin: sdk.NewInt64Coin(denom, amount),
			EvmLogs:  []evm.LogLite{evmtest.LogLiteEventErc20Transfer(token, evm.EVM_MODULE_ADDRESS, safe, big.NewInt(amount))},
		})
	}
	transfer := func(sender gethcommon.Address, amount int64) abci.Event {
		lite := evmtest.LogLiteEventErc20Transfer(token, sender, safe, big.NewInt(amount))
		// No signed tx exists for upgrade calls. Different movements can share
		// zero tx hashes and log indices, so those are not deduplication keys.
		return reportTyped(t, &evm.EventTxLog{Logs: []evm.Log{{Address: lite.Address, Topics: lite.Topics, Data: lite.Data}}})
	}
	account := func(representation, source, amount, disposition, status string) abci.Event {
		return reportEvent("usdc_migration_account", "representation", representation, "source", source,
			"destination", safe.Hex(), "amount", amount, "disposition", disposition, "status", status)
	}
	return []abci.Event{
		transfer(eoa, 999), // Other movement in the same block, outside the frame.
		frame("started"),
		conversion(bankUser, 100), account("bank", bankUser, "100", "repayment", "collected"),
		conversion(bankCore, 200), account("bank", bankCore, "200", "custody_recovery", "collected"),
		transfer(eoa, 300), account("erc20", eoa.Hex(), "300", "repayment", "collected"),
		transfer(sai, 400), account("erc20", sai.Hex(), "400", "custody_recovery", "collected"),
		account("erc20", gethcommon.HexToAddress("0x500").Hex(), "55", "repayment", "skipped"),
		account("erc20", gethcommon.HexToAddress("0x600").Hex(), "22", "repayment", "failed"),
		account("erc20", gethcommon.HexToAddress("0x700").Hex(), "0", "repayment", "zero"),
		frame("completed"),
		transfer(eoa, 888),
	}
}

func TestUSDCMigrationReportFromBlockEvents(t *testing.T) {
	// JSON round-trip the block events. The reader receives neither keepers,
	// the original balances, nor a separate migration ledger/config object.
	bz, err := json.Marshal(reportFixture(t))
	require.NoError(t, err)
	var events []abci.Event
	require.NoError(t, json.Unmarshal(bz, &events))
	report, err := upgrades.ReconstructUSDCMigration(events)
	require.NoError(t, err)
	require.Equal(t, int64(49130384), report.Height)
	require.Equal(t, "completed", report.Status)
	require.Equal(t, "1000", report.Collected)
	require.Equal(t, "400", report.Repayment)
	require.Equal(t, "600", report.CustodyRecovery)
	require.Len(t, report.Accounts, 7)
	require.Equal(t, sdk.AccAddress(bytes.Repeat([]byte{4}, 32)).String(), report.Accounts[1].Source)
	readable, err := json.MarshalIndent(report, "", "  ")
	require.NoError(t, err)
	t.Log(string(readable))
}

func TestUSDCMigrationReportRejectsIncompleteOrDuplicateAccounting(t *testing.T) {
	for name, change := range map[string]func([]abci.Event) []abci.Event{
		"no closing frame": func(events []abci.Event) []abci.Event { return events[:13] },
		"collected record without movement": func(events []abci.Event) []abci.Event {
			return append(events[:2:2], events[3:]...)
		},
		"movement without account": func(events []abci.Event) []abci.Event {
			return append(events[:3:3], events[4:]...)
		},
		"duplicate collected record": func(events []abci.Event) []abci.Event {
			return append(events[:4:4], append([]abci.Event{events[3]}, events[4:]...)...)
		},
		"duplicate backing movement": func(events []abci.Event) []abci.Event {
			return append(events[:3:3], append([]abci.Event{events[2]}, events[3:]...)...)
		},
		"account after closing frame": func(events []abci.Event) []abci.Event {
			return append(events, events[3])
		},
	} {
		t.Run(name, func(t *testing.T) {
			report, err := upgrades.ReconstructUSDCMigration(change(reportFixture(t)))
			require.Error(t, err)
			require.Nil(t, report)
		})
	}
}

func TestUSDCMigrationReportSkippedOrFailedPreflight(t *testing.T) {
	for _, status := range []string{"skipped", "failed"} {
		t.Run(status, func(t *testing.T) {
			event := reportEvent("usdc_migration", "status", status, "height", "5", "reason", "preflight")
			report, err := upgrades.ReconstructUSDCMigration([]abci.Event{event})
			require.NoError(t, err)
			require.Equal(t, status, report.Status)
			require.Equal(t, "0", report.Collected)
			require.Empty(t, report.Accounts)
		})
	}
}
