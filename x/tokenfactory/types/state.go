package types

import (
	"fmt"
	"strings"

	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

// ----------------------------------------------------
// ModuleParams functions
// ----------------------------------------------------

func DefaultModuleParams() ModuleParams {
	return ModuleParams{
		DenomCreationGasConsume: 4_000_000,
	}
}

func (params ModuleParams) Validate() error {
	if params.DenomCreationGasConsume < 1 {
		return ErrInvalidModuleParams.Wrap("cannot set gas creation cost to zero")
	}
	return nil
}

// ----------------------------------------------------
// TFDenom functions
// ----------------------------------------------------

func (tfd TFDenom) Denom() DenomStr {
	return DenomStr(tfd.String())
}

// String: returns the standard string representation.
func (tfd TFDenom) String() string {
	return fmt.Sprintf("tf/%s/%s", tfd.Creator, tfd.Subdenom)
}

func (tfd TFDenom) Validate() error {
	return tfd.Denom().Validate()
}

func (tfd TFDenom) DefaultBankMetadata() banktypes.Metadata {
	denom := tfd.String()
	return banktypes.Metadata{
		DenomUnits: []*banktypes.DenomUnit{{
			Denom:    denom,
			Exponent: 0,
		}},
		Base: denom,
		// The following is necessary for x/bank denom validation
		Display: denom,
		Name:    denom,
		Symbol:  denom,
	}
}

// ----------------------------------------------------
// DenomStr functions
// ----------------------------------------------------

// DenomStr: string identifier for a token factory denom (TFDenom)
type DenomStr string

func DenomFormatError(got string, msg ...string) error {
	errStr := fmt.Sprintf(`denom format error: expected "tf/{creator-bech32}/{subdenom}", got %v`, got)
	if len(msg) > 0 {
		errStr += fmt.Sprintf(": %v", msg)
	}
	return fmt.Errorf(errStr)
}

func (denomStr DenomStr) Validate() error {
	_, err := denomStr.ToStruct()
	return err
}

func (denomStr DenomStr) String() string { return string(denomStr) }

func (genDenom GenesisDenom) Validate() error {
	return DenomStr(genDenom.Denom).Validate()
}

func (denomStr DenomStr) ToStruct() (res TFDenom, err error) {
	str := string(denomStr)
	parts := strings.Split(str, "/")
	switch {
	case len(parts) != 3:
		return res, DenomFormatError("denom has invalid number of sections separated by '/'")
	case parts[0] != "tf":
		return res, DenomFormatError(str, `missing denom prefix "tf"`)
	case len(parts[1]) < 1:
		return res, DenomFormatError(str, "empty creator address")
	case len(parts[2]) < 1:
		return res, DenomFormatError(str, "empty subdenom")
	}

	return TFDenom{
		Creator:  parts[1],
		Subdenom: parts[2],
	}, nil
}

func (denomStr DenomStr) MustToStruct() TFDenom {
	out, _ := denomStr.ToStruct()
	return out
}
