package pipe

import (
	"errors"
	"fmt"
	"strings"
)

// CheckError returns 'err' if 'cond' is true, nil otherwise
func CheckError(cond bool, err error) error {
	if cond {
		return err
	}
	return nil
}

// CheckErrorf returns a new formatted error if 'cond' is true, nil otherwise
func CheckErrorf(cond bool, format string, args ...interface{}) error {
	if cond {
		return fmt.Errorf(format, args...)
	}
	return nil
}

// Error is an error returned from Pipe.Do().
// Can contain 1 or more errors. Conforms to the standard library's errors package interfaces like errors.As().
type Error struct {
	errs []error
}

// Error implements the builtin error interface
func (e Error) Error() string {
	if len(e.errs) == 1 {
		return fmt.Sprintf("pipe: %v", e.errs[0])
	}
	errStrs := make([]string, len(e.errs))
	for i, err := range e.errs {
		errStrs[i] = err.Error()
	}
	return fmt.Sprintf("pipe: multiple errors: %s", strings.Join(errStrs, "; "))
}

// Unwrap implements errors.Unwrap() defined interface.
// Returns the first error. If there was not exactly 1 error, returns nil.
func (e Error) Unwrap() error {
	if len(e.errs) != 1 {
		// if there's not exactly 1 error, unwrapping no longer makes sense
		return nil
	}
	return e.errs[0]
}

// As implements errors.As() defined interface
func (e Error) As(target interface{}) bool {
	if len(e.errs) != 1 {
		// if this error can't be effectively unwrapped to only 1 error, then fail error equivalence
		return false
	}
	for _, candidate := range e.errs {
		if errors.As(candidate, target) {
			return true
		}
	}
	return false
}

// Is implements errors.Is() defined interface
func (e Error) Is(target error) bool {
	if len(e.errs) != 1 {
		// if this error can't be effectively unwrapped to only 1 error, then fail error equivalence
		return false
	}
	for _, candidate := range e.errs {
		if errors.Is(candidate, target) {
			return true
		}
	}
	return false
}
