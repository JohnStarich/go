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
