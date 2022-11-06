package main

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"

	"github.com/hack-pad/hackpadfs/mem"
	"github.com/stretchr/testify/require"
)

type testAppOptions struct {
	runCmd func(*TestApp, *exec.Cmd) error
}

type TestApp struct {
	App

	options testAppOptions
}

func newTestApp(t *testing.T, options testAppOptions) *TestApp {
	t.Helper()

	fs, err := mem.NewFS()
	require.NoError(t, err)
	testApp := &TestApp{
		options: options,
	}
	testApp.App = App{
		errWriter:      newTestWriter(t),
		fs:             fs,
		getEnv:         func(string) string { return "" },
		outWriter:      newTestWriter(t),
		runCmd:         testApp.runCmd,
		staticBinDir:   "bin",
		staticCacheDir: "cache",
	}
	return testApp
}

func (t *TestApp) Stdout() string {
	return t.outWriter.(testWriter).out.String()
}

func (t *TestApp) Stderr() string {
	return t.errWriter.(testWriter).out.String()
}

func (t *TestApp) runCmd(cmd *exec.Cmd) error {
	cmd.Stdin = nil
	cmd.Stdout = t.outWriter
	cmd.Stderr = t.errWriter
	return t.options.runCmd(t, cmd)
}

type testWriter struct {
	testingT *testing.T
	out      *bytes.Buffer
}

func newTestWriter(t *testing.T) testWriter {
	return testWriter{
		testingT: t,
		out:      bytes.NewBuffer(nil),
	}
}

func (w testWriter) Write(b []byte) (n int, err error) {
	w.testingT.Log(strings.TrimSuffix(string(b), "\n"))
	n, err = w.out.Write(b)
	return
}
