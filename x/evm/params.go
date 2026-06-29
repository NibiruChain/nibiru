// Copyright (c) 2023-2024 Nibi, Inc.
package evm

import (
	"fmt"

	sdkioerrors "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v7/modules/core/24-host"

	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"golang.org/x/exp/slices"

	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/eth"
)

const (
	// EVMBankDenom specifies the bank denomination for the asset used to run EVM
	// state transitions as the analog to "ether". 1 ether in solidity means 1
	// NIBI on Nibru EVM, implying that the EVMBankDenom is "unibi", the coin
	// base of the NIBI token.
	EVMBankDenom = appconst.DENOM_UNIBI

	// WasmPluginNameXOracle identifies the Wasm adapter backing the legacy
	// oracle precompile at 0x0000000000000000000000000000000000000801.
	WasmPluginNameXOracle = "x-oracle"
)

// DefaultParams returns default evm parameters
// ExtraEIPs is empty to prevent overriding the latest hard fork instruction set
func DefaultParams() Params {
	return Params{
		ExtraEIPs: []int64{},
		// EVMChannels: Unused but intended for use with future IBC functionality
		EVMChannels:       []string{},
		CreateFuntokenFee: sdkmath.NewIntWithDecimal(10_000, 6), // 10_000 NIBI
		CanonicalWnibi: eth.EIP55Addr{
			Address: gethcommon.HexToAddress("0x0CaCF669f8446BeCA826913a3c6B96aCD4b02a97"),
		},
		WasmPlugins: []WasmPlugin{},
	}
}

// validateChannels checks if channels ids are valid
func validateChannels(i any) error {
	channels, ok := i.([]string)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	for _, channel := range channels {
		if err := host.ChannelIdentifierValidator(channel); err != nil {
			return sdkioerrors.Wrap(
				channeltypes.ErrInvalidChannelIdentifier, err.Error(),
			)
		}
	}

	return nil
}

// Validate performs basic validation on evm parameters.
func (p Params) Validate() error {
	if err := validateEIPs(p.ExtraEIPs); err != nil {
		return fmt.Errorf("ParamsError: %w", err)
	}

	if err := validateChannels(p.EVMChannels); err != nil {
		return fmt.Errorf("ParamsError: %w", err)
	}

	if _, err := eth.NewEIP55AddrFromStr(p.CanonicalWnibi.Hex()); err != nil {
		return fmt.Errorf("ParamsError: %w", err)
	} else if (p.CanonicalWnibi.Address == gethcommon.Address{}) {
		err = fmt.Errorf("ParamsError: evm.Params.CanonicalWnibi cannot be the zero address")
		return err
	}

	if err := ValidateWasmPlugins(p.WasmPlugins); err != nil {
		return fmt.Errorf("ParamsError: %w", err)
	}

	return nil
}

// EIPs returns the ExtraEIPS as a int slice
func (p Params) EIPs() []int {
	eips := make([]int, len(p.ExtraEIPs))
	for i, eip := range p.ExtraEIPs {
		eips[i] = int(eip)
	}
	return eips
}

// IsEVMChannel returns true if the channel provided is in the list of
// EVM channels
func (p Params) IsEVMChannel(channel string) bool {
	return slices.Contains(p.EVMChannels, channel)
}

func validateEIPs(i any) error {
	eips, ok := i.([]int64)
	if !ok {
		return fmt.Errorf("invalid EIP slice type: %T", i)
	}

	uniqueEIPs := make(map[int64]struct{})

	for _, eip := range eips {
		if !vm.ValidEip(int(eip)) {
			return fmt.Errorf("EIP %d is not activateable, valid EIPs are: %s", eip, vm.ActivateableEips())
		}

		if _, ok := uniqueEIPs[eip]; ok {
			return fmt.Errorf("found duplicate EIP: %d", eip)
		}
		uniqueEIPs[eip] = struct{}{}
	}

	return nil
}

// ValidateWasmPlugins checks the named Wasm plugin config used by EVM
// execution paths. Plugin order is preserved by callers, so the returned names
// from WasmPluginAddrsByName should be used for deterministic writes.
func ValidateWasmPlugins(plugins []WasmPlugin) error {
	_, _, err := WasmPluginAddrsByName(plugins)
	return err
}

// WasmPluginAddrsByName validates plugins and returns decoded addresses keyed by
// name, plus the original name order for deterministic state writes.
func WasmPluginAddrsByName(
	plugins []WasmPlugin,
) (map[string]sdk.AccAddress, []string, error) {
	pluginAddrByName := make(map[string]sdk.AccAddress, len(plugins))
	pluginNames := make([]string, 0, len(plugins))
	for _, plugin := range plugins {
		if plugin.Name == "" {
			return nil, nil, fmt.Errorf("wasm plugin name cannot be empty")
		}
		if _, exists := pluginAddrByName[plugin.Name]; exists {
			return nil, nil, fmt.Errorf("duplicate wasm plugin name: %s", plugin.Name)
		}
		addr, err := sdk.AccAddressFromBech32(plugin.Addr)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid wasm plugin address for %s: %w", plugin.Name, err)
		}
		pluginAddrByName[plugin.Name] = addr
		pluginNames = append(pluginNames, plugin.Name)
	}
	return pluginAddrByName, pluginNames, nil
}
