package cmd

import (
	"os"
	"testing"

	"github.com/pkg/errors"
)

var exiter = os.Exit

func Exit(code int) {
	exiter(code)
}

func SetupTestExiter(t *testing.T) {
	// require testing.T to ensure this is a test and not real code
	t.Log("Setting up exiter")
	exiter = func(code int) {
		panic(errors.Errorf("Attempted to exit with exit code %d", code))
	}
}
