package main

import (
	"context"
	"os/exec"
	"runtime"
	"testing"

	"github.com/hack-pad/hackpadfs"
	"github.com/hack-pad/hackpadfs/mem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSystemExt(t *testing.T) {
	t.Parallel()
	assert.Equal(t, ".exe", systemExt(goosWindows))
	assert.Equal(t, "", systemExt("foo"))
}

func TestBuild(t *testing.T) {
	t.Parallel()
	t.Run("invalid stat", func(t *testing.T) {
		t.Parallel()
		app := newTestApp(t, testAppOptions{})
		_, err := app.build(context.Background(), "../../../../..", Package{}, false)
		assert.EqualError(t, err, "stat ../../../../../../../.."+systemExt(runtime.GOOS)+": invalid argument")
	})

	t.Run("build latest *nix", func(t *testing.T) {
		t.Parallel()
		const (
			name         = "foo"
			nonWindowsOS = "bar"
			somePackage  = "example.local/baz"
		)
		var commands [][]string
		app := newTestApp(t, testAppOptions{
			runCmd: func(app *TestApp, cmd *exec.Cmd) error {
				f, err := hackpadfs.Create(app.fs, "cache/install/"+name+"/"+name)
				require.NoError(t, err)
				require.NoError(t, f.Close())
				commands = append(commands, cmd.Args)
				return nil
			},
		})

		binaryPath, err := app.buildOS(context.Background(), name, Package{
			Name: name,
			Path: somePackage,
		}, false, nonWindowsOS)
		assert.NoError(t, err)
		assert.Equal(t, "cache/install/"+name+"/"+name, binaryPath)
		assert.Equal(t, [][]string{
			{"go", "install", somePackage + "@latest"},
		}, commands)
	})

	t.Run("build semver *nix", func(t *testing.T) {
		t.Parallel()
		const (
			name         = "foo"
			nonWindowsOS = "bar"
			somePackage  = "example.local/baz"
			someVersion  = "v0.1.2"
		)
		var commands [][]string
		app := newTestApp(t, testAppOptions{
			runCmd: func(app *TestApp, cmd *exec.Cmd) error {
				f, err := hackpadfs.Create(app.fs, "cache/install/"+name+"/"+name)
				require.NoError(t, err)
				require.NoError(t, f.Close())
				commands = append(commands, cmd.Args)
				return nil
			},
		})

		binaryPath, err := app.buildOS(context.Background(), name, Package{
			Name:          name,
			Path:          somePackage,
			ModuleVersion: someVersion,
		}, false, nonWindowsOS)
		assert.NoError(t, err)
		assert.Equal(t, "cache/install/"+name+"/"+name, binaryPath)
		assert.Equal(t, [][]string{
			{"go", "install", somePackage + "@" + someVersion},
		}, commands)
	})

	t.Run("build latest windows", func(t *testing.T) {
		t.Parallel()
		const (
			name        = "foo"
			somePackage = "example.local/baz"
		)
		var commands [][]string
		app := newTestApp(t, testAppOptions{
			runCmd: func(app *TestApp, cmd *exec.Cmd) error {
				f, err := hackpadfs.Create(app.fs, "cache/install/"+name+"/"+name)
				require.NoError(t, err)
				require.NoError(t, f.Close())
				commands = append(commands, cmd.Args)
				return nil
			},
		})

		binaryPath, err := app.buildOS(context.Background(), name, Package{
			Name: name,
			Path: somePackage,
		}, false, goosWindows)
		assert.NoError(t, err)
		assert.Equal(t, "cache/install/"+name+"/"+name+".exe", binaryPath)
		assert.Equal(t, [][]string{
			{"go", "install", somePackage + "@latest"},
		}, commands)
	})

	t.Run("build only once", func(t *testing.T) {
		t.Parallel()
		const (
			name         = "foo"
			nonWindowsOS = "bar"
			somePackage  = "example.local/baz"
		)
		var commands [][]string
		app := newTestApp(t, testAppOptions{
			runCmd: func(_ *TestApp, cmd *exec.Cmd) error {
				commands = append(commands, cmd.Args)
				return nil
			},
		})
		require.NoError(t, hackpadfs.MkdirAll(app.fs, "cache/install/"+name, 0700))
		f, err := hackpadfs.Create(app.fs, "cache/install/"+name+"/"+name)
		require.NoError(t, err)
		require.NoError(t, f.Close())

		binaryPath, err := app.buildOS(context.Background(), name, Package{
			Name: name,
			Path: somePackage,
		}, false, nonWindowsOS)
		assert.NoError(t, err)
		assert.Equal(t, "cache/install/"+name+"/"+name, binaryPath)
		assert.Empty(t, commands)
	})

	t.Run("build and overwrite", func(t *testing.T) {
		t.Parallel()
		const (
			name         = "foo"
			nonWindowsOS = "bar"
			somePackage  = "example.local/baz"
		)
		var commands [][]string
		app := newTestApp(t, testAppOptions{
			runCmd: func(app *TestApp, cmd *exec.Cmd) error {
				f, err := hackpadfs.Create(app.fs, "cache/install/"+name+"/"+name)
				require.NoError(t, err)
				require.NoError(t, f.Close())
				commands = append(commands, cmd.Args)
				return nil
			},
		})
		require.NoError(t, hackpadfs.MkdirAll(app.fs, "cache/install/"+name, 0700))
		f, err := hackpadfs.Create(app.fs, "cache/install/"+name+"/"+name)
		require.NoError(t, err)
		require.NoError(t, f.Close())

		binaryPath, err := app.buildOS(context.Background(), name, Package{
			Name: name,
			Path: somePackage,
		}, true, nonWindowsOS)
		assert.NoError(t, err)
		assert.Equal(t, "cache/install/"+name+"/"+name, binaryPath)
		assert.Equal(t, [][]string{
			{"go", "install", somePackage + "@latest"},
		}, commands)
	})
}

func TestFindBinary(t *testing.T) {
	t.Parallel()
	t.Run("directory missing", func(t *testing.T) {
		t.Parallel()
		fs, err := mem.NewFS()
		require.NoError(t, err)

		_, ok, err := findBinary(fs, "missing")
		assert.False(t, ok)
		assert.NoError(t, err)
	})

	t.Run("install dir is file", func(t *testing.T) {
		t.Parallel()
		fs, err := mem.NewFS()
		require.NoError(t, err)
		f, err := hackpadfs.Create(fs, "foo")
		require.NoError(t, err)
		require.NoError(t, f.Close())

		_, ok, _ := findBinary(fs, "foo")
		assert.False(t, ok, "Binary should not be found. Can't expect error though, as windows returns an empty dir listing")
	})

	t.Run("found", func(t *testing.T) {
		t.Parallel()
		fs, err := mem.NewFS()
		require.NoError(t, err)
		require.NoError(t, hackpadfs.Mkdir(fs, "foo", 0700))
		f, err := hackpadfs.Create(fs, "foo/bar")
		require.NoError(t, err)
		require.NoError(t, f.Close())

		filePath, ok, err := findBinary(fs, "foo")
		assert.Equal(t, "foo/bar", filePath)
		assert.True(t, ok)
		assert.NoError(t, err)
	})
}
