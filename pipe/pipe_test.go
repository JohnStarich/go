package pipe

import (
	"fmt"
	"reflect"
	"sort"
	"testing"
)

func TestPipeData(t *testing.T) {
	t.Parallel()
	p := New(Options{}).
		Append(func([]interface{}) []int {
			return []int{1, 2, 3}
		}).
		Append(func(v []int) ([]int, bool) {
			sort.Sort(sort.Reverse(sort.IntSlice(v)))
			return v, len(v) > 0
		}).
		Append(func(v []int, ok bool) bool {
			return ok && v[0] == 3
		})

	out, err := p.Do()
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 {
		t.Fatal("Out must have 1 return value, found:", len(out))
	}
	if out[0] != true {
		t.Error("Return value must be true, found:", out[0])
	}
}

func TestPipeErr(t *testing.T) {
	t.Parallel()
	t.Run("error", func(t *testing.T) {
		t.Parallel()
		p := New(Options{}).
			Append(func([]interface{}) ([]int, error) {
				return nil, fmt.Errorf("failed")
			}).
			Append(func(v []int) ([]int, bool) {
				sort.Sort(sort.Reverse(sort.IntSlice(v)))
				return v, len(v) > 0
			})

		out, err := p.Do([]int{1, 2, 3})
		if err == nil {
			t.Fatal("Expected err, got nil")
		}
		if out != nil {
			t.Error("Out must be unset")
		}
		if err.Error() != "pipe: failed" {
			t.Error("Unexpected error message:", err.Error())
		}
	})

	t.Run("no error", func(t *testing.T) {
		t.Parallel()
		p := New(Options{}).
			Append(func([]interface{}) []int {
				return []int{1, 2, 3}
			}).
			Append(func(v []int) ([]int, error) {
				return v, nil
			}).
			Append(func(v []int) ([]int, bool) {
				sort.Sort(sort.Reverse(sort.IntSlice(v)))
				return v, len(v) > 0
			})

		out, err := p.Do()
		if err != nil {
			t.Fatal("Expected no err, got:", err)
		}
		if len(out) != 2 {
			t.Fatal("Out must have 2 return values, found:", len(out))
		}
		if !reflect.DeepEqual(out[0], []int{3, 2, 1}) {
			t.Error("Return value 0 must be [3, 2, 1], found:", out[0])
		}
		if out[1] != true {
			t.Error("Return value 1 must be true, found:", out[1])
		}
	})
}

func TestPipeInvalid(t *testing.T) {
	t.Parallel()
	t.Run("mismatched output and input types", func(t *testing.T) {
		t.Parallel()
		assertPanics(t,
			fmt.Errorf("new function's parameter type bool does not match the expected return type []int"),
			func() {
				New(Options{}).
					Append(func([]interface{}) []int {
						return nil
					}).
					Append(func(v bool) bool {
						return v
					})
			})
	})

	t.Run("missing input param", func(t *testing.T) {
		t.Parallel()
		assertPanics(t,
			fmt.Errorf("new function's parameter types do not match output function's return types: [[]int] != []"),
			func() {
				New(Options{}).
					Append(func([]interface{}) []int {
						return nil
					}).
					Append(func() bool {
						return false
					})
			})
	})

	t.Run("zero return values and zero params", func(t *testing.T) {
		t.Parallel()
		p := New(Options{}).
			Append(func([]interface{}) {}).
			Append(func() bool {
				return false
			})
		out, err := p.Do()
		if err != nil {
			t.Fatal("Unexpected error:", err)
		}
		if !reflect.DeepEqual(out, []interface{}{false}) {
			t.Error("Unexpected out:", out)
		}
	})

	t.Run("not a function", func(t *testing.T) {
		t.Parallel()
		assertPanics(t,
			fmt.Errorf("pipe value must a function, got: string"),
			func() {
				New(Options{}).
					Append("not a func")
			})
	})

	t.Run("first pipe must accept args", func(t *testing.T) {
		t.Parallel()
		assertPanics(t,
			fmt.Errorf("first pipe must accept 1 parameter of type []interface{}"),
			func() {
				New(Options{}).
					Append(func() {})
			})
	})
}

func assertPanics(t *testing.T, value interface{}, fn func()) {
	t.Helper()
	defer func() {
		v := recover()
		shouldPanic := value != nil
		if (v != nil) != shouldPanic {
			if shouldPanic {
				t.Error("Expected panic")
			} else {
				t.Error("Unexpected panic")
			}
		}
		if shouldPanic && !reflect.DeepEqual(v, value) {
			t.Errorf("Unexpected panic value: %v != %v", v, value)
		}
	}()
	fn()
}

func TestPipeReuse(t *testing.T) {
	t.Parallel()
	p := New(Options{}).
		Append(func([]interface{}) string {
			return "hello"
		})

	p1 := p.Append(func(s string) string {
		return s + " world"
	})

	p2 := p.Append(func(s string) string {
		return s + " universe"
	})

	out1, err := p1.Do()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual([]interface{}{"hello world"}, out1) {
		t.Error("Unexpected p1 output:", out1)
	}

	out2, err := p2.Do()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual([]interface{}{"hello universe"}, out2) {
		t.Error("Unexpected p2 output:", out2)
	}
}

func TestPipeConcat(t *testing.T) {
	t.Parallel()
	t.Run("empty pipes", func(t *testing.T) {
		t.Parallel()
		empty := New(Options{}).Concat(New(Options{}))
		if !reflect.DeepEqual(New(Options{}), empty) {
			t.Error("Empty pipe concat with empty pipe should be equivalent")
		}
	})

	t.Run("data flows", func(t *testing.T) {
		t.Parallel()
		p1 := New(Options{}).
			Append(func(args []interface{}) int {
				arg0 := args[0].(int)
				return arg0 + 1
			})
		p2 := New(Options{}).
			Append(func(args []interface{}) int {
				arg0 := args[0].(int)
				return arg0 + 1
			})
		p := p1.Concat(p2)
		out, err := p.Do(0)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual([]interface{}{2}, out) {
			t.Error("Out must equal 2, got:", out)
		}
	})

	t.Run("previous error type dropped from bridge func", func(t *testing.T) {
		t.Parallel()
		p1 := New(Options{}).
			Append(func([]interface{}) (int, error) {
				return 1, nil
			})
		p2 := New(Options{}).
			Append(func(args []interface{}) int {
				input := args[0].(int)
				return input
			}).
			Append(func(i int) int {
				return i + 1
			})
		p := p1.Concat(p2)
		out, err := p.Do()
		if err != nil {
			t.Fatal("Unexpected error:", err)
		}
		if !reflect.DeepEqual([]interface{}{2}, out) {
			t.Error("Unexpected result:", out)
		}
	})

	t.Run("stops on first pipe err", func(t *testing.T) {
		t.Parallel()
		p1 := New(Options{}).
			Append(func([]interface{}) (int, error) {
				return 0, fmt.Errorf("failed")
			})
		p2 := New(Options{}).
			Append(func([]interface{}) int {
				return 0
			})
		p := p1.Concat(p2)
		out, err := p.Do(0)
		if err == nil {
			t.Fatal("Expected error")
		}
		if err.Error() != "pipe: failed" {
			t.Error("Unexpected error:", err)
		}
		if out != nil {
			t.Error("Out must be nil")
		}
	})
}

func ExamplePipe() {
	isPositive := New(Options{}).
		Append(func(args []interface{}) (int, error) {
			i, ok := args[0].(int)
			return i, CheckErrorf(!ok, "invalid int: %v", args[0])
		}).
		Append(func(i int) bool {
			return i > 0
		})

	_, err := isPositive.Do("string")
	fmt.Println(err)

	out, err := isPositive.Do(1)
	if err != nil {
		panic(err)
	}
	fmt.Println(out[0])
	// Output:
	// pipe: invalid int: string
	// true
}
