package pipe

import (
	"reflect"
	"testing"
)

func TestFunnel(t *testing.T) {
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
		results, err := p.DoFunnel(multiArgs[:5])
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
		results, err := p.DoFunnel(multiArgs)
		if err == nil || err.Error() != "pipe: input was 5" {
			t.Error("Expected error to be 'input was 5', got:", err)
		}
		if results != nil {
			t.Error("Expected nil results, got:", results)
		}
	})
}
