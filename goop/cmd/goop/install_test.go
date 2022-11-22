package main

import (
	"errors"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hack-pad/hackpadfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstall(t *testing.T) {
	t.Parallel()
	t.Run("pattern failure", func(t *testing.T) {
		t.Parallel()
		const name = "foo"
		app := newTestApp(t, testAppOptions{})
		err := app.Run([]string{"", "install", "-name", name, "-p", thisPackage + "/..."})
		assert.EqualError(t, err, `package pattern must not use the '/...' operator: "github.com/johnstarich/go/goop/cmd/goop/..."`)

		assert.Empty(t, app.Stderr())
		assert.Empty(t, app.Stdout())
	})

	t.Run("build failure", func(t *testing.T) {
		t.Parallel()
		const name = "foo"
		someErr := errors.New("some error")
		app := newTestApp(t, testAppOptions{
			runCmd: func(app *TestApp, cmd *exec.Cmd) error {
				return someErr
			},
		})
		err := app.Run([]string{"", "install", "-name", name, "-p", thisPackage})
		assert.ErrorIs(t, err, someErr)

		assert.Equal(t, strings.TrimSpace(`
Building "github.com/johnstarich/go/goop/cmd/goop"...
Env: PWD="" GOBIN="cache/install/foo"
Running 'go install github.com/johnstarich/go/goop/cmd/goop@latest'...
`), strings.TrimSpace(app.Stderr()))
		assert.Empty(t, app.Stdout())
	})

	t.Run("install with name", func(t *testing.T) {
		t.Parallel()
		const name = "foo"
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
				default:
					t.Errorf("Unexpected command: %q", cmd.Args[0])
				}
				return nil
			},
		})
		err := app.Run([]string{"", "install", "-name", name, "-p", thisPackage})
		assert.NoError(t, err)

		assert.Equal(t, strings.TrimSpace(`
Building "github.com/johnstarich/go/goop/cmd/goop"...
Env: PWD="" GOBIN="cache/install/foo"
Running 'go install github.com/johnstarich/go/goop/cmd/goop@latest'...
Build successful.
`), strings.TrimSpace(app.Stderr()))
		assert.Empty(t, app.Stdout())

		assert.Equal(t, [][]string{
			{"go", "install", thisPackage + "@latest"},
		}, commandsToRun)
		assert.Equal(t, "go", filepath.Base(commandPaths[0]))

		dir, err := hackpadfs.ReadDir(app.fs, "cache/install/foo")
		require.NoError(t, err)
		require.Len(t, dir, 1)
		binary := dir[0]
		assert.Equal(t, name, binary.Name())
		binFile, err := hackpadfs.ReadFile(app.fs, "bin/foo")
		assert.NoError(t, err)
		assert.Equal(t, "#!/usr/bin/env -S goop exec -encoded-name Zm9v -encoded-package Z2l0aHViLmNvbS9qb2huc3RhcmljaC9nby9nb29wL2NtZC9nb29w --\n", string(binFile))
	})

	t.Run("install without name", func(t *testing.T) {
		t.Parallel()
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
					f, err := hackpadfs.Create(app.fs, path.Join(fromEnv(cmd.Env)["GOBIN"], appName))
					require.NoError(t, err)
					require.NoError(t, f.Close())
				default:
					t.Errorf("Unexpected command: %q", cmd.Args[0])
				}
				return nil
			},
		})
		err := app.Run([]string{"", "install", "-p", thisPackage})
		assert.NoError(t, err)

		assert.Equal(t, strings.TrimSpace(`
Building "github.com/johnstarich/go/goop/cmd/goop"...
Env: PWD="" GOBIN="cache/install/goop"
Running 'go install github.com/johnstarich/go/goop/cmd/goop@latest'...
Build successful.
`), strings.TrimSpace(app.Stderr()))
		assert.Empty(t, app.Stdout())

		assert.Equal(t, [][]string{
			{"go", "install", thisPackage + "@latest"},
		}, commandsToRun)
		assert.Equal(t, "go", filepath.Base(commandPaths[0]))

		dir, err := hackpadfs.ReadDir(app.fs, "cache/install/goop")
		require.NoError(t, err)
		require.Len(t, dir, 1)
		binary := dir[0]
		assert.Equal(t, appName, binary.Name())
		binFile, err := hackpadfs.ReadFile(app.fs, "bin/goop")
		assert.NoError(t, err)
		assert.Equal(t, "#!/usr/bin/env -S goop exec -encoded-name Z29vcA== -encoded-package Z2l0aHViLmNvbS9qb2huc3RhcmljaC9nby9nb29wL2NtZC9nb29w --\n", string(binFile))
	})

	t.Run("install also reinstalls", func(t *testing.T) {
		t.Parallel()
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
				default:
					t.Errorf("Unexpected command: %q", cmd.Args[0])
				}
				return nil
			},
		})
		// binary already installed, still does a reinstall
		require.NoError(t, hackpadfs.MkdirAll(app.fs, path.Join("cache/install", appName), 0700))
		f, err := hackpadfs.Create(app.fs, path.Join("cache/install", appName, appName))
		require.NoError(t, err)
		require.NoError(t, f.Close())
		require.NoError(t, hackpadfs.Mkdir(app.fs, "bin", 0700))
		f, err = hackpadfs.Create(app.fs, path.Join("bin", appName))
		require.NoError(t, err)
		require.NoError(t, f.Close())

		err = app.Run([]string{"", "install", "-p", thisPackage})
		assert.NoError(t, err)

		assert.Equal(t, strings.TrimSpace(`
Building "github.com/johnstarich/go/goop/cmd/goop"...
Env: PWD="" GOBIN="cache/install/goop"
Running 'go install github.com/johnstarich/go/goop/cmd/goop@latest'...
Build successful.
`), strings.TrimSpace(app.Stderr()))
		assert.Empty(t, app.Stdout())

		assert.Equal(t, [][]string{
			{"go", "install", thisPackage + "@latest"},
		}, commandsToRun)
		assert.Equal(t, "go", filepath.Base(commandPaths[0]))

		dir, err := hackpadfs.ReadDir(app.fs, "cache/install/goop")
		require.NoError(t, err)
		require.Len(t, dir, 1)
		binary := dir[0]
		assert.Equal(t, appName, binary.Name())
		binFile, err := hackpadfs.ReadFile(app.fs, "bin/goop")
		assert.NoError(t, err)
		assert.Equal(t, "#!/usr/bin/env -S goop exec -encoded-name Z29vcA== -encoded-package Z2l0aHViLmNvbS9qb2huc3RhcmljaC9nby9nb29wL2NtZC9nb29w --\n", string(binFile))
	})
}
