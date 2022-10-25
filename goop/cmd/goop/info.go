package main

import (
	"errors"
	"fmt"
	"path"

	"github.com/hack-pad/hackpadfs"
	"github.com/urfave/cli/v2"
)

func (a App) info(c *cli.Context) error {
	dirEntries, err := hackpadfs.ReadDir(a.fs, path.Join(a.configDir, configBin))
	if err != nil && !errors.Is(err, hackpadfs.ErrNotExist) {
		return err
	}

	fmt.Fprintln(c.App.Writer, "Installed:")
	for _, entry := range dirEntries {
		if entry.Type().IsRegular() {
			fmt.Fprintln(c.App.Writer, "-", entry.Name())
		}
	}
	return nil
}
