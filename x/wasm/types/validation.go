package types

import (
	"fmt"
	"net/url"
	"strings"
	"unicode"

	"github.com/distribution/reference"

	sdkioerrors "cosmossdk.io/errors"

	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	sdkerrors "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types/errors"
)

// MaxSaltSize is the longest salt that can be used when instantiating a contract
const MaxSaltSize = 64

var (
	// MaxLabelSize is the longest label that can be used when instantiating a contract
	MaxLabelSize = 128 // extension point for chains to customize via compile flag.

	// MaxWasmSize is the largest a compiled contract code can be when storing code on chain
	MaxWasmSize = 800 * 1024 // extension point for chains to customize via compile flag.

	// MaxProposalWasmSize is the largest a gov proposal compiled contract code can be when storing code on chain
	MaxProposalWasmSize = 3 * 1024 * 1024 // extension point for chains to customize via compile flag.

	// MaxAddressCount is the maximum number of addresses allowed within a message.
	MaxAddressCount = 50
)

func validateWasmCode(s []byte, maxSize int) error {
	if len(s) == 0 {
		return sdkioerrors.Wrap(ErrEmpty, "is required")
	}
	if len(s) > maxSize {
		return sdkioerrors.Wrapf(ErrLimit, "cannot be longer than %d bytes", maxSize)
	}
	return nil
}

// ValidateLabel ensure label constraints
func ValidateLabel(label string) error {
	if label == "" {
		return sdkioerrors.Wrap(ErrEmpty, "is required")
	}
	if len(label) > MaxLabelSize {
		return ErrLimit.Wrapf("cannot be longer than %d characters", MaxLabelSize)
	}
	if label != strings.TrimSpace(label) {
		return ErrInvalid.Wrap("label must not start/end with whitespaces")
	}
	labelWithPrintableCharsOnly := strings.Map(func(r rune) rune {
		if unicode.IsPrint(r) {
			return r
		}
		return -1
	}, label)
	if label != labelWithPrintableCharsOnly {
		return ErrInvalid.Wrap("label must have printable characters only")
	}
	return nil
}

// ValidateSalt ensure salt constraints
func ValidateSalt(salt []byte) error {
	switch n := len(salt); {
	case n == 0:
		return sdkioerrors.Wrap(ErrEmpty, "is required")
	case n > MaxSaltSize:
		return ErrLimit.Wrapf("cannot be longer than %d characters", MaxSaltSize)
	}
	return nil
}

// ValidateVerificationInfo ensure source, builder and checksum constraints
func ValidateVerificationInfo(source, builder string, codeHash []byte) error {
	// if any set require others to be set
	if len(source) != 0 || len(builder) != 0 || len(codeHash) != 0 {
		if source == "" {
			return fmt.Errorf("source is required")
		}
		if _, err := url.ParseRequestURI(source); err != nil {
			return fmt.Errorf("source: %s", err)
		}
		if builder == "" {
			return fmt.Errorf("builder is required")
		}
		if _, err := reference.ParseDockerRef(builder); err != nil {
			return fmt.Errorf("builder: %s", err)
		}
		if codeHash == nil {
			return fmt.Errorf("code hash is required")
		}
		// code hash checksum match validation is done in the keeper, ungzipping consumes gas
	}
	return nil
}

// validateBech32Addresses ensures the list is not empty, has no duplicates,
// and does not exceed the maximum number of addresses.
func validateBech32Addresses(addresses []string) error {
	switch n := len(addresses); {
	case n == 0:
		return sdkioerrors.Wrap(ErrEmpty, "addresses")
	case n > MaxAddressCount:
		return sdkioerrors.Wrapf(sdkerrors.ErrInvalidRequest, "total number of addresses is greater than %d", MaxAddressCount)
	}

	index := make(map[string]struct{}, len(addresses))
	for _, addr := range addresses {
		if _, err := sdk.AccAddressFromBech32(addr); err != nil {
			return sdkioerrors.Wrapf(err, "address: %s", addr)
		}
		// Normalize valid Bech32 addresses to detect case variants of the same address.
		addr = strings.ToUpper(addr)
		if _, found := index[addr]; found {
			return sdkioerrors.Wrap(ErrDuplicate, "duplicate addresses")
		}
		index[addr] = struct{}{}
	}
	return nil
}
