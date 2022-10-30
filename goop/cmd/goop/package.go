package main

import (
	"fmt"
	"strings"
	"unicode"
)

type Package struct {
	Path          string
	Name          string
	ModuleVersion string
}

func (p Package) InstallPath() string {
	path := p.Path
	if p.ModuleVersion != "" {
		path += "@" + p.ModuleVersion
	}
	return path
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
