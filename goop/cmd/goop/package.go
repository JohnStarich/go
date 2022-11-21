package main

import (
	"fmt"
	"path/filepath"
	"strings"
)

type Package struct {
	Path          string
	Name          string
	ModuleVersion string
}

func (p Package) InstallPaths() (workingDir, installPattern string) {
	installPattern = p.Path
	if filepath.IsAbs(installPattern) {
		workingDir = installPattern
		installPattern = "."
	} else {
		version := p.ModuleVersion
		if version == "" {
			version = "latest"
		}
		installPattern += "@" + version
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
