package main

import (
	"os/exec"
	"path"
	"testing"

	"github.com/hack-pad/hackpadfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemove(t *testing.T) {
	t.Parallel()

	t.Run("does not exist", func(t *testing.T) {
		t.Parallel()
		app := newTestApp(t, testAppOptions{})
		err := app.Run([]string{"", "rm", "-name", "foo"})
		assert.NoError(t, err)
	})

	t.Run("exists", func(t *testing.T) {
		t.Parallel()
		const name = "foo"
		var binDir, installDir string
		app := newTestApp(t, testAppOptions{
			runCmd: func(app *TestApp, cmd *exec.Cmd) error {
				binDir = app.staticBinDir
				installDir = app.packageInstallDir(name)
				require.NoError(t, hackpadfs.MkdirAll(app.fs, binDir, 0700))
				require.NoError(t, hackpadfs.MkdirAll(app.fs, installDir, 0700))
				return hackpadfs.WriteFullFile(app.fs, path.Join(installDir, name), nil, 0700)
			},
		})
		err := app.Run([]string{"", "install", "-name", name, "-p", thisPackage})
		require.NoError(t, err)
		dir, err := hackpadfs.ReadDir(app.fs, binDir)
		assert.NoError(t, err)
		assert.NotEmpty(t, dir)
		dir, err = hackpadfs.ReadDir(app.fs, path.Dir(installDir))
		assert.NoError(t, err)
		assert.NotEmpty(t, dir)

		err = app.Run([]string{"", "rm", "-name", name})
		assert.NoError(t, err)
		dir, err = hackpadfs.ReadDir(app.fs, binDir)
		assert.NoError(t, err)
		assert.Empty(t, dir)
		dir, err = hackpadfs.ReadDir(app.fs, path.Dir(installDir))
		assert.NoError(t, err)
		assert.Empty(t, dir)
	})
}
