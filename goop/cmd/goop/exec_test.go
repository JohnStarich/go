package main

import (
	"encoding/base64"
	"fmt"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hack-pad/hackpadfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const thisPackage = "github.com/johnstarich/go/goop/cmd/goop"

func TestExec(t *testing.T) {
	t.Parallel()

	t.Run("invalid package name", func(t *testing.T) {
		encodedName := base64EncodeString("foo")
		encodedPackage := base64EncodeString("bar")
		app := newTestApp(t, testAppOptions{
			runCmd: func(app *TestApp, cmd *exec.Cmd) error {
				return runCmd(cmd) // use real command
			},
		})
		err := app.Run([]string{"", "exec", "-encoded-name", encodedName, "-encoded-package", encodedPackage})
		assert.EqualError(t, err, "pipe: go install bar@latest: exit status 1")
	})

	t.Run("exec install then run", func(t *testing.T) {
		const name = "foo"
		encodedName := base64EncodeString(name)
		encodedPackage := base64EncodeString(thisPackage)
		var commandsToRun [][]string
		var commandPaths []string
		app := newTestApp(t, testAppOptions{
			runCmd: func(app *TestApp, cmd *exec.Cmd) error {
				commandsToRun = append(commandsToRun, cmd.Args)
				commandPaths = append(commandPaths, cmd.Path)
				switch cmd.Args[0] {
				case "go":
					assert.Equal(t, []string{
						"go",
						"install",
						thisPackage + "@latest",
					}, cmd.Args)
					f, err := hackpadfs.Create(app.fs, path.Join(fromEnv(cmd.Env)["GOBIN"], name))
					require.NoError(t, err)
					require.NoError(t, f.Close())
				case "foo":
					fmt.Fprintln(cmd.Stdout, "Running foo!")
				default:
					t.Errorf("Unexpected command: %q", cmd.Args[0])
				}
				return nil
			},
		})
		err := app.Run([]string{"", "exec", "-encoded-name", encodedName, "-encoded-package", encodedPackage})
		assert.NoError(t, err)

		assert.Equal(t, strings.TrimSpace(`
Building "github.com/johnstarich/go/goop/cmd/goop"...
Env: PWD="" GOBIN="cache/install/foo"
Running 'go install github.com/johnstarich/go/goop/cmd/goop@latest'...
Build successful.
`), strings.TrimSpace(app.Stderr()))
		assert.Equal(t, "Running foo!\n", app.Stdout())

		assert.Equal(t, [][]string{
			{"go", "install", thisPackage + "@latest"},
			{"foo"},
		}, commandsToRun)
		const goCmdIndex = 0
		assert.Equal(t, "go", filepath.Base(commandPaths[goCmdIndex]))
		commandPaths = append(commandPaths[:goCmdIndex], commandPaths[goCmdIndex+1:]...)
		assert.Equal(t, []string{
			"cache/install/foo/foo",
		}, commandPaths)
	})

	t.Run("exec already installed run with args", func(t *testing.T) {
		const name = "foo"
		encodedName := base64EncodeString(name)
		encodedPackage := base64EncodeString(thisPackage)
		var commandsToRun [][]string
		var commandPaths []string
		app := newTestApp(t, testAppOptions{
			runCmd: func(_ *TestApp, cmd *exec.Cmd) error {
				commandsToRun = append(commandsToRun, cmd.Args)
				commandPaths = append(commandPaths, cmd.Path)
				switch cmd.Args[0] {
				case "foo":
					fmt.Fprintln(cmd.Stdout, "Running foo!")
				default:
					t.Errorf("Unexpected command: %q", cmd.Args[0])
				}
				return nil
			},
		})
		require.NoError(t, hackpadfs.MkdirAll(app.fs, app.packageInstallDir(name), 0700))
		f, err := hackpadfs.Create(app.fs, path.Join(app.packageInstallDir(name), name))
		require.NoError(t, err)
		require.NoError(t, f.Close())

		err = app.Run([]string{
			"", "exec", "-encoded-name", encodedName, "-encoded-package", encodedPackage,
			"--", name, "-bar",
		})
		assert.NoError(t, err)

		assert.Equal(t, "", app.Stderr())
		assert.Equal(t, "Running foo!\n", app.Stdout())

		assert.Equal(t, [][]string{
			{name, "-bar"},
		}, commandsToRun)
		assert.Equal(t, []string{
			"cache/install/foo/foo",
		}, commandPaths)
	})
}

func fromEnv(env []string) map[string]string {
	m := make(map[string]string)
	for _, envPair := range env {
		equalIndex := strings.IndexRune(envPair, '=')
		if equalIndex == -1 {
			panic("Invalid env key-value pair: " + envPair)
		}
		key, value := envPair[:equalIndex], envPair[equalIndex+1:]
		m[key] = value
	}
	return m
}

func base64EncodeString(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}
