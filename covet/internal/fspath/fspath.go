package fspath

import (
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

const separator = "/"

func FromFSPath(p string) string {
	// TODO fix for windows
	return filepath.FromSlash(path.Join(separator, p))
}

func ToFSPath(p string) string {
	// TODO fix for windows
	return strings.TrimPrefix(filepath.ToSlash(p), separator)
}

func ToFSPathList(list string) string {
	items := filepath.SplitList(list)
	for i, item := range items {
		items[i] = ToFSPath(item)
	}
	return strings.Join(items, string(filepath.ListSeparator))
}

func CommonBase(a, b string) string {
	a = path.Clean(a)
	b = path.Clean(b)
	i := 0
	for ; i < len(a) && i < len(b); i++ {
		if a[i] != b[i] {
			break
		}
	}
	return path.Clean(a[:i])
}

func Rel(basePath, targetPath string) (string, error) {
	basePath = path.Clean(basePath)
	targetPath = path.Clean(targetPath)

	common := CommonBase(basePath, targetPath)
	if common == "" {
		return "", errors.New("could not make relative path between basePath and targetPath")
	}
	base := basePath
	base = strings.TrimPrefix(base, common)
	base = strings.TrimPrefix(base, separator)
	target := targetPath
	target = strings.TrimPrefix(target, common)
	target = strings.TrimPrefix(target, separator)

	switch {
	case base == "" && target == "":
		return ".", nil
	case base == "":
		return target, nil
	default:
		p := strings.Repeat("../", strings.Count(base, separator)+1)
		return path.Join(p, target), nil
	}
}
