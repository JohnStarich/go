package main

import (
	"encoding/base64"
	"fmt"
	"os/exec"
	"path"

	"github.com/urfave/cli/v2"
)

func (a App) packageInstallDir(name string) string {
	return path.Join(a.cacheDir, "install", name)
}

func (a App) exec(c *cli.Context) error {
	decode := func(s string) (string, error) {
		b, err := base64.StdEncoding.DecodeString(s)
		return string(b), err
	}
	name, err := decode(c.String("encoded-name"))
	if err != nil {
		return err
	}
	packagePattern, err := decode(c.String("encoded-package"))
	if err != nil {
		return err
	}
	pkg, err := a.parsePackagePattern(packagePattern)
	if err != nil {
		return err
	}

	binaryPath, err := a.build(c.Context, name, pkg, false)
	if err != nil {
		return err
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
