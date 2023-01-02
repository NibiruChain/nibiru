package common

import (
	"errors"
	"fmt"
)

// ToError converts a value to error if if (1) is a string, (2) has a String() function
// or (3) is already an error.
func ToError(v any) error {
	if v == nil {
		return nil
	}
	switch v := v.(type) {
	case string:
		return errors.New(v)
	case error:
		return v
	case fmt.Stringer:
		return errors.New(v.String())
	default:
		panic(fmt.Errorf("invalid type: %T", v))
	}
}

// Combines errors into single error. Error descriptions are ordered the same way
// they're passed to the function.
func CombineErrors(errs ...error) error {
	var err error
	for _, e := range errs {
		switch {
		case e != nil && err == nil:
			err = e
		case e != nil && err != nil:
			err = fmt.Errorf("%s: %s", err, e)
		}
	}
	return err
}

func CombineErrorsFromStrings(strs ...string) error {
	var errs []error
	for _, s := range strs {
		errs = append(errs, ToError(s))
	}
	return CombineErrors(errs...)
}
