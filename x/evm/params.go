// Copyright (c) 2023-2024 Nibi, Inc.
package evm

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v7/modules/core/24-host"

	"github.com/ethereum/go-ethereum/core/vm"
	"golang.org/x/exp/slices"

	"github.com/NibiruChain/nibiru/v2/app/appconst"
)

const (
	// DefaultEVMDenom defines the default EVM denomination
	DefaultEVMDenom = appconst.BondDenom
)

// DefaultParams returns default evm parameters
// ExtraEIPs is empty to prevent overriding the latest hard fork instruction set
func DefaultParams() Params {
	return Params{
		EvmDenom:            DefaultEVMDenom,
		ExtraEIPs:           []int64{},
		AllowUnprotectedTxs: false,
		EVMChannels:         []string{},
		CreateFuntokenFee:   math.NewIntWithDecimal(10_000, 6), // 10_000 NIBI
	}
}

// validateChannels checks if channels ids are valid
func validateChannels(i interface{}) error {
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
	if err := validateEVMDenom(p.EvmDenom); err != nil {
		return err
	}

	if err := validateEIPs(p.ExtraEIPs); err != nil {
		return err
	}

	if err := validateBool(p.AllowUnprotectedTxs); err != nil {
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

func validateEVMDenom(i interface{}) error {
	denom, ok := i.(string)
	if !ok {
		return fmt.Errorf("invalid parameter EVM denom type: %T", i)
	}

	return sdk.ValidateDenom(denom)
}

func validateBool(i interface{}) error {
	_, ok := i.(bool)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	return nil
}

func validateEIPs(i interface{}) error {
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
