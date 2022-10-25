package main

import (
	"encoding/base64"
	"fmt"
	"path"

	"github.com/hack-pad/hackpadfs"
)

const configBin = "bin"

func (a App) moduleBinPath(module Module) string {
	return path.Join(a.configDir, configBin, module.Name)
}

func (a App) add(module Module) error {
	scriptPath := a.moduleBinPath(module)
	err := hackpadfs.MkdirAll(a.fs, path.Dir(scriptPath), 0700)
	if err != nil {
		return err
	}
	scriptFile, err := hackpadfs.OpenFile(a.fs, scriptPath, hackpadfs.FlagWriteOnly|hackpadfs.FlagCreate|hackpadfs.FlagTruncate, 0700)
	if err != nil {
		return err
	}
	defer scriptFile.Close()

	safePath := base64.StdEncoding.EncodeToString([]byte(module.Path)) // shebangs do not support spaces or quotes, so encode it

	// Script shebang should run as follows:
	// goop run abc123== -decode-module -name ~/.config/goop/bin/foo arg1 arg2 ...
	script := fmt.Sprintf("#!/usr/bin/env goop exec -encoded-module %s --\n", safePath)
	_, err = hackpadfs.WriteFile(scriptFile, []byte(script))
	return err
}
