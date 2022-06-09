package pipe

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"testing"
)

func TestCheckError(t *testing.T) {
	someErr := fmt.Errorf("some error")
	if CheckError(true, someErr) == nil {
		t.Error("Expected an error when cond is true")
	}

	if CheckError(false, someErr) != nil {
		t.Error("Expected no error when cond is false")
	}
}

func ExampleCheckError() {
	isPositive := New(Options{}).
		Append(func(args []interface{}) (int, error) {
			in, ok := args[0].(int)
			return in + 1, CheckError(!ok, fmt.Errorf("not an int"))
		}).
		Append(func(x int) bool {
			return x > 0
		})

	_, err := isPositive.Do([]interface{}{"some string"})
	fmt.Println(err)
	// Output: pipe: not an int
}

func TestCheckErrorf(t *testing.T) {
	err := CheckErrorf(true, "some error %d", 1)
	if err == nil {
		t.Error("Expected an error when cond is true")
	}
	if err.Error() != "some error 1" {
		t.Error("Unexpected error:", err)
	}

	if CheckErrorf(false, "some error %d", 1) != nil {
		t.Fatal("Expected no error when cond is false")
	}
}

func TestError(t *testing.T) {
	e := Error{errs: []error{
		&os.PathError{Op: "create", Path: "foo", Err: os.ErrExist},
	}}

	if e.Error() != "pipe: create foo: file already exists" {
		t.Error("Unexpected single error message:", e)
	}

	if errors.Unwrap(e) != e.errs[0] {
		t.Error("errors.Unwrap() must return the first error")
	}

	if !errors.Is(e, os.ErrExist) {
		t.Error("errors.Is(e, target) must be true when matching child error")
	}
	if errors.Is(e, os.ErrClosed) {
		t.Error("errors.Is(e, target) must be false when no matching child error")
	}

	var linkErr *os.LinkError
	if errors.As(e, &linkErr) {
		t.Error("errors.As(e, target) must be false when no matching child error")
	}
	var pathErr *os.PathError
	if !errors.As(e, &pathErr) {
		t.Error("errors.As(e, target) must be true when matching child error")
	}
	if !reflect.DeepEqual(pathErr, e.errs[0]) {
		t.Error("Path error from errors.As() should be set")
	}
}

func TestErrorMulti(t *testing.T) {
	e := Error{errs: []error{
		fmt.Errorf("error 1"),
		fmt.Errorf("error 2"),
		&os.PathError{Op: "create", Path: "foo", Err: os.ErrExist},
	}}

	if expect := "pipe: multiple errors: error 1; error 2; create foo: file already exists"; e.Error() != expect {
		t.Errorf("Error() must equal %q, but found: %q", expect, e.Error())
	}

	if errors.Unwrap(e) != nil {
		t.Error("errors.Unwrap() must return nil for multiple errors")
	}

	if errors.Is(e, os.ErrExist) {
		t.Error("errors.Is(e, target) must be false when matching more than 1 error")
	}
	if errors.Is(e, os.ErrClosed) {
		t.Error("errors.Is(e, target) must be false when matching more than 1 error")
	}

	var linkErr *os.LinkError
	if errors.As(e, &linkErr) {
		t.Error("errors.As(e, target) must be false when matching more than 1 error")
	}
	var pathErr *os.PathError
	if errors.As(e, &pathErr) {
		t.Error("errors.As(e, target) false when matching more than 1 error")
	}
}
