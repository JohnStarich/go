package main

import (
	"encoding/base64"
	"fmt"
	"path"
	"strings"

	"github.com/hack-pad/hackpadfs"
)

func (a App) packageBinPath(name string) (string, error) {
	binDir, err := a.userBinDir()
	return path.Join(binDir, name), err
}

func (a App) userBinDir() (string, error) {
	binDir := a.staticBinDir
	appBinEnvironmentVar := strings.ToUpper(appName) + "_BIN"
	if configBin := a.getEnv(appBinEnvironmentVar); configBin != "" {
		var err error
		binDir, err = a.fromOSPath(configBin)
		if err != nil {
			return "", err
		}
	}
	return binDir, nil
}

func (a App) add(name string, pkg Package) error {
	scriptPath, err := a.packageBinPath(name)
	if err != nil {
		return err
	}
	err = hackpadfs.MkdirAll(a.fs, path.Dir(scriptPath), 0700)
	if err != nil {
		return err
	}
	scriptFile, err := hackpadfs.OpenFile(a.fs, scriptPath, hackpadfs.FlagWriteOnly|hackpadfs.FlagCreate|hackpadfs.FlagTruncate, 0700)
	if err != nil {
		return err
	}
	defer scriptFile.Close()

	encode := func(s string) string {
		return base64.StdEncoding.EncodeToString([]byte(s))
	}
	// Script shebang should run as follows:
	// goop exec -name foo -encoded-package abc123== -- ~/.config/goop/bin/foo arg1 arg2 ...
	script := fmt.Sprintf("goop exec -encoded-name %s -encoded-package %s --\n",
		// shebangs do not support spaces or quotes, so encode all variables
		encode(name),
		encode(pkg.Path),
	)
	_, err = hackpadfs.WriteFile(scriptFile, []byte(makeShebang(script)))
	return err
}

func makeShebang(s string) string {
	return "#!/usr/bin/env -S " + s
}
