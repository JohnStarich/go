//go:build !windows
// +build !windows

package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
)

const maxOpenFiles = 1 << 14 // 2^14 appears to be the maximum value for macOS and 2^20 on linux

func watch(ctx context.Context, path string, do func() error) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	// increase soft open file limit to maximum
	var rLimit syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
		return errors.Wrap(err, "Failed to get open file limit")
	}
	if rLimit.Cur < rLimit.Max && rLimit.Cur < maxOpenFiles {
		rLimit.Cur = min(rLimit.Max, maxOpenFiles)
		err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit)
		if err != nil {
			return errors.Wrap(err, "Failed to increase soft open file limit")
		}
	}

	err = filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return watcher.Add(path)
		}
		return nil
	})
	if err != nil {
		watcher.Close()
		return err
	}

	go func() {
		timer := time.NewTimer(0) // fire watch right away
		defer timer.Stop()

		const debounce = 2 * time.Second
		for {
			select {
			case <-timer.C:
				log.Println("Running watch call...")
				err := do()
				if err != nil {
					log.Println("Error running watch call:", err)
				}
			case <-ctx.Done():
				watcher.Close()
				return
			case event := <-watcher.Events:
				switch {
				case event.Op&fsnotify.Write == fsnotify.Write,
					event.Op&fsnotify.Create == fsnotify.Create:
					timer.Reset(debounce)
				}
			case err := <-watcher.Errors:
				log.Println("error:", err)
			}
		}
	}()
	return nil
}
