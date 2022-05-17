package main

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func init() {
	osExiter = func(code int) {
		panic(errors.Errorf("exited with code %d", code))
	}
}

func TestMain(t *testing.T) {
	assert.PanicsWithError(t, "exited with code 1", main)
}
