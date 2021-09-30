package pipe

import (
	"fmt"
	"reflect"
)

type Pipe struct {
	ops []reflect.Value
}

func New() Pipe {
	return Pipe{}
}

func (p Pipe) Do(args ...interface{}) ([]interface{}, error) {
	argVals := []reflect.Value{reflect.ValueOf(args)}
	for _, op := range p.ops {
		var err error
		argVals, err = splitErrValue(op.Call(argVals))
		if err != nil {
			return nil, err
		}
	}

	resultVals := argVals
	results := make([]interface{}, len(resultVals))
	for i := range resultVals {
		results[i] = resultVals[i].Interface()
	}
	return results, nil
}

func (p Pipe) Append(fn interface{}) Pipe {
	p, err := p.appendFunc(fn)
	if err != nil {
		panic(err)
	}
	return p
}

var interfaceSliceType = reflect.TypeOf([]interface{}{})

func (p Pipe) appendFunc(fn interface{}) (Pipe, error) {
	op := reflect.ValueOf(fn)
	if op.Kind() != reflect.Func {
		return p, fmt.Errorf("pipe value must a function, got: %T", fn)
	}
	if len(p.ops) == 0 {
		opType := op.Type()
		if opType.NumIn() != 1 || opType.In(0) != interfaceSliceType {
			return p, fmt.Errorf("first pipe must accept 1 parameter of type []interface{}")
		}
	} else {
		lastOp := p.ops[len(p.ops)-1]
		if err := outMatchesIn(lastOp.Type(), op.Type()); err != nil {
			return p, err
		}
	}
	p.ops = append(p.ops, op)
	return p, nil
}

func splitErrValue(args []reflect.Value) ([]reflect.Value, error) {
	if len(args) == 0 {
		return args, nil
	}
	lastVal := args[len(args)-1]
	if !isErr(lastVal.Type()) {
		return args, nil
	}

	errInt := lastVal.Interface()
	var err error
	if errInt != nil {
		err = errInt.(error)
	}
	return args[:len(args)-1], err
}

var errType = reflect.TypeOf((*error)(nil)).Elem()

func isErr(v reflect.Type) bool {
	return v.Implements(errType)
}

func outMatchesIn(outFn, inFn reflect.Type) error {
	var outTypes []reflect.Type
	for i := 0; i < outFn.NumOut(); i++ {
		outTypes = append(outTypes, outFn.Out(i))
	}
	var inTypes []reflect.Type
	for i := 0; i < inFn.NumIn(); i++ {
		inTypes = append(inTypes, inFn.In(i))
	}

	if len(outTypes) == len(inTypes)+1 && isErr(outTypes[len(outTypes)-1]) {
		outTypes = outTypes[:len(outTypes)-1]
	}

	if len(outTypes) != len(inTypes) {
		return fmt.Errorf("new function's parameter types do not match output function's return types: %v != %v", outTypes, inTypes)
	}

	for i := range outTypes {
		out := outTypes[i]
		in := inTypes[i]
		if out != in {
			return fmt.Errorf("new function's parameter type %v does not match the expected return type %v", in, out)
		}
	}
	return nil
}
