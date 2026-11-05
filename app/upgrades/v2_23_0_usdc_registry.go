package upgrades

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	gethcommon "github.com/ethereum/go-ethereum/common"

	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	wasmtypes "github.com/NibiruChain/nibiru/v2/x/wasm/types"
)

const usdcHolderPageSize = 200

// USDCHolderSetKeeper is the read-only Wasm surface used by the migration.
// The holder set must already be deployed before the upgrade runs.
type USDCHolderSetKeeper interface {
	GetContractInfo(sdk.Context, sdk.AccAddress) *wasmtypes.ContractInfo
	GetCodeInfo(sdk.Context, uint64) *wasmtypes.CodeInfo
	QuerySmart(sdk.Context, sdk.AccAddress, []byte) ([]byte, error)
}

// ReadUSDCHolderSet reads candidates, not balances or transfer permissions.
// It checks the deployed bytecode and validates every page before returning any
// candidates. Callers must not start moving funds until this succeeds.
// The gzipped contract fixture is test-only; this function never deploys code.
func ReadUSDCHolderSet(
	ctx sdk.Context, keeper USDCHolderSetKeeper, contract sdk.AccAddress, checksum []byte,
) ([]gethcommon.Address, error) {
	if err := sdk.VerifyAddressFormat(contract); err != nil {
		return nil, fmt.Errorf("USDC holder set: invalid contract address: %w", err)
	}
	if len(checksum) != 32 {
		return nil, fmt.Errorf("USDC holder set: expected a 32-byte checksum")
	}
	info := keeper.GetContractInfo(ctx, contract)
	if info == nil {
		return nil, fmt.Errorf("USDC holder set %s is not deployed", contract)
	}
	code := keeper.GetCodeInfo(ctx, info.CodeID)
	if code == nil || !bytes.Equal(code.CodeHash, checksum) {
		return nil, fmt.Errorf("USDC holder set %s has unexpected bytecode", contract)
	}

	var candidates []gethcommon.Address
	var cursor *string
	previous := ""
	for {
		request := struct {
			Addresses struct {
				StartAfter *string `json:"start_after"`
				Limit      uint32  `json:"limit"`
			} `json:"addresses"`
		}{}
		request.Addresses.StartAfter = cursor
		request.Addresses.Limit = usdcHolderPageSize
		msg, err := json.Marshal(request)
		if err != nil {
			return nil, err
		}
		bz, err := keeper.QuerySmart(ctx, contract, msg)
		if err != nil {
			return nil, fmt.Errorf("query USDC holder set: %w", err)
		}
		var page struct {
			Addresses      []string `json:"addresses"`
			NextStartAfter *string  `json:"next_start_after"`
		}
		if err := json.Unmarshal(bz, &page); err != nil {
			return nil, fmt.Errorf("decode USDC holder set: %w", err)
		}
		if len(page.Addresses) > usdcHolderPageSize {
			return nil, fmt.Errorf("USDC holder set: oversized page")
		}
		for _, address := range page.Addresses {
			if !gethcommon.IsHexAddress(address) || len(address) != 42 ||
				!strings.HasPrefix(address, "0x") || address != strings.ToLower(address) {
				return nil, fmt.Errorf("USDC holder set: noncanonical address %q", address)
			}
			if address <= previous {
				return nil, fmt.Errorf("USDC holder set: addresses are not strictly increasing")
			}
			previous = address
			candidates = append(candidates, gethcommon.HexToAddress(address))
		}
		if page.NextStartAfter == nil {
			return candidates, nil
		}
		if len(page.Addresses) == 0 || *page.NextStartAfter != previous {
			return nil, fmt.Errorf("USDC holder set: cursor does not match the last address")
		}
		cursor = page.NextStartAfter
	}
}
