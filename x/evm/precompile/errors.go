package precompile

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/core/vm"
)

// Error short-hand for type validation
func ErrArgTypeValidation(solidityHint string, arg any) error {
	return fmt.Errorf("type validation failed for (%s) argument: %s", solidityHint, arg)
}

// Error when parsing method arguments
func ErrInvalidArgs(err error) error {
	return fmt.Errorf("invalid method args: %w", err)
}

// Check required for transactions but not needed for queries
func assertNotReadonlyTx(readOnly bool, isTx bool) error {
	if readOnly && isTx {
		return errors.New("cannot write state from staticcall (a read-only call)")
	}
	return nil
}

func assertContractQuery(contract *vm.Contract) error {
	weiValue := contract.Value()
	if weiValue != nil && weiValue.Sign() != 0 {
		return fmt.Errorf(
			"funds (value) must not be expended calling a query function; received wei value %s", weiValue,
		)
	}

	return nil
}

func assertNumArgs(lenArgs, wantArgsLen int) error {
	if lenArgs != wantArgsLen {
		return fmt.Errorf("expected %d arguments but got %d", wantArgsLen, lenArgs)
	}
	return nil
}
