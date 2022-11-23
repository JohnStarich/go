package main

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Package represents a package pattern used in the install and exec commands. Example: github.com/johnstarich/go/goop/cmd/goop@latest
type Package struct {
	Path          string
	Name          string
	ModuleVersion string
}

const homeDir = "~"

func (a App) parsePackagePattern(packagePattern string) (Package, error) {
	pkg := Package{
		Path: packagePattern,
	}
	if i := strings.IndexRune(pkg.Path, '@'); i != -1 {
		pkg.Path, pkg.ModuleVersion = pkg.Path[:i], pkg.Path[i+1:]
	}
	if strings.HasSuffix(pkg.Path, "/...") {
		return pkg, fmt.Errorf("package pattern must not use the '/...' operator: %q", pkg.Path)
	}
	if strings.HasPrefix(pkg.Path, a.staticOSHomeDir+string(filepath.Separator)) {
		// Canonicalize home directory to '~' for better cross-machine bin support, as home directories can change from user to user.
		pkg.Path = homeDir + strings.TrimPrefix(pkg.Path, a.staticOSHomeDir)
	}

	pkg.Name = pkg.Path
	if i := strings.LastIndexAny(pkg.Path, `\/`); i != -1 {
		pkg.Name = pkg.Path[i+1:]
	}
	return pkg, nil
}

// packageFilePath returns pkg's local file path and true if it is a local module.
// Empty string and false otherwise.
func (a App) packageFilePath(pkg Package) (string, bool) {
	filePath := pkg.Path
	if strings.HasPrefix(filePath, homeDir+string(filepath.Separator)) {
		// expand home directory '~' to full file path
		filePath = a.staticOSHomeDir + strings.TrimPrefix(filePath, homeDir)
	}
	if filepath.IsAbs(filePath) {
		return filePath, true
	}
	return "", false
}

// packageInstallPaths returns this module's "go install ..." args and working directory.
func (a App) packageInstallPaths(pkg Package) (workingDir, installPattern string) {
	if filePath, ok := a.packageFilePath(pkg); ok {
		workingDir = filePath
		installPattern = "."
	} else {
		version := pkg.ModuleVersion
		if version == "" {
			version = "latest"
		}
		installPattern = pkg.Path + "@" + version
	}
	return
}
