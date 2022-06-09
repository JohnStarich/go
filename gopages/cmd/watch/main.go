// Command watch generates docs and starts an HTTP endpoint to serve them. Also runs a file watcher on the current module to regenerate docs on change events.
//
// watch is useful for testing godoc code comments and while developing on gopages itself.
// Accepts the same flags as gopages, -gh-pages flags are ignored.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
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
	"github.com/johnstarich/go/gopages/internal/generate/source"
	"github.com/johnstarich/go/gopages/internal/module"
	"github.com/johnstarich/go/pipe"
)

const (
	lastUpdatedHeader = "GoPages-Last-Updated"
)

func main() {
	args, usageOutput, err := flags.Parse(os.Args[1:]...)
	switch {
	case err == nil:
	case errors.Is(err, flag.ErrHelp):
		fmt.Print(usageOutput)
		return
	default:
		fmt.Print(usageOutput)
		cmd.Exit(cmd.ExitCodeInvalidUsage)
	}
	args.Watch = true
	var updatedTime string

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	modulePackage, err := module.Package(wd)
	if err != nil {
		panic(err)
	}
	linker, err := args.Linker(modulePackage)
	if err != nil {
		panic(err)
	}
	fs, err := pagesFileSystem(ctx, wd, &updatedTime, args, linker)
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

type pagesFSArgs struct {
	flags.Args
	Ctx        context.Context
	ModulePath string
	UpdateTime *string
	Linker     source.Linker
}

var pagesFSPipe = pipe.New(pipe.Options{}).
	Append(func(args []interface{}) pagesFSArgs {
		return args[0].(pagesFSArgs)
	}).
	Append(func(args pagesFSArgs) (pagesFSArgs, string, error) {
		modulePackage, err := module.Package(args.ModulePath)
		return args, modulePackage, err
	}).
	Append(func(args pagesFSArgs, modulePackage string) (pagesFSArgs, billy.Filesystem, error) {
		src := osfs.New("")
		fs := memfs.New()
		err := watch(args.Ctx, args.ModulePath, func() error {
			err := generate.Docs(args.ModulePath, modulePackage, src, fs, args.Args, args.Linker)
			*args.UpdateTime = time.Now().Format(time.RFC3339)
			return err
		})
		return args, fs, err
	}).
	Append(func(args pagesFSArgs, fs billy.Filesystem) (http.FileSystem, error) {
		rootedFS, err := fs.Chroot(args.OutputPath)
		return &httpFSWrapper{base: args.BaseURL, Filesystem: rootedFS}, err
	})

func pagesFileSystem(ctx context.Context, modulePath string, updateTime *string, args flags.Args, linker source.Linker) (http.FileSystem, error) {
	out, err := pagesFSPipe.Do(pagesFSArgs{
		Args:       args,
		Ctx:        ctx,
		ModulePath: modulePath,
		UpdateTime: updateTime,
		Linker:     linker,
	})
	var fs http.FileSystem
	if err == nil {
		fs = out[0].(http.FileSystem)
	}
	return fs, err
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
