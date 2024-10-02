package main

import (
	"encoding/base64"
	"fmt"
	"os/exec"
	"path"

	"github.com/johnstarich/go/pipe"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type execPipeArgs struct {
	App     App
	Cmd     *cobra.Command
	Package Package
	Name    string
}

var execPipe = pipe.New(pipe.Options{}).
	Append(func(args []interface{}) execPipeArgs {
		return execPipeArgs{
			App: args[1].(App),
			Cmd: args[0].(*cobra.Command),
		}
	}).
	Append(func(args execPipeArgs) (execPipeArgs, string, error) {
		encodedName, err := args.Cmd.Flags().GetString("encoded-name")
		return args, encodedName, err
	}).
	Append(func(args execPipeArgs, encodedName string) (execPipeArgs, error) {
		var err error
		args.Name, err = base64DecodeString(encodedName)
		return args, err
	}).
	Append(func(args execPipeArgs) (execPipeArgs, string, error) {
		encodedPackage, err := args.Cmd.Flags().GetString("encoded-package")
		return args, encodedPackage, err
	}).
	Append(func(args execPipeArgs, encodedPackage string) (execPipeArgs, string, error) {
		packagePattern, err := base64DecodeString(encodedPackage)
		return args, packagePattern, err
	}).
	Append(func(args execPipeArgs, packagePattern string) (execPipeArgs, error) {
		var err error
		args.Package, err = args.App.parsePackagePattern(packagePattern)
		return args, err
	}).
	Append(func(args execPipeArgs) (execPipeArgs, string, error) {
		binaryPath, err := args.App.build(args.Cmd.Context(), args.Name, args.Package, false)
		return args, binaryPath, err
	}).
	Append(func(args execPipeArgs, binaryPath string) (execPipeArgs, string, error) {
		binaryOSPath, err := args.App.toOSPath(binaryPath)
		return args, binaryOSPath, err
	}).
	Append(func(args execPipeArgs, binaryOSPath string) error {
		arg0, argv := popFirst(args.Cmd.Flags().Args())
		if arg0 == "" {
			arg0 = path.Base(binaryOSPath)
		}
		cmd := exec.CommandContext(args.Cmd.Context(), binaryOSPath, argv...)
		cmd.Args[0] = arg0
		err := args.App.runCmd(cmd)
		return errors.WithMessage(err, formatCmd(cmd))
	})

func popFirst(strings []string) (string, []string) {
	if len(strings) > 0 {
		return strings[0], strings[1:]
	}
	return "", nil
}

func (a App) packageInstallDir(name string) string {
	return path.Join(a.staticCacheDir, "install", name)
}

func (a App) exec(cmd *cobra.Command, _ []string) error {
	_, err := execPipe.Do(cmd, a)
	return err
}

func toEnv(envMap map[string]string) []string {
	var envKeys []string
	for key, value := range envMap {
		envKeys = append(envKeys, fmt.Sprintf("%s=%s", key, value))
	}
	return envKeys
}

func base64DecodeString(s string) (string, error) {
	b, err := base64.StdEncoding.DecodeString(s)
	return string(b), err
}
