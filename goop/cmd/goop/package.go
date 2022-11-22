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

// FilePath returns this package's local file path and true if it is a local module.
// Empty string and false otherwise.
func (p Package) FilePath() (string, bool) {
	if filepath.IsAbs(p.Path) {
		return p.Path, true
	}
	return "", false
}

// InstallPaths returns this module's "go install ..." args and working directory.
func (p Package) InstallPaths() (workingDir, installPattern string) {
	if filePath, ok := p.FilePath(); ok {
		workingDir = filePath
		installPattern = "."
	} else {
		version := p.ModuleVersion
		if version == "" {
			version = "latest"
		}
		installPattern = p.Path + "@" + version
	}
	return
}

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

	pkg.Name = pkg.Path
	if i := strings.LastIndexAny(pkg.Path, `\/`); i != -1 {
		pkg.Name = pkg.Path[i+1:]
	}
	return pkg, nil
}
