// Copyright (c) 2023-2024 Nibi, Inc.
package evm

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v7/modules/core/24-host"

	"github.com/ethereum/go-ethereum/core/vm"
	"golang.org/x/exp/slices"

	"github.com/NibiruChain/nibiru/v2/app/appconst"
)

const (
	// EVMBankDenom specifies the bank denomination for the asset used to run EVM
	// state transitions as the analog to "ether". 1 ether in solidity means 1
	// NIBI on Nibru EVM, implying that the EVMBankDenom is "unibi", the coin
	// base of the NIBI token.
	EVMBankDenom = appconst.BondDenom
)

// DefaultParams returns default evm parameters
// ExtraEIPs is empty to prevent overriding the latest hard fork instruction set
func DefaultParams() Params {
	return Params{
		ExtraEIPs: []int64{},
		// EVMChannels: Unused but intended for use with future IBC functionality
		EVMChannels:       []string{},
		CreateFuntokenFee: math.NewIntWithDecimal(10_000, 6), // 10_000 NIBI
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
			return errorsmod.Wrap(
				channeltypes.ErrInvalidChannelIdentifier, err.Error(),
			)
		}
	}

	return nil
}

// Validate performs basic validation on evm parameters.
func (p Params) Validate() error {
	if err := validateEIPs(p.ExtraEIPs); err != nil {
		return err
	}

	return validateChannels(p.EVMChannels)
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
