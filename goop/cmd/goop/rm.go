package main

import (
	"github.com/hack-pad/hackpadfs"
	"github.com/urfave/cli/v2"
)

func (a App) rm(c *cli.Context) error {
	module, err := a.parseModulePathArg(c.String("module"))
	if err != nil {
		return err
	}

	installDir := a.moduleInstallDir(module)
	if err := hackpadfs.RemoveAll(a.fs, installDir); err != nil {
		return err
	}
	binPath := a.moduleBinPath(module)
	if err := hackpadfs.RemoveAll(a.fs, binPath); err != nil {
		return err
	}
	return nil
}
