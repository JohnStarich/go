package main

import (
	"github.com/hack-pad/hackpadfs"
	"github.com/urfave/cli/v2"
)

func (a App) rm(c *cli.Context) error {
	name := c.String("name")
	binPath := a.packageBinPath(name)
	if isInstalled, err := isAppExecutable(a.fs, binPath); !isInstalled || err != nil {
		return err
	}

	if err := hackpadfs.RemoveAll(a.fs, binPath); err != nil {
		return err
	}
	installDir := a.packageInstallDir(name)
	if err := hackpadfs.RemoveAll(a.fs, installDir); err != nil {
		return err
	}
	return nil
}
