package main

import (
	"encoding/base64"
	"fmt"
	"os/exec"
	"path"

	"github.com/johnstarich/go/pipe"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

type execPipeArgs struct {
	App     App
	Context *cli.Context
	Package Package
	Name    string
}

var execPipe = pipe.New(pipe.Options{}).
	Append(func(args []interface{}) execPipeArgs {
		return execPipeArgs{
			App:     args[1].(App),
			Context: args[0].(*cli.Context),
		}
	}).
	Append(func(args execPipeArgs) (execPipeArgs, error) {
		var err error
		args.Name, err = base64DecodeString(args.Context.String("encoded-name"))
		return args, err
	}).
	Append(func(args execPipeArgs) (execPipeArgs, string, error) {
		packagePattern, err := base64DecodeString(args.Context.String("encoded-package"))
		return args, packagePattern, err
	}).
	Append(func(args execPipeArgs, packagePattern string) (execPipeArgs, error) {
		var err error
		args.Package, err = args.App.parsePackagePattern(packagePattern)
		return args, err
	}).
	Append(func(args execPipeArgs) (execPipeArgs, string, error) {
		binaryPath, err := args.App.build(args.Context.Context, args.Name, args.Package, false)
		return args, binaryPath, err
	}).
	Append(func(args execPipeArgs, binaryPath string) (execPipeArgs, string, error) {
		binaryOSPath, err := args.App.toOSPath(binaryPath)
		return args, binaryOSPath, err
	}).
	Append(func(args execPipeArgs, binaryOSPath string) error {
		arg0 := args.Context.Args().First()
		if arg0 == "" {
			arg0 = path.Base(binaryOSPath)
		}
		argv := args.Context.Args().Tail()
		cmd := exec.CommandContext(args.Context.Context, binaryOSPath, argv...)
		cmd.Args[0] = arg0
		err := args.App.runCmd(cmd)
		return errors.WithMessage(err, formatCmd(cmd))
	})

func (a App) packageInstallDir(name string) string {
	return path.Join(a.staticCacheDir, "install", name)
}

func (a App) exec(c *cli.Context) error {
	_, err := execPipe.Do(c, a)
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
