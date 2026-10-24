package types

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	sdkerrors "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/errors"
)

// TestAddressListValidation guards against permission entry points applying
// different validation rules. Every path must enforce the same count boundaries
// and reject case-variant duplicates without accepting malformed mixed-case
// Bech32 input. Valid uppercase addresses remain accepted.
func TestAddressListValidation(t *testing.T) {
	addresses := make([]string, MaxAddressCount+1)
	for i := range addresses {
		addresses[i] = sdk.AccAddress(bytes.Repeat([]byte{byte(i)}, 20)).String()
	}
	authority := addresses[0]
	validators := map[string]func([]string) error{
		"access config": func(addrs []string) error {
			return (AccessConfig{Permission: AccessTypeAnyOfAddresses, Addresses: addrs}).ValidateBasic()
		},
		"store code": func(addrs []string) error {
			return (MsgStoreCode{
				Sender: authority, WASMByteCode: []byte("wasm"),
				InstantiatePermission: &AccessConfig{Permission: AccessTypeAnyOfAddresses, Addresses: addrs},
			}).ValidateBasic()
		},
		"update instantiate config": func(addrs []string) error {
			return (MsgUpdateInstantiateConfig{
				Sender: authority, CodeID: 1,
				NewInstantiatePermission: &AccessConfig{Permission: AccessTypeAnyOfAddresses, Addresses: addrs},
			}).ValidateBasic()
		},
		"add upload addresses": func(addrs []string) error {
			return (MsgAddCodeUploadParamsAddresses{Authority: authority, Addresses: addrs}).ValidateBasic()
		},
		"remove upload addresses": func(addrs []string) error {
			return (MsgRemoveCodeUploadParamsAddresses{Authority: authority, Addresses: addrs}).ValidateBasic()
		},
	}
	cases := map[string]struct {
		addresses   []string
		err         error
		errContains string
	}{
		"at limit":               {addresses: addresses[:MaxAddressCount]},
		"over limit":             {addresses: addresses, err: sdkerrors.ErrInvalidRequest},
		"empty":                  {err: ErrEmpty},
		"uppercase":              {addresses: []string{strings.ToUpper(authority)}},
		"duplicate":              {addresses: []string{authority, authority}, err: ErrDuplicate},
		"case variant duplicate": {addresses: []string{authority, strings.ToUpper(authority)}, err: ErrDuplicate},
		"invalid address":        {addresses: []string{"invalid"}, errContains: "decoding bech32 failed"},
		"mixed case address":     {addresses: []string{strings.ToUpper(authority[:1]) + authority[1:]}, errContains: "decoding bech32 failed"},
	}
	for name, validate := range validators {
		t.Run(name, func(t *testing.T) {
			for name, tc := range cases {
				t.Run(name, func(t *testing.T) {
					err := validate(tc.addresses)
					if tc.errContains != "" {
						require.ErrorContains(t, err, tc.errContains)
					} else if tc.err == nil {
						require.NoError(t, err)
					} else {
						require.ErrorIs(t, err, tc.err)
					}
				})
			}
		})
	}
}

// TestAccessTypeWithAddressLimit checks that the programmatic constructor uses
// the same count limit. AccessType.With reports invalid configurations by panic,
// unlike ValidateBasic methods, which return errors for transaction input.
func TestAccessTypeWithAddressLimit(t *testing.T) {
	addresses := make([]sdk.AccAddress, MaxAddressCount+1)
	for i := range addresses {
		addresses[i] = sdk.AccAddress(bytes.Repeat([]byte{byte(i)}, 20))
	}
	require.NotPanics(t, func() { AccessTypeAnyOfAddresses.With(addresses[:MaxAddressCount]...) })
	require.Panics(t, func() { AccessTypeAnyOfAddresses.With(addresses...) })
	require.Panics(t, func() { AccessTypeAnyOfAddresses.With() })
	require.Panics(t, func() { AccessTypeAnyOfAddresses.With(addresses[0], addresses[0]) })
}
