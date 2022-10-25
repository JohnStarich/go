package main

import (
	"encoding/base64"
	"fmt"
	"os/exec"
	"path"
	"strings"

	"github.com/hack-pad/hackpadfs"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

func (a App) moduleInstallDir(module Module) string {
	return path.Join(a.cacheDir, "install", module.Name)
}

func (a App) exec(c *cli.Context) error {
	modulePath := c.String("encoded-module")
	decodedPath, err := base64.StdEncoding.DecodeString(modulePath)
	if err != nil {
		return err
	}
	modulePath = string(decodedPath)
	module, err := a.parseModulePathArg(modulePath)
	if err != nil {
		return err
	}

	installDir := a.moduleInstallDir(module)
	binaryPath, found, err := findBinary(a.fs, installDir, module.Name)
	if err != nil {
		return err
	}
	if !found {
		binaryPath, err = a.build(c.Context, module)
		if err != nil {
			return err
		}
	}
	binaryOSPath, err := a.toOSPath(binaryPath)
	if err != nil {
		return err
	}

	arg0 := c.Args().First()
	args := c.Args().Tail()
	cmd := exec.CommandContext(c.Context, binaryOSPath, args...)
	cmd.Args[0] = arg0
	return a.runCmd(cmd)
}

func toEnv(envMap map[string]string) []string {
	var envKeys []string
	for key, value := range envMap {
		envKeys = append(envKeys, fmt.Sprintf("%s=%s", key, value))
	}
	return envKeys
}

func findBinary(fs hackpadfs.FS, installDir, moduleName string) (string, bool, error) {
	dirEntries, err := hackpadfs.ReadDir(fs, installDir)
	if err != nil && !errors.Is(err, hackpadfs.ErrNotExist) {
		return "", false, err
	}
	for _, entry := range dirEntries {
		if strings.HasPrefix(entry.Name(), moduleName) {
			return path.Join(installDir, entry.Name()), true, nil
		}
	}
	return "", false, nil
}
