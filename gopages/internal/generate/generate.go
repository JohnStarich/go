package generate

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/util"
	"github.com/johnstarich/go/gopages/internal/flags"
	"github.com/johnstarich/go/gopages/internal/pipe"
	"github.com/johnstarich/go/gopages/internal/safememfs"
	"github.com/pkg/errors"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/godoc"
	"golang.org/x/tools/godoc/static"
	"golang.org/x/tools/godoc/vfs"
	"golang.org/x/tools/godoc/vfs/mapfs"
)

func Docs(modulePath, modulePackage string, src, fs billy.Filesystem, args flags.Args) error {
	var ns vfs.NameSpace
	var srcRoot billy.Filesystem
	var corpus *godoc.Corpus
	err := pipe.ChainFuncs(
		func() error {
			return errors.Wrap(util.RemoveAll(fs, args.OutputPath), "Failed to clean output directory")
		},
		func() error {
			return errors.Wrap(fs.MkdirAll(args.OutputPath, 0700), "Failed to create output directory")
		},
		func() error {
			ns = vfs.NewNameSpace()
			ns.Bind("/lib/godoc", mapfs.New(static.Files), "/", vfs.BindReplace)
			var err error
			srcRoot, err = src.Chroot(modulePath)
			return errors.Wrapf(err, "Failed to chroot the source file system to %q", modulePath)
		},
		func() error {
			modFS := &filesystemOpener{Filesystem: srcRoot}
			ns.Bind(path.Join("/src", modulePackage), modFS, "/", vfs.BindReplace)
			parentDirectoriesFS := &filesystemOpener{Filesystem: makePath("src/" + modulePackage)} // create empty directories for outside module
			ns.Bind("/", parentDirectoriesFS, "/", vfs.BindAfter)

			corpus = godoc.NewCorpus(ns)
			return errors.Wrap(corpus.Init(), "Are there any Go files present? Failed to initialize corpus")
		},
	).Do()
	if err != nil {
		return err
	}

	pres := godoc.NewPresentation(corpus)
	// attempt to override URLs for source code links
	// TODO fix links from source pages back to docs
	pres.URLForSrc = func(src string) string {
		// seems godoc lib documentation is incorrect here, 'src' is actually the whole package path to the file
		return path.Join(args.BaseURL, "/src", src)
	}
	pres.URLForSrcPos = func(src string, line, low, high int) string {
		return (&url.URL{
			Path:     path.Join(args.BaseURL, src),
			Fragment: fmt.Sprintf("L%d", line),
		}).String()
	}
	pres.URLForSrcQuery = func(src, query string, line int) string {
		return (&url.URL{
			Path:     path.Join(args.BaseURL, src),
			RawQuery: query,
			Fragment: fmt.Sprintf("L%d", line),
		}).String()
	}
	funcs := pres.FuncMap()
	addGoPagesFuncs(funcs, args)
	readTemplates(pres, funcs, ns)

	// Generate all static assets and save to /lib/godoc
	var ops []pipe.OpFunc
	for name := range static.Files {
		content := static.Files[name]
		path := filepath.Join(args.OutputPath, "lib", "godoc", name)
		ops = append(ops, func() error {
			return fs.MkdirAll(filepath.Dir(path), 0700)
		}, func() error {
			return util.WriteFile(fs, path, []byte(content), 0600)
		})
	}
	err = pipe.ChainFuncs(ops...).Do()
	if err != nil {
		return err
	}

	var packagePaths []string
	var custom404 []byte
	return pipe.ChainFuncs(
		func() error {
			// Generate main index to redirect to actual content page. Important to separate from 'lib' top-level dir.
			return util.WriteFile(fs, filepath.Join(args.OutputPath, "index.html"), []byte(redirect("pkg/"+modulePackage)), 0600)
		},
		func() error {
			// Generate a custom 404 page as a catch-all
			var err error
			custom404, err = genericPage(pres, "Page not found", `
<p>
<span class="alert" style="font-size:120%">
Oops, this page doesn't exist.
</span>
</p>
<p>If something should be here, <a href="https://github.com/JohnStarich/go/issues/new" target="_blank">open an issue</a>.</p>
`)
			return err
		},
		func() error {
			return util.WriteFile(fs, filepath.Join(args.OutputPath, "404.html"), custom404, 0600)
		},
		func() error {
			// For each package, generate an index page
			var err error
			packagePaths, err = getPackagePaths(modulePackage)
			return err
		},
		func() error {
			var ops []pipe.OpFunc
			for i := range packagePaths {
				path := packagePaths[i]
				ops = append(ops, func() error {
					return writePackageIndex(fs, pres, path, args.OutputPath)
				})
			}
			return pipe.ChainFuncs(ops...).Do()
		},
		func() error {
			// For each directory in and above above the module, generate a src and pkg page
			var ops []pipe.OpFunc
			var base string // collect the last base path (should always be the same after a loop iteration
			for _, packagePath := range packagePaths {
				for base = ""; packagePath != ""; packagePath, base = path.Split(packagePath) {
					packagePath = path.Clean(packagePath)
					p, b := packagePath, base // local scope copies
					ops = append(ops,
						func() error {
							return writePackageIndex(fs, pres, p, args.OutputPath)
						},
						func() error {
							return writeSourceFile(fs, pres, args.BaseURL, p, true, b, args.OutputPath)
						},
					)
				}
			}
			// and run again on the last path segment (e.g. github.com)
			ops = append(ops,
				func() error {
					return writeSourceFile(fs, pres, args.BaseURL, "", true, base, args.OutputPath)
				},
				func() error {
					return writePackageIndex(fs, pres, base, args.OutputPath)
				},
			)
			return pipe.ChainFuncs(ops...).Do()
		},
		func() error {
			// For each source file and directory inside the module, generate a src page
			return walkFiles(srcRoot, "/", func(file string, isDir bool) error {
				dir, base := filepath.Split(file)
				if strings.HasPrefix(base, ".") {
					// skip dot files, e.g. '.git'
					return pipe.ErrIf(isDir, filepath.SkipDir)
				}
				if isDir && args.OutputPath != "" && strings.TrimPrefix(file, "/") == args.OutputPath {
					// skip the destination directory if it's set to avoid infinite recursion
					return filepath.SkipDir
				}
				if !(isDir || filepath.Ext(file) == ".go") {
					// only scrape directories and Go files
					return nil
				}
				packagePath := path.Join(modulePackage, dir)
				return writeSourceFile(fs, pres, args.BaseURL, packagePath, isDir, base, args.OutputPath)
			})
		},
		func() error {
			// Generate root package index, displaying all packages
			return writePackageIndex(fs, pres, "", args.OutputPath)
		},
		func() error {
			// Generate root src index, displaying all top level files
			return writeSourceFile(fs, pres, args.BaseURL, "", true, "", args.OutputPath)
		},
	).Do()
}

func doRequest(do func(w http.ResponseWriter)) ([]byte, error) {
	recorder := httptest.NewRecorder()
	do(recorder)
	return recorder.Body.Bytes(), pipe.ErrIf(
		recorder.Result().StatusCode != http.StatusOK,
		errors.Errorf("Error generating page: [%d]\n%s\n%s", recorder.Result().StatusCode, recorder.Header(), recorder.Body.String()),
	)
}

func getPage(pres *godoc.Presentation, path string) ([]byte, error) {
	u, err := url.Parse(path)
	if err != nil {
		panic(err)
	}
	return doRequest(func(w http.ResponseWriter) {
		pres.ServeHTTP(w, &http.Request{URL: u})
	})
}

func genericPage(pres *godoc.Presentation, title, body string) ([]byte, error) {
	return doRequest(func(w http.ResponseWriter) {
		pres.ServePage(w, godoc.Page{
			Title:    title,
			Tabtitle: title,
			Body:     []byte(body),
		})
	})
}

func pathSplit(path string) []string {
	return strings.Split(path, "/")
}

func writePackageIndex(fs billy.Filesystem, pres *godoc.Presentation, packagePath, outputBasePath string) error {
	outputComponents := append([]string{outputBasePath, "pkg"}, pathSplit(packagePath)...)
	outputComponents = append(outputComponents, "index.html")
	outputPath := filepath.Join(outputComponents...)

	var page []byte
	return pipe.ChainFuncs(
		func() error {
			var err error
			page, err = getPage(pres, path.Join("/pkg", packagePath)+"/?m=all") // show index pages for internal packages
			return err
		},
		func() error {
			return fs.MkdirAll(filepath.Dir(outputPath), 0700)
		},
		func() error {
			return util.WriteFile(fs, outputPath, page, 0600)
		},
	).Do()
}

func writeSourceFile(fs billy.Filesystem, pres *godoc.Presentation, baseURL, packagePath string, isDir bool, fileName, outputBasePath string) error {
	outputComponents := append([]string{outputBasePath, "src"}, pathSplit(packagePath)...)
	outputComponents = append(outputComponents, fileName)
	var pathSuffix string
	if isDir {
		outputComponents = append(outputComponents, "index")
		pathSuffix = "/"
	}
	outputPath := filepath.Join(outputComponents...) + ".html"

	var page []byte
	return pipe.ChainFuncs(
		func() error {
			var err error
			page, err = getPage(pres, path.Join("/src", packagePath, fileName)+pathSuffix)
			return err
		},
		func() error {
			var err error
			page, err = customizeSourceCodePage(baseURL, page)
			return err
		},
		func() error {
			return fs.MkdirAll(filepath.Dir(outputPath), 0700)
		},
		func() error {
			return util.WriteFile(fs, outputPath, page, 0600)
		},
	).Do()
}

func getPackagePaths(modulePackage string) ([]string, error) {
	pkgs, err := packages.Load(&packages.Config{
		Mode: packages.NeedName,
	}, modulePackage+"/...")

	paths := make([]string, len(pkgs))
	for i, pkg := range pkgs {
		paths[i] = pkg.PkgPath
	}
	return paths, err
}

func redirect(url string) string {
	var buf bytes.Buffer
	err := template.Must(template.New("").Parse(`<!DOCTYPE html>
<html>
<head>
<script>
window.location = {{.URL}}
</script>
</head>
<body>
	<a href={{.URL}}>Click here to see this module's documentation.</a>
</body>
</html>
`)).Execute(&buf, map[string]interface{}{
		"URL": fmt.Sprintf("%q", url),
	})
	if err != nil {
		panic(err)
	}
	return buf.String()
}

type filesystemOpener struct {
	billy.Filesystem
}

func (f *filesystemOpener) Open(name string) (vfs.ReadSeekCloser, error) {
	return f.OpenFile(name, 0, 0)
}

func (f *filesystemOpener) RootType(path string) vfs.RootType {
	return ""
}

func (f *filesystemOpener) String() string {
	return "*filesystemOpener"
}

func walkFiles(fs billy.Filesystem, path string, visit func(path string, isDir bool) error) error {
	err := walkFilesFn(fs, path, visit)
	return pipe.ErrIf(err != filepath.SkipDir, err)
}

func walkFilesFn(fs billy.Filesystem, path string, visit func(path string, isDir bool) error) error {
	info, err := fs.Lstat(path)
	if err != nil {
		return errors.Wrap(err, "Error looking up file")
	}

	if info.Mode()&os.ModeSymlink == os.ModeSymlink {
		// ignore symlinks to avoid infinite recursion
		return nil
	}

	isDir := info.IsDir()
	if err := visit(path, isDir); err != nil {
		return err
	}

	if !isDir {
		return nil
	}

	dir, err := fs.ReadDir(path)
	if err != nil {
		return errors.Wrapf(err, "Error reading directory %q", path)
	}
	for _, info = range dir {
		err := walkFilesFn(fs, filepath.Join(path, info.Name()), visit)
		if err == filepath.SkipDir {
			if !info.IsDir() {
				break // for SkipDir on a file, skip remaining files in directory
			}
			// otherwise continue recursing other files in this dir
		} else if err != nil {
			return err
		}
	}
	return nil
}

func makePath(path string) billy.Filesystem {
	fs := safememfs.New()
	err := fs.MkdirAll(path, 0600)
	if err != nil {
		panic(err)
	}
	return fs
}
