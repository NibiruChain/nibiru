package precompile

import (
	"fmt"
	"reflect"

	gethabi "github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core/vm"
)

func assertNotReadonlyTx(readOnly bool, method *gethabi.Method) error {
	if readOnly {
		return fmt.Errorf("method %s cannot be called in a read-only context (e.g. staticcall)", method.Name)
	}
	return nil
}

// ErrPrecompileRun is error function intended for use in a `defer` pattern,
// which modifies the input error in the event that its value becomes non-nil.
// This creates a concise way to prepend extra information to the original error.
func ErrPrecompileRun(err error, p vm.PrecompiledContract) error {
	if err != nil {
		precompileType := reflect.TypeOf(p).Name()
		err = fmt.Errorf("precompile error: failed to run %s: %w", precompileType, err)
	}
	return err
}

// Error short-hand for type validation
func ErrArgTypeValidation(solidityHint string, arg any) error {
	return fmt.Errorf("type validation failed for (%s) argument: %s", solidityHint, arg)
}

// Error when parsing method arguments
func ErrInvalidArgs(err error) error {
	return fmt.Errorf("invalid method args: %w", err)
}

func ErrMethodCalled(method *gethabi.Method, wrapped error) error {
	return fmt.Errorf("%s method called: %w", method.Name, wrapped)
}

// assertContractQuery checks if a contract call is a valid query operation. This
// function verifies that no funds (wei) are being sent with the query, as query
// operations should be read-only and not involve any value transfer.
func assertContractQuery(contract *vm.Contract) error {
	weiValue := contract.Value()
	if weiValue != nil && weiValue.Sign() != 0 {
		return fmt.Errorf(
			"funds (wei value) must not be expended calling a query function; received wei value %s", weiValue,
		)
	}

	return nil
}

// assertNumArgs checks if the number of provided arguments matches the expected
// count. If lenArgs does not equal wantArgsLen, it returns an error describing
// the mismatch between expected and actual argument counts.
func assertNumArgs(args []any, wantArgsLen int) error {
	lenArgs := len(args)
	if lenArgs != wantArgsLen {
		return fmt.Errorf("expected %d arguments but got %d; args: %v", wantArgsLen, lenArgs, args)
	}
	return nil
}
