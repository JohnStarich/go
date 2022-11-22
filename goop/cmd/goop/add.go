package main

import (
	"encoding/base64"
	"fmt"
	"path"

	"github.com/hack-pad/hackpadfs"
	"github.com/johnstarich/go/pipe"
)

func (a App) packageBinPath(name string) (string, error) {
	binDir, err := a.userBinDir()
	return path.Join(binDir, name), err
}

func (a App) userBinDir() (string, error) {
	binDir := a.staticBinDir
	var err error
	const appBinEnvironmentVar = "GOOP_BIN"
	if configBin := a.getEnv(appBinEnvironmentVar); configBin != "" {
		binDir, err = a.fromOSPath(configBin)
	}
	return binDir, err
}

type addArgs struct {
	App     App
	Name    string
	Package Package
}

const binPermission = 0700

var addPipe = pipe.New(pipe.Options{}).
	Append(func(args []interface{}) addArgs {
		return args[0].(addArgs)
	}).
	Append(func(args addArgs) (addArgs, string, error) {
		scriptPath, err := args.App.packageBinPath(args.Name)
		return args, scriptPath, err
	}).
	Append(func(args addArgs, scriptPath string) (addArgs, string, error) {
		err := hackpadfs.MkdirAll(args.App.fs, path.Dir(scriptPath), binPermission)
		return args, scriptPath, err
	}).
	Append(func(args addArgs, scriptPath string) (addArgs, string, error) {
		err := hackpadfs.MkdirAll(args.App.fs, path.Dir(scriptPath), binPermission)
		return args, scriptPath, err
	}).
	Append(func(args addArgs, scriptPath string) error {
		encode := func(s string) string {
			return base64.StdEncoding.EncodeToString([]byte(s))
		}
		// Script shebang should run as follows:
		// goop exec -name foo -encoded-package abc123== -- ~/.config/goop/bin/foo arg1 arg2 ...
		script := fmt.Sprintf("goop exec -encoded-name %s -encoded-package %s --\n",
			// shebangs do not support spaces or quotes, so encode all variables
			encode(args.Name),
			encode(args.Package.Path),
		)
		return hackpadfs.WriteFullFile(args.App.fs, scriptPath, []byte(makeShebang(script)), binPermission)
	})

func (a App) add(name string, pkg Package) error {
	_, err := addPipe.Do(addArgs{
		App:     a,
		Name:    name,
		Package: pkg,
	})
	return err
}

func makeShebang(s string) string {
	return "#!/usr/bin/env -S " + s
}
