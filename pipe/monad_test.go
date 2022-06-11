package pipe

import (
	"reflect"
	"testing"
)

func TestMap(t *testing.T) {
	t.Parallel()
	p := New(Options{}).Append(func(args []interface{}) (int, error) {
		in := args[0].(int)
		return in, CheckErrorf(in == 5, "input was 5")
	})

	var multiArgs [][]interface{}
	for i := 0; i < 10; i++ {
		multiArgs = append(multiArgs, []interface{}{
			i,
		})
	}

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		results, err := Map(p, multiArgs[:5])
		if err != nil {
			t.Error("Unexpected error:", err)
		}
		expect := [][]interface{}{
			{0},
			{1},
			{2},
			{3},
			{4},
		}
		if !reflect.DeepEqual(expect, results) {
			t.Errorf("%v != %v", expect, results)
		}
	})

	t.Run("failed", func(t *testing.T) {
		t.Parallel()
		results, err := Map(p, multiArgs)
		if err == nil || err.Error() != "pipe: input was 5" {
			t.Error("Expected error to be 'input was 5', got:", err)
		}
		if results != nil {
			t.Error("Expected nil results, got:", results)
		}
	})
}

func TestFilter(t *testing.T) {
	t.Parallel()
	p := New(Options{}).Append(func(args []interface{}) (int, error) {
		in := args[0].(int)
		return in, CheckErrorf(in != 5, "input was not 5")
	})

	var multiArgs [][]interface{}
	for i := 0; i < 10; i++ {
		multiArgs = append(multiArgs, []interface{}{
			i,
		})
	}

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		results, err := Filter(p, multiArgs)
		if err != nil {
			t.Error("Unexpected error:", err)
		}
		expect := [][]interface{}{
			{5},
		}
		if !reflect.DeepEqual(expect, results) {
			t.Errorf("%v != %v", expect, results)
		}
	})

	t.Run("failed", func(t *testing.T) {
		t.Parallel()
		results, err := Filter(p, multiArgs[:5])
		if err == nil || err.Error() != "pipe: input was not 5" {
			t.Error("Expected error to be 'input was 5', got:", err)
		}
		if results != nil {
			t.Error("Expected nil results, got:", results)
		}
	})
}
