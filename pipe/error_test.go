package pipe

import (
	"fmt"
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
	isPositive := New().
		Append(func(args []interface{}) (int, error) {
			in, ok := args[0].(int)
			return in + 1, CheckError(!ok, fmt.Errorf("not an int!"))
		}).
		Append(func(x int) bool {
			return x > 0
		})

	_, err := isPositive.Do([]interface{}{"some string"})
	fmt.Println(err)
	// Output: not an int!
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
