package main

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	osErr = bytes.NewBuffer(nil)
	osExiter = func(code int) {
		buf := osErr.(*bytes.Buffer)
		bytes, _ := io.ReadAll(buf)
		err := errors.New(string(bytes))
		panic(errors.Wrapf(err, "exited with code %d and output", code))
	}
	setColorOnce(false)
	os.Exit(m.Run())
}

func TestRunMain(t *testing.T) {
	t.Parallel()
	handlePanic(t, main, func(v interface{}) {
		require.Implements(t, (*error)(nil), v)
		err := v.(error)
		assert.ErrorContains(t, err, "exited with code 1")
	})
}

func handlePanic(t *testing.T, fn func(), handler func(v interface{})) {
	t.Helper()
	defer func() {
		handler(recover())
	}()
	fn()
}
