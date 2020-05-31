package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

func watch(ctx context.Context, path string, do func() error) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
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
