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
		isInstalled, err := isAppExecutable(a.fs, path.Join(binPath, entry.Name()))
		if err != nil {
			return err
		}
		if isInstalled {
			fmt.Fprintln(c.App.Writer, "-", entry.Name())
		}
	}
	return nil
}

func isAppExecutable(fs hackpadfs.FS, filePath string) (bool, error) {
	f, err := fs.Open(filePath)
	if err != nil {
		if errors.Is(err, hackpadfs.ErrNotExist) {
			err = nil
		}
		return false, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return false, err
	}
	if !info.Mode().IsRegular() && info.Mode()&hackpadfs.ModeSymlink == 0 {
		return false, nil
	}

	expectedShebangPrefix := makeShebang("goop ")
	shebangPrefix := make([]byte, len(expectedShebangPrefix))
	_, err = f.Read(shebangPrefix)
	if err != nil {
		return false, err
	}
	return expectedShebangPrefix == string(shebangPrefix), nil
}
