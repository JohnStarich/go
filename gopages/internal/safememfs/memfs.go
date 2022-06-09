// Package safememfs wraps go-billy/memfs's Open to correctly handle opening directories
package safememfs

import (
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
)

// New creates a new billy/memfs file system that can handle opening directories
func New() *SafeOpener {
	fs := memfs.New()
	return &SafeOpener{fs}
}

// SafeOpener is a billy.Filesystem that fixes Open() behavior on directories with a work-around
type SafeOpener struct {
	billy.Filesystem
}

// Open reimplements memfs.FS.Open() to fix bad behavior on opening directories
func (s *SafeOpener) Open(name string) (billy.File, error) {
	info, err := s.Filesystem.Stat(name)
	if err != nil {
		return nil, err
	}

	var file billy.File
	// memfs.Open doesn't work for directories, so create a false dir for those instead
	if info.IsDir() {
		file, err = memfs.New().Create(name)
	} else {
		file, err = s.Filesystem.Open(name)
	}
	return file, err
}
