package main

import (
	"context"
	"os"
	"os/exec"

	"github.com/hack-pad/hackpadfs"
	"github.com/pkg/errors"
)

func (a App) build(ctx context.Context, module Module) (string, error) {
	installDir := a.moduleInstallDir(module)
	err := hackpadfs.MkdirAll(a.fs, installDir, 0700)
	if err != nil {
		return "", err
	}
	gobin, err := a.toOSPath(installDir)
	if err != nil {
		return "", err
	}
	cmd := exec.CommandContext(ctx, "go", "install", module.InstallPath())
	cmd.Env = append(os.Environ(), toEnv(map[string]string{
		"GOBIN": gobin,
	})...)
	if err := a.runCmd(cmd); err != nil {
		return "", err
	}

	binaryPath, found, err := findBinary(a.fs, installDir, module.Name)
	if err != nil {
		return "", err
	}
	if !found {
		return "", errors.Errorf("go install result not found at path: %s", installDir)
	}
	return binaryPath, nil
}
