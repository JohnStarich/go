package pipe

import (
	"errors"
	"fmt"
	"strings"
)

func CheckError(cond bool, err error) error {
	if cond {
		return err
	}
	return nil
}

func CheckErrorf(cond bool, format string, args ...interface{}) error {
	if cond {
		return fmt.Errorf(format, args...)
	}
	return nil
}

type Error struct {
	errs []error
}

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

// Unwrap implements errors.Unwrap() defined interface. Returns the first error only.
func (e Error) Unwrap() error {
	return e.errs[0]
}

// As implements errors.As() defined interface
func (e Error) As(target interface{}) bool {
	for _, candidate := range e.errs {
		if errors.As(candidate, target) {
			return true
		}
	}
	return false
}

// Is implements errors.Is() defined interface
func (e Error) Is(target error) bool {
	for _, candidate := range e.errs {
		if errors.Is(candidate, target) {
			return true
		}
	}
	return false
}
