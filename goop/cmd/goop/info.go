package main

import (
	"errors"
	"fmt"
	"io"
	"path"

	"github.com/hack-pad/hackpadfs"
	"github.com/urfave/cli/v2"
)

func (a App) info(c *cli.Context) error {
	binDir, err := a.userBinDir()
	if err != nil {
		return err
	}
	dirEntries, err := hackpadfs.ReadDir(a.fs, binDir)
	if err != nil && !errors.Is(err, hackpadfs.ErrNotExist) {
		return err
	}

	fmt.Fprintf(c.App.Writer, "Installed: (%s)\n", binDir)
	for _, entry := range dirEntries {
		isInstalled, err := isAppExecutable(a.fs, path.Join(binDir, entry.Name()))
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
		if errors.Is(err, io.EOF) {
			err = nil
		}
		return false, err
	}
	return expectedShebangPrefix == string(shebangPrefix), nil
}
