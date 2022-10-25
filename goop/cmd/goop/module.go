package main

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"unicode"
)

type Module struct {
	Path    string
	Name    string
	Version string
}

func (m Module) InstallPath() string {
	version := m.Version
	if version == "" {
		version = "latest"
	}
	return m.Path + "@" + version
}

func (a App) parseModulePathArg(modulePath string) (Module, error) {
	if filepath.IsAbs(modulePath) {
		var err error
		modulePath, err = a.fromOSPath(modulePath)
		if err != nil {
			return Module{}, err
		}
	}
	return parseModulePath(modulePath)
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
