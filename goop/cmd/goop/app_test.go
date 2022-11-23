package main

import (
	"bytes"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"testing"

	"github.com/hack-pad/hackpadfs/mem"
	osfs "github.com/hack-pad/hackpadfs/os"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testAppOptions struct {
	runCmd func(*TestApp, *exec.Cmd) error
}

type TestApp struct {
	App

	options  testAppOptions
	testingT *testing.T
}

func newTestApp(t *testing.T, options testAppOptions) *TestApp {
	t.Helper()

	fs, err := mem.NewFS()
	require.NoError(t, err)
	testApp := &TestApp{
		options:  options,
		testingT: t,
	}
	testApp.App = App{
		errWriter: newTestWriter(t),
		fs:        fs,
		getEnv:    func(string) string { return "" },
		lookPath: func(name string) (string, error) {
			return path.Join("bin", name), nil
		},
		outWriter:       newTestWriter(t),
		runCmd:          testApp.runCmd,
		staticBinDir:    "bin",
		staticCacheDir:  "cache",
		staticOSHomeDir: "home",
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
	t.testingT.Helper()
	if t.options.runCmd == nil {
		t.testingT.Fatal("No runCmd provided")
	}
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

func TestOSPath(t *testing.T) {
	t.Parallel()
	t.Run("non-OS FS", func(t *testing.T) {
		t.Parallel()
		memFS, err := mem.NewFS()
		assert.NoError(t, err)
		app := App{fs: memFS}
		osPath := "/a/b/c"
		newFSPath, err := app.fromOSPath(osPath)
		assert.NoError(t, err)
		assert.Equal(t, osPath, newFSPath)
		newOSPath, err := app.toOSPath(newFSPath)
		assert.NoError(t, err)
		assert.Equal(t, osPath, newOSPath)
	})

	t.Run("OS FS", func(t *testing.T) {
		t.Parallel()
		switch runtime.GOOS {
		case "darwin", "linux":
		default:
			t.Skip("Only testing OS path behavior on a handful of similar platforms. Hackpadfs has the rest covered.")
		}
		osFS := osfs.NewFS()
		app := App{fs: osFS}
		newFSPath, err := app.fromOSPath("/a/b/c")
		assert.NoError(t, err)
		assert.Equal(t, "a/b/c", newFSPath)
		newOSPath, err := app.toOSPath("a/b/c")
		assert.NoError(t, err)
		assert.Equal(t, "/a/b/c", newOSPath)
	})
}

func TestPanicIfError(t *testing.T) {
	t.Parallel()
	assert.PanicsWithError(t, "some error", func() {
		panicIfErr(errors.New("some error"))
	})
	assert.NotPanics(t, func() {
		panicIfErr(nil)
	})
}
