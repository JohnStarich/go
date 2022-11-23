package main

import (
	"encoding/base64"
	"fmt"
	"path"

	"github.com/hack-pad/hackpadfs"
	"github.com/johnstarich/go/pipe"
	"github.com/pkg/errors"
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
		isExecutable, err := isAppExecutable(args.App.fs, scriptPath)
		if errors.Is(err, hackpadfs.ErrNotExist) {
			err = nil
		} else if err == nil {
			err = pipe.CheckError(!isExecutable, errors.Errorf("refusing to overwrite non-goop script file: %q", scriptPath))
		}
		return args, scriptPath, err
	}).
	Append(func(args addArgs, scriptPath string) (string, error) {
		encode := func(s string) string {
			return base64.StdEncoding.EncodeToString([]byte(s))
		}
		// Script shebang should run as follows:
		// goop exec --name foo --encoded-package abc123== -- ~/.config/goop/bin/foo arg1 arg2 ...
		script := fmt.Sprintf("goop exec --encoded-name %s --encoded-package %s --\n",
			// shebangs do not support spaces or quotes, so encode all variables
			encode(args.Name),
			encode(args.Package.Path),
		)
		err := hackpadfs.WriteFullFile(args.App.fs, scriptPath, []byte(makeShebang(script)), binPermission)
		return scriptPath, err
	})

func (a App) add(name string, pkg Package) error {
	results, err := addPipe.Do(addArgs{
		App:     a,
		Name:    name,
		Package: pkg,
	})
	if err != nil {
		return err
	}
	scriptPath := results[0].(string)
	scriptOSPath, err := a.toOSPath(scriptPath)
	if err != nil {
		return err
	}
	executableOSPath, err := a.lookPath(name)
	if err != nil {
		fmt.Fprintf(a.errWriter, "WARNING: Failed to find %q on PATH. Check to ensure the directory %q is added to your PATH environment variable.\n", name, path.Dir(scriptOSPath))
	}
	if scriptOSPath != executableOSPath {
		fmt.Fprintf(a.errWriter, "WARNING: Executable on PATH for name %q does not match install location: %q != %q\n", name, executableOSPath, scriptOSPath)
	}
	return nil
}

func makeShebang(s string) string {
	return "#!/usr/bin/env -S " + s
}
