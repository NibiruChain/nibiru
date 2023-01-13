package common

import (
	"errors"
	"fmt"
	"runtime/debug"
)

// TryCatch is an implementation of the try-catch block from languages like C++ and JS.
// Given a 'callback' function, TryCatch defers and recovers from any panics or
// errors, allowing one to safely handle multiple panics in succession.
//
// Typically, you'll write something like: `err := TryCatch(aRiskyFunction)()`
//
// Usage example:
//
//	var calmPanic error = TryCatch(func() {
//	  panic("something crazy")
//	})()
//	fmt.Println(calmPanic.Error()) // prints "something crazy"
//
// Note that TryCatch doesn't return an error. It returns a function that returns
// an error. Only when you call the output of TryCatch will it "feel" like a
// try-catch block from other languages.
//
// This means that TryCatch can be used to restart go routines that panic as well.
func TryCatch(callback func()) func() error {
	return func() (err error) {
		defer func() {
			if panicInfo := recover(); panicInfo != nil {
				err = fmt.Errorf("%v, %s", panicInfo, string(debug.Stack()))
				return
			}
		}()
		callback() // calling the decorated function
		return err
	}
}

// ToError converts a value to an error type if it:
// (1) is a string,
// (2) has a String() function
// (3) is already an error.
// (4) or is a slice of these cases
// I.e., the types supported are:
// string, []string, error, []error, fmt.Stringer, []fmt.Stringer
//
// The type is inferred from try catch blocks at runtime.
func ToError(v any) (out error, ok bool) {
	if v == nil {
		return nil, true
	}
	switch v := v.(type) {
	case string:
		return errors.New(v), true
	case error:
		return v, true
	case fmt.Stringer:
		return errors.New(v.String()), true
	case []string:
		return toErrorFromSlice(v)
	case []error:
		return toErrorFromSlice(v)
	case []fmt.Stringer:
		return toErrorFromSlice(v)
	default:
		// Cases for duck typing at runtime

		// case: error
		if tcErr := TryCatch(func() {
			v := v.(error)
			out = errors.New(v.Error())
		})(); tcErr == nil {
			return out, true
		}

		// case: string
		if tcErr := TryCatch(func() {
			v := v.(string)
			out = errors.New(v)
		})(); tcErr == nil {
			return out, true
		}

		// case: fmt.Stringer (object with String method)
		if tcErr := TryCatch(func() {
			v := v.(fmt.Stringer)
			out = errors.New(v.String())
		})(); tcErr == nil {
			return out, true
		}

		// case: []string
		if tcErr := TryCatch(func() {
			if maybeOut, okLocal := ToError(v.([]string)); okLocal {
				out = maybeOut
			}
		})(); tcErr == nil {
			return out, true
		}

		// case: []error
		if tcErr := TryCatch(func() {
			if maybeOut, okLocal := ToError(v.([]error)); okLocal {
				out = maybeOut
			}
		})(); tcErr == nil {
			return out, true
		}

		// case: []fmt.Stringer
		if tcErr := TryCatch(func() {
			if maybeOut, okLocal := ToError(v.([]fmt.Stringer)); okLocal {
				out = maybeOut
			}
		})(); tcErr == nil {
			return out, true
		}

		return fmt.Errorf("invalid type: %T", v), false
	}
}

func toErrorFromSlice(slice any) (out error, ok bool) {
	switch slice := slice.(type) {
	case []string:
		var errs []error
		for _, str := range slice {
			if err, okLocal := ToError(str); okLocal {
				errs = append(errs, err)
			} else {
				return err, false
			}
		}
		return CombineErrors(errs...), true
	case []error:
		return CombineErrors(slice...), true
	case []fmt.Stringer:
		var errs []error
		for _, stringer := range slice {
			if err, okLocal := ToError(stringer.String()); okLocal {
				errs = append(errs, err)
			} else {
				return err, false
			}
		}
		return CombineErrors(errs...), true
	}
	return nil, false
}

// Combines errors into single error. Error descriptions are ordered the same way
// they're passed to the function.
func CombineErrors(errs ...error) (outErr error) {
	for _, e := range errs {
		switch {
		case e != nil && outErr == nil:
			outErr = e
		case e != nil && outErr != nil:
			outErr = fmt.Errorf("%s: %s", outErr, e)
		}
	}
	return outErr
}

func CombineErrorsGeneric(errAnySlice any) (out error, ok bool) {
	err, ok := ToError(errAnySlice)
	if ok {
		return err, true
	} else {
		return err, false
	}
}

func CombineErrorsFromStrings(strs ...string) (err error) {
	var errs []error
	for _, s := range strs {
		err, _ := ToError(s)
		errs = append(errs, err)
	}
	return CombineErrors(errs...)
}
