// Command watch generates docs and starts an HTTP endpoint to serve them. Also runs a file watcher on the current module to regenerate docs on change events.
//
// watch is useful for testing godoc code comments and while developing on gopages itself.
// Accepts the same flags as gopages, -gh-pages flags are ignored.
package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/johnstarich/go/gopages/cmd"
	"github.com/johnstarich/go/gopages/internal/flags"
	"github.com/johnstarich/go/gopages/internal/generate"
	"github.com/johnstarich/go/gopages/internal/pipe"
	"github.com/pkg/errors"
	"golang.org/x/mod/modfile"
)

const (
	lastUpdatedHeader = "GoPages-Last-Updated"
)

func main() {
	args, usageOutput, err := flags.Parse(os.Args[1:]...)
	switch err {
	case nil:
	case flag.ErrHelp:
		fmt.Print(usageOutput)
		return
	default:
		fmt.Print(usageOutput)
		cmd.Exit(2)
	}
	args.Watch = true
	var updatedTime string

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	fs, err := pagesFileSystem(ctx, wd, &updatedTime, args)
	if err != nil {
		panic(err)
	}
	fileServer := http.FileServer(fs)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(lastUpdatedHeader, updatedTime)

		_, err := fs.Open(r.URL.Path)
		if err == nil {
			fileServer.ServeHTTP(w, r)
			return
		}
		if !strings.HasSuffix(r.URL.Path, "/") {
			// GitHub Pages automatically looks up a corresponding .html file if it exists
			_, err := fs.Open(r.URL.Path + ".html")
			if err == nil {
				r.URL.Path += ".html"
				fileServer.ServeHTTP(w, r)
				return
			}
		}
		r.URL.Path = "/404.html"
		r.URL.RawPath = "/404.html"
		fileServer.ServeHTTP(w, r)
	})

	server := http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
	fmt.Println("Starting demo server on :8080...")
	_ = server.ListenAndServe()
}

func pagesFileSystem(ctx context.Context, modulePath string, updateTime *string, args flags.Args) (http.FileSystem, error) {
	src := osfs.New("")
	fs := memfs.New()

	goMod := filepath.Join(modulePath, "go.mod")
	var modulePackage string
	var rootedFS billy.Filesystem
	err := pipe.ChainFuncs(
		func() error {
			_, err := os.Stat(goMod)
			return pipe.ErrIf(os.IsNotExist(err), errors.New("go.mod not found in the current directory"))
		},
		func() error {
			buf, err := ioutil.ReadFile(goMod)
			modulePackage = modfile.ModulePath(buf)
			return err
		},
		func() error {
			return pipe.ErrIf(modulePackage == "", errors.Errorf("Unable to find module package name in go.mod file: %s", goMod))
		},
		func() error {
			return watch(ctx, modulePath, func() error {
				err := generate.Docs(modulePath, modulePackage, src, fs, args)
				*updateTime = time.Now().Format(time.RFC3339)
				return err
			})
		},
		func() error {
			var err error
			rootedFS, err = fs.Chroot(args.OutputPath)
			return err
		},
	).Do()

	return &httpFSWrapper{base: args.BaseURL, Filesystem: rootedFS}, err
}

type httpFSWrapper struct {
	base string
	billy.Filesystem
}

func (h *httpFSWrapper) Open(name string) (http.File, error) {
	if !strings.HasPrefix(name, h.base) {
		return nil, os.ErrNotExist
	}
	name = strings.TrimPrefix(name, h.base)
	name = filepath.FromSlash(name)
	info, err := h.Filesystem.Stat(name)
	if err != nil {
		return nil, err
	}

	var file billy.File
	// memfs.Open doesn't work for directories, so create a false dir for those instead
	if info.IsDir() {
		file, err = memfs.New().Create(name)
	} else {
		file, err = h.Filesystem.Open(name)
	}
	return &httpFileWrapper{
		File: file,
		fs:   h.Filesystem,
		name: name,
	}, err
}

type httpFileWrapper struct {
	billy.File

	name string
	fs   billy.Filesystem
}

func (h *httpFileWrapper) Readdir(count int) ([]os.FileInfo, error) {
	return h.fs.ReadDir(h.name)
}

func (h *httpFileWrapper) Stat() (os.FileInfo, error) {
	return h.fs.Stat(h.name)
}
