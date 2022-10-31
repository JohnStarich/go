package main

import (
	"errors"
	"fmt"
	"path"

	"github.com/hack-pad/hackpadfs"
	"github.com/urfave/cli/v2"
)

func (a App) info(c *cli.Context) error {
	binPath := path.Join(a.configDir, configBin)
	dirEntries, err := hackpadfs.ReadDir(a.fs, binPath)
	if err != nil && !errors.Is(err, hackpadfs.ErrNotExist) {
		return err
	}

	fmt.Fprintf(c.App.Writer, "Installed: (%s)\n", binPath)
	for _, entry := range dirEntries {
		isInstalled, err := isAppExecutable(a.fs, entry.Type(), path.Join(binPath, entry.Name()))
		if err != nil {
			return err
		}
		if isInstalled {
			fmt.Fprintln(c.App.Writer, "-", entry.Name())
		}
	}
	return nil
}

func isAppExecutable(fs hackpadfs.FS, mode hackpadfs.FileMode, filePath string) (bool, error) {
	if !mode.IsRegular() && mode&hackpadfs.ModeSymlink == 0 {
		return false, nil
	}
	f, err := fs.Open(filePath)
	if err != nil {
		return false, err
	}
	defer f.Close()

	const expectedShebangPrefix = `#!/usr/bin/env goop `
	shebangPrefix := make([]byte, len(expectedShebangPrefix))
	_, err = f.Read(shebangPrefix)
	if err != nil {
		return false, err
	}
	return expectedShebangPrefix == string(shebangPrefix), nil
}
