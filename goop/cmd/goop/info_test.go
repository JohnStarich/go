package main

import (
	"testing"

	"github.com/hack-pad/hackpadfs"
	"github.com/stretchr/testify/assert"
)

func TestInfo(t *testing.T) {
	t.Parallel()
	t.Run("no bin", func(t *testing.T) {
		t.Parallel()
		app := newTestApp(t, testAppOptions{})
		err := app.Run([]string{"info"})
		assert.NoError(t, err)
		assert.Equal(t, `Installed: (bin)
`, app.Stdout())
	})

	t.Run("none installed", func(t *testing.T) {
		t.Parallel()
		app := newTestApp(t, testAppOptions{})
		assert.NoError(t, hackpadfs.Mkdir(app.fs, "bin", 0700))
		err := app.Run([]string{"info"})
		assert.NoError(t, err)
		assert.Equal(t, `Installed: (bin)
`, app.Stdout())
	})

	t.Run("one installed and one unrecognized file", func(t *testing.T) {
		t.Parallel()
		app := newTestApp(t, testAppOptions{})
		assert.NoError(t, hackpadfs.Mkdir(app.fs, "bin", 0700))
		assert.NoError(t, hackpadfs.WriteFullFile(app.fs, "bin/foo", nil, 0700))
		const barShebang = `#!/usr/bin/env -S goop exec --encoded-name YmFy --encoded-package YmF6 --`
		assert.NoError(t, hackpadfs.WriteFullFile(app.fs, "bin/bar", []byte(barShebang), 0700))

		err := app.Run([]string{"info"})
		assert.NoError(t, err)
		assert.Equal(t, `Installed: (bin)
- bar
`, app.Stdout())
	})

	t.Run("unrecognized dir", func(t *testing.T) {
		t.Parallel()
		app := newTestApp(t, testAppOptions{})
		assert.NoError(t, hackpadfs.Mkdir(app.fs, "bin", 0700))
		assert.NoError(t, hackpadfs.Mkdir(app.fs, "bin/foo", 0700))
		err := app.Run([]string{"info"})
		assert.NoError(t, err)
		assert.Equal(t, `Installed: (bin)
`, app.Stdout())
	})
}
