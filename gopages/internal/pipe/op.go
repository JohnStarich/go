// Package pipe helps chain together multiple error-returning operations, such that the first error is returned immediately.
// This is especially useful on error guards for eliminating boilerplate and easily achieving full test coverage.
//
package pipe

// Op creates an error-returning operation abstraction to help eliminate error handling boilerplate
type Op interface {
	Do() error
}

// OpFunc is an Op and is interchangeable with the corresponding function type.
// OpFuncs are useful for inlining custom functions as Ops.
type OpFunc func() error

func (o OpFunc) Do() error {
	return o()
}

type chain []Op

// Chain combines each Op into a chain. When executed, each Op is run in-order
// and the first error is returned immediately.
func Chain(ops ...Op) Op {
	return chain(ops)
}

// ChainFuncs is identical to Chain, but takes the more convenient OpFunc type instead
func ChainFuncs(opFuncs ...OpFunc) Op {
	ops := make(chain, len(opFuncs))
	for i := range ops {
		ops[i] = opFuncs[i]
	}
	return ops
}

func (c chain) Do() error {
	for _, op := range c {
		if err := op.Do(); err != nil {
			return err
		}
	}
	return nil
}

// ErrIf returns 'err' if 'cond' is true, nil otherwise
func ErrIf(cond bool, err error) error {
	if cond {
		return err
	}
	return nil
}
