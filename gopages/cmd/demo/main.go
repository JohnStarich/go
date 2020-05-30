package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/johnstarich/go/gopages/internal/flags"
	"github.com/johnstarich/go/gopages/internal/generate"
	"github.com/johnstarich/go/gopages/internal/pipe"
	"github.com/pkg/errors"
	"golang.org/x/mod/modfile"
)

func main() {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	fs, err := pagesFileSystem(wd)
	if err != nil {
		panic(err)
	}
	fileServer := http.FileServer(fs)
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, err := fs.Open(r.URL.Path)
		if err == nil {
			fileServer.ServeHTTP(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, ".go") {
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

func pagesFileSystem(modulePath string) (http.FileSystem, error) {
	src := osfs.New("")
	fs := memfs.New()

	goMod := filepath.Join(modulePath, "go.mod")
	var modulePackage string
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
			return generate.Docs(modulePath, modulePackage, src, fs, flags.Args{})
		},
	).Do()
	return &httpFSWrapper{fs}, err
}

type httpFSWrapper struct {
	billy.Filesystem
}

func (h *httpFSWrapper) Open(name string) (http.File, error) {
	info, err := h.Filesystem.Stat(name)
	if err != nil {
		return nil, err
	}

	var file billy.File
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
