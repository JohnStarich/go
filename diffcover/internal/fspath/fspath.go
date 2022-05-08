package fspath

import (
	"path"
	"path/filepath"
	"strings"
)

func FromFSPath(p string) string {
	// TODO fix for windows
	return filepath.FromSlash(path.Join("/", p))
}

func ToFSPath(p string) string {
	// TODO fix for windows
	return strings.TrimPrefix(filepath.ToSlash(p), "/")
}

func ToFSPathList(list string) string {
	items := filepath.SplitList(list)
	for i, item := range items {
		items[i] = ToFSPath(item)
	}
	return strings.Join(items, string(filepath.ListSeparator))
}
