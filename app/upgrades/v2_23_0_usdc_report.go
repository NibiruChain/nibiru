package upgrades

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	abci "github.com/NibiruChain/nibiru/v2/bft/abci/types"
	"github.com/NibiruChain/nibiru/v2/evm"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
)

// USDCMigrationAccount preserves the original representation and recipient.
// Amount is an exact integer in legacy token base units. Only collected entries
// create a repayment or custody-recovery allocation.
type USDCMigrationAccount struct {
	Representation string `json:"representation"`
	Source         string `json:"source"`
	Destination    string `json:"destination"`
	Amount         string `json:"amount_base_units"`
	Disposition    string `json:"disposition"`
	Status         string `json:"status"`
	Reason         string `json:"reason,omitempty"`
	Error          string `json:"error,omitempty"`
}

// USDCMigrationReport is reconstructed without keeper state or a pre-upgrade
// balance ledger. Context records must match real conversion/transfer events.
type USDCMigrationReport struct {
	Upgrade               string                 `json:"upgrade"`
	Height                int64                  `json:"height"`
	Status                string                 `json:"status"`
	Registry              string                 `json:"registry"`
	Token                 string                 `json:"token"`
	BankDenom             string                 `json:"bank_denom"`
	Safe                  string                 `json:"safe"`
	Reason                string                 `json:"reason,omitempty"`
	Error                 string                 `json:"error,omitempty"`
	Collected             string                 `json:"collected_base_units"`
	Repayment             string                 `json:"repayment_base_units"`
	CustodyRecovery       string                 `json:"custody_recovery_base_units"`
	BankSupplyRemaining   string                 `json:"bank_supply_remaining_base_units,omitempty"`
	ModuleEscrowRemaining string                 `json:"module_escrow_remaining_base_units,omitempty"`
	EscrowExcess          string                 `json:"escrow_excess_base_units,omitempty"`
	Accounts              []USDCMigrationAccount `json:"accounts"`
}

// ReconstructUSDCMigration consumes the upgrade block's ABCI events, including
// typed bank conversion events and EVM logs. It uses the migration's start/end
// markers to exclude unrelated transactions in that same block. Repeated zero
// transaction hashes/log indices are deliberately not used for deduplication.
// Missing/duplicate records, incomplete framing, or unmatched movements return
// no report rather than an incomplete repayment plan.
func ReconstructUSDCMigration(events []abci.Event) (*USDCMigrationReport, error) {
	var report *USDCMigrationReport
	active, closed := false, false
	evidence := make(map[string]int)
	seen := make(map[string]bool)
	collected, repayment, recovery := new(big.Int), new(big.Int), new(big.Int)
	for _, event := range events {
		if event.Type == "usdc_migration" {
			attrs, err := usdcReportAttributes(event)
			if err != nil {
				return nil, err
			}
			if attrs["upgrade"] != "v2.23.0" {
				continue
			}
			status := attrs["status"]
			if report == nil {
				height, err := strconv.ParseInt(attrs["height"], 10, 64)
				if err != nil || height < 1 {
					return nil, fmt.Errorf("USDC migration: invalid event height")
				}
				report = &USDCMigrationReport{
					Upgrade: attrs["upgrade"], Height: height, Status: status,
					Registry: attrs["registry"], Token: attrs["token"], BankDenom: attrs["denom"],
					Safe: attrs["safe"], Reason: attrs["reason"], Error: attrs["error"],
					Collected: "0", Repayment: "0", CustodyRecovery: "0", Accounts: []USDCMigrationAccount{},
				}
				switch status {
				case "started":
					if !gethcommon.IsHexAddress(report.Token) || !gethcommon.IsHexAddress(report.Safe) || report.BankDenom == "" {
						return nil, fmt.Errorf("USDC migration: invalid token, Safe or bank denom")
					}
					active = true
				case "skipped", "failed":
					closed = true
				default:
					return nil, fmt.Errorf("USDC migration: expected start/skip/failure event")
				}
				continue
			}
			if !active || closed || (status != "completed" && status != "failed") {
				return nil, fmt.Errorf("USDC migration: duplicate or unexpected summary event")
			}
			for key, want := range map[string]string{"registry": report.Registry, "token": report.Token, "denom": report.BankDenom, "safe": report.Safe, "height": strconv.FormatInt(report.Height, 10)} {
				if attrs[key] != want {
					return nil, fmt.Errorf("USDC migration: closing event changed %s", key)
				}
			}
			report.Status, report.Reason, report.Error = status, attrs["reason"], attrs["error"]
			for key, target := range map[string]*string{
				"bank_supply_remaining":   &report.BankSupplyRemaining,
				"module_escrow_remaining": &report.ModuleEscrowRemaining,
				"escrow_excess":           &report.EscrowExcess,
			} {
				if value, present := attrs[key]; present {
					amount, ok := new(big.Int).SetString(value, 10)
					if !ok || amount.Sign() < 0 || amount.BitLen() > 256 || amount.String() != value {
						return nil, fmt.Errorf("USDC migration: invalid %s audit amount", key)
					}
					*target = value
				}
			}
			active, closed = false, true
			continue
		}
		if !active {
			// Account records outside a migration frame must not be silently lost.
			if event.Type == "usdc_migration_account" {
				attrs, err := usdcReportAttributes(event)
				if err != nil {
					return nil, err
				}
				if attrs["upgrade"] == "v2.23.0" {
					return nil, fmt.Errorf("USDC migration: account outside migration frame")
				}
			}
			continue
		}
		switch event.Type {
		case "eth.evm.v1.EventConvertCoinToEvm":
			parsed, err := sdk.ParseTypedEvent(event)
			if err != nil {
				return nil, fmt.Errorf("decode bank conversion event: %w", err)
			}
			conversion, ok := parsed.(*evm.EventConvertCoinToEvm)
			if !ok {
				return nil, fmt.Errorf("USDC migration: unexpected conversion event type")
			}
			if !strings.EqualFold(conversion.Erc20ContractAddress, report.Token) || !strings.EqualFold(conversion.ToEthAddr, report.Safe) {
				continue
			}
			if !conversion.BankCoin.IsValid() || !conversion.BankCoin.IsPositive() || conversion.BankCoin.Denom != report.BankDenom {
				return nil, fmt.Errorf("USDC migration: invalid bank conversion coin")
			}
			matches := 0
			for _, log := range conversion.EvmLogs {
				from, amount, match, err := usdcReportTransfer(log, report.Token, report.Safe)
				if err != nil {
					return nil, err
				}
				if match {
					if from != evm.EVM_MODULE_ADDRESS || amount.Cmp(conversion.BankCoin.Amount.BigInt()) != 0 {
						return nil, fmt.Errorf("USDC migration: bank conversion and ERC20 backing transfer disagree")
					}
					matches++
				}
			}
			if matches != 1 {
				return nil, fmt.Errorf("USDC migration: bank conversion needs exactly one backing transfer")
			}
			evidence[usdcReportKey("bank", conversion.Sender, conversion.BankCoin.Amount.String())]++
		case evm.TypeUrlEventTxLog:
			logs, err := evm.EventTxLogFromABCIEvent(event)
			if err != nil {
				return nil, err
			}
			for _, log := range logs.Logs {
				from, amount, match, err := usdcReportTransfer(log.ToLogLite(), report.Token, report.Safe)
				if err != nil {
					return nil, err
				}
				if match {
					evidence[usdcReportKey("erc20", from.Hex(), amount.String())]++
				}
			}
		case "usdc_migration_account":
			attrs, err := usdcReportAttributes(event)
			if err != nil {
				return nil, err
			}
			if attrs["upgrade"] != report.Upgrade {
				return nil, fmt.Errorf("USDC migration: account upgrade mismatch")
			}
			account := USDCMigrationAccount{
				Representation: attrs["representation"], Source: attrs["source"], Destination: attrs["destination"],
				Amount: attrs["amount"], Disposition: attrs["disposition"], Status: attrs["status"],
				Reason: attrs["reason"], Error: attrs["error"],
			}
			amount, ok := new(big.Int).SetString(account.Amount, 10)
			if !ok || amount.Sign() < 0 || amount.BitLen() > 256 || amount.String() != account.Amount {
				return nil, fmt.Errorf("USDC migration: invalid account amount")
			}
			if !strings.EqualFold(account.Destination, report.Safe) {
				return nil, fmt.Errorf("USDC migration: account destination differs from Safe")
			}
			switch account.Representation {
			case "bank":
				if _, err := sdk.AccAddressFromBech32(account.Source); err != nil {
					return nil, fmt.Errorf("USDC migration: invalid bank holder: %w", err)
				}
			case "erc20":
				if !gethcommon.IsHexAddress(account.Source) {
					return nil, fmt.Errorf("USDC migration: invalid ERC20 holder")
				}
			default:
				return nil, fmt.Errorf("USDC migration: invalid representation")
			}
			key := usdcReportKey(account.Representation, account.Source, "")
			if seen[key] {
				return nil, fmt.Errorf("USDC migration: duplicate account record")
			}
			seen[key] = true
			switch account.Status {
			case "collected":
				if amount.Sign() == 0 || (account.Disposition != "repayment" && account.Disposition != "custody_recovery") {
					return nil, fmt.Errorf("USDC migration: invalid collection disposition or zero collection")
				}
				evidenceKey := usdcReportKey(account.Representation, account.Source, account.Amount)
				if evidence[evidenceKey] < 1 {
					return nil, fmt.Errorf("USDC migration: collected record has no matching movement for %s", account.Source)
				}
				evidence[evidenceKey]--
				collected.Add(collected, amount)
				if account.Disposition == "repayment" {
					repayment.Add(repayment, amount)
				} else {
					recovery.Add(recovery, amount)
				}
			case "zero":
				if amount.Sign() != 0 {
					return nil, fmt.Errorf("USDC migration: nonzero zero-balance record")
				}
			case "failed", "skipped":
			default:
				return nil, fmt.Errorf("USDC migration: invalid account status")
			}
			report.Accounts = append(report.Accounts, account)
		}
	}
	if report == nil || !closed || active {
		return nil, fmt.Errorf("USDC migration: missing or incomplete migration frame")
	}
	for _, count := range evidence {
		if count != 0 {
			return nil, fmt.Errorf("USDC migration: unmatched token movement")
		}
	}
	report.Collected, report.Repayment, report.CustodyRecovery = collected.String(), repayment.String(), recovery.String()
	return report, nil
}

func usdcReportAttributes(event abci.Event) (map[string]string, error) {
	attrs := make(map[string]string, len(event.Attributes))
	for _, attr := range event.Attributes {
		if _, exists := attrs[attr.Key]; exists {
			return nil, fmt.Errorf("USDC migration: duplicate event attribute %q", attr.Key)
		}
		attrs[attr.Key] = attr.Value
	}
	return attrs, nil
}

func usdcReportKey(representation, source, amount string) string {
	if representation == "erc20" {
		source = strings.ToLower(source)
	}
	return representation + "|" + source + "|" + amount
}

func usdcReportTransfer(log evm.LogLite, token, safe string) (gethcommon.Address, *big.Int, bool, error) {
	if !strings.EqualFold(log.Address, token) {
		return gethcommon.Address{}, nil, false, nil
	}
	signature := crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)")).Hex()
	if len(log.Topics) == 0 || !strings.EqualFold(log.Topics[0], signature) {
		return gethcommon.Address{}, nil, false, nil
	}
	if len(log.Topics) != 3 || len(log.Data) != 32 {
		return gethcommon.Address{}, nil, false, fmt.Errorf("USDC migration: malformed ERC20 Transfer log")
	}
	for _, topic := range log.Topics[1:] {
		if len(topic) != 66 || !strings.HasPrefix(topic, "0x") {
			return gethcommon.Address{}, nil, false, fmt.Errorf("USDC migration: malformed indexed address")
		}
		if _, ok := new(big.Int).SetString(topic[2:], 16); !ok || !strings.EqualFold(topic[2:26], strings.Repeat("0", 24)) {
			return gethcommon.Address{}, nil, false, fmt.Errorf("USDC migration: noncanonical indexed address")
		}
	}
	from := gethcommon.BytesToAddress(gethcommon.HexToHash(log.Topics[1]).Bytes())
	to := gethcommon.BytesToAddress(gethcommon.HexToHash(log.Topics[2]).Bytes())
	if to != gethcommon.HexToAddress(safe) {
		return gethcommon.Address{}, nil, false, nil
	}
	return from, new(big.Int).SetBytes(log.Data), true, nil
}
