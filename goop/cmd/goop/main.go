package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/hack-pad/hackpadfs"
	osfs "github.com/hack-pad/hackpadfs/os"
)

func main() {
	fmt.Println("goop!")

	if len(os.Args) < 3 {
		os.Exit(1)
		return
	}
	commandName, modulePath, args := os.Args[1], os.Args[2], os.Args[3:]

	runner, modulePath, err := newApp(modulePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
		return
	}

	err = runner.Run(commandName, modulePath, args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
		return
	}
}

func newApp(modulePathArg string) (app App, modulePath string, err error) {
	modulePath = modulePathArg
	configDir, err := os.UserConfigDir()
	if err != nil {
		return
	}
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return
	}

	// convert OS paths to FS paths
	osPaths := []*string{
		&configDir,
		&cacheDir,
	}
	if filepath.IsAbs(modulePath) {
		osPaths = append(osPaths, &modulePath)
	}
	fs := osfs.NewFS()
	for _, p := range osPaths {
		*p, err = fs.FromOSPath(*p)
		if err != nil {
			return
		}
	}

	const appName = "goop"
	app = App{
		cacheDir:  path.Join(cacheDir, appName),
		configDir: path.Join(configDir, appName),
		fs:        fs,
		getenv:    os.Getenv,
		runCmd:    runCmd,
	}
	return
}

func runCmd(cmd *exec.Cmd) error {
	return cmd.Run()
}

type App struct {
	cacheDir  string
	configDir string
	fs        hackpadfs.FS
	getenv    func(string) string
	runCmd    func(*exec.Cmd) error
}

func (a App) Run(commandName, modulePath string, args []string) error {
	// TODO $PATH check
	switch commandName {
	case "install":
		return a.add(modulePath)
	case "rm":
		return nil
	case "run":
		return a.run(modulePath, args)
	default:
		return fmt.Errorf("unknown command: %q", commandName)
	}
}

type Module struct {
	Path    string
	Name    string
	Version string
}

func parseModulePath(modulePath string) (Module, error) {
	var module Module
	module.Path = modulePath
	module.Name = path.Base(modulePath)
	if i := strings.IndexRune(module.Name, '@'); i != -1 {
		module.Name, module.Version = module.Name[:i], module.Name[i+1:]
	}
	if module.Name == "" {
		return module, fmt.Errorf("module base name must not be empty: %q", modulePath)
	}
	if strings.IndexFunc(module.Name, unicode.IsSpace) != -1 {
		return module, fmt.Errorf("module names must not contain spaces: %q", module.Name)
	}
	return module, nil
}

func isBlankNotSpace(r rune) bool {
	return unicode.IsSpace(r) && r != ' '
}

func replaceAll(str string, shouldReplace func(rune) bool, replacement string) string {
	var sb strings.Builder
	for _, r := range str {
		if shouldReplace(r) {
			sb.WriteString(replacement)
		} else {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

func (a App) add(modulePath string) error {
	module, err := parseModulePath(modulePath)
	if err != nil {
		return err
	}
	scriptPath := path.Join(a.configDir, "bin", module.Name)
	if _, err := hackpadfs.Stat(a.fs, scriptPath); err == nil || !errors.Is(err, hackpadfs.ErrNotExist) {
		// stop early if script is already added or we hit an unexpected error
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

	safePath := base64.StdEncoding.EncodeToString([]byte(module.Path)) // shebangs do not support spaces or quotes, so encode it

	// Script shebang should run as follows:
	// goop run abc123== -decode-module -name ~/.config/goop/bin/foo arg1 arg2 ...
	script := fmt.Sprintf("#!/usr/bin/env goop run %s -decode-module -name\n", safePath)
	_, err = hackpadfs.WriteFile(scriptFile, []byte(script))
	return err
}

func (a App) run(modulePath string, args []string) error {
	if len(args) > 0 && args[0] == "-decode-module" {
		decodedPath, err := base64.StdEncoding.DecodeString(modulePath)
		if err != nil {
			return err
		}
		modulePath = string(decodedPath)
		args = args[1:]
	}
	arg0 := modulePath
	if len(args) > 1 && args[0] == "-name" {
		arg0 = args[1]
		args = args[2:]
	}
	fmt.Println("module:", modulePath)
	fmt.Println("arg0:  ", arg0)
	fmt.Println("argv:  ", args)
	return nil
}
