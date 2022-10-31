package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"

	"github.com/hack-pad/hackpadfs"
	"github.com/pkg/errors"
)

func (a App) build(ctx context.Context, name string, pkg Package, alwaysBuild bool) (string, error) {
	desiredPath := path.Join(a.packageInstallDir(name), name)
	const (
		windowsGOOS = "windows"
		windowsExt  = ".exe"
	)
	if runtime.GOOS == windowsGOOS {
		desiredPath += windowsExt
	}
	// TODO support checking version hash of file-based modules
	if !alwaysBuild {
		info, err := hackpadfs.Stat(a.fs, desiredPath)
		if err == nil && info.Mode().IsRegular() {
			return desiredPath, nil
		}
		if err != nil && !errors.Is(err, hackpadfs.ErrNotExist) {
			return "", err
		}
	}
	fmt.Fprintf(a.errWriter, "Building %q...\n", pkg.Path)
	return desiredPath, a.buildAtPath(ctx, name, pkg, desiredPath)
}

func (a App) buildAtPath(ctx context.Context, name string, pkg Package, desiredPath string) error {
	installDir := a.packageInstallDir(name)
	err := hackpadfs.MkdirAll(a.fs, installDir, 0700)
	if err != nil {
		return err
	}
	gobin, err := a.toOSPath(installDir)
	if err != nil {
		return err
	}

	workingDir, installPattern := pkg.InstallPaths()
	args := []string{"install", installPattern}
	fmt.Fprintf(a.errWriter, "Env: PWD=%q GOBIN=%q\nRunning 'go %s'...\n", workingDir, gobin, strings.Join(args, " "))
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = workingDir
	cmd.Env = append(os.Environ(), toEnv(map[string]string{
		"GOBIN": gobin,
	})...)
	if err := a.runCmd(cmd); err != nil {
		return err
	}

	binaryPath, found, err := findBinary(a.fs, installDir)
	if err != nil {
		return err
	}
	if !found {
		return errors.Errorf("go install result not found at path: %s", installDir)
	}

	if binaryPath != desiredPath {
		err := hackpadfs.Rename(a.fs, binaryPath, desiredPath)
		if err != nil {
			return err
		}
	}
	fmt.Fprintln(a.errWriter, "Build successful.")
	return nil
}

func findBinary(fs hackpadfs.FS, installDir string) (string, bool, error) {
	dirEntries, err := hackpadfs.ReadDir(fs, installDir)
	if err != nil && !errors.Is(err, hackpadfs.ErrNotExist) {
		return "", false, err
	}
	for _, entry := range dirEntries {
		if entry.Type().IsRegular() {
			return path.Join(installDir, entry.Name()), true, nil
		}
	}
	return "", false, nil
}
