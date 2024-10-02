// Package minmax contains min and max functions.
package minmax

type ordered interface {
	~int | ~int64 | ~uint | ~uint64
}

// Min returns the smallest of a and b
func Min[Value ordered](a, b Value) Value {
	if a < b {
		return a
	}
	return b
}

// Max returns the largest of a and b
func Max[Value ordered](a, b Value) Value {
	if a > b {
		return a
	}
	return b
}
