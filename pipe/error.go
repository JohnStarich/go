package pipe

import "fmt"

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
