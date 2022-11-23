package main

import (
	"github.com/hack-pad/hackpadfs"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func (a App) rm(cmd *cobra.Command, args []string) error {
	name, err := cmd.Flags().GetString("name")
	if err != nil {
		return err
	}
	binPath, err := a.packageBinPath(name)
	if err != nil {
		return err
	}
	if isInstalled, err := isAppExecutable(a.fs, binPath); !isInstalled || err != nil {
		if errors.Is(err, hackpadfs.ErrNotExist) {
			err = nil
		}
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
