// Package generate generates documentation pages for a given package.
package generate

import (
	"bytes"
	_ "embed"
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
	"github.com/johnstarich/go/gopages/internal/generate/source"
	"github.com/johnstarich/go/gopages/internal/safememfs"
	"github.com/johnstarich/go/pipe"
	"github.com/pkg/errors"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/godoc"
	"golang.org/x/tools/godoc/static"
	"golang.org/x/tools/godoc/vfs"
	"golang.org/x/tools/godoc/vfs/mapfs"
)

const (
	scrapeDirPermission  = 0700
	scrapeFilePermission = 0600
)

var (
	//go:embed redirect.html
	redirectHTML string
	//go:embed not-found.html
	notFoundHTML string
)

type docsArgs struct {
	flags.Args
	ModulePath    string
	ModulePackage string
	Src, FS       billy.Filesystem
	Linker        source.Linker
	Presentation  *godoc.Presentation // set after first makePresentationPipe pipe call
	SrcRoot       billy.Filesystem    // set after first makePresentationPipe pipe call
}

var makePresentationPipe = pipe.New(pipe.Options{}).
	Append(func(args []interface{}) docsArgs {
		dArgs := args[0].(docsArgs)
		return dArgs
	}).
	Append(func(args docsArgs) (docsArgs, error) {
		return args, errors.Wrap(util.RemoveAll(args.FS, args.OutputPath), "Failed to clean output directory")
	}).
	Append(func(args docsArgs) (docsArgs, error) {
		return args, errors.Wrap(args.FS.MkdirAll(args.OutputPath, scrapeDirPermission), "Failed to create output directory")
	}).
	Append(func(args docsArgs) (docsArgs, vfs.NameSpace, error) {
		ns := vfs.NewNameSpace()
		ns.Bind("/lib/godoc", mapfs.New(static.Files), "/", vfs.BindReplace)
		srcRoot, err := args.Src.Chroot(args.ModulePath)
		args.SrcRoot = srcRoot
		return args, ns, errors.Wrapf(err, "Failed to chroot the source file system to %q", args.ModulePath)
	}).
	Append(func(args docsArgs, ns vfs.NameSpace) (docsArgs, vfs.NameSpace, *godoc.Corpus, error) {
		modFS := &filesystemOpener{Filesystem: args.SrcRoot}
		ns.Bind(path.Join("/src", args.ModulePackage), modFS, "/", vfs.BindReplace)
		parentDirectoriesFS := &filesystemOpener{Filesystem: makePath("src/" + args.ModulePackage)} // create empty directories for outside module
		ns.Bind("/", parentDirectoriesFS, "/", vfs.BindAfter)
		corpus := godoc.NewCorpus(ns)
		return args, ns, corpus, errors.Wrap(corpus.Init(), "Are there any Go files present? Failed to initialize corpus")
	}).
	Append(func(args docsArgs, ns vfs.NameSpace, corpus *godoc.Corpus) docsArgs {
		pres := godoc.NewPresentation(corpus)
		pres.AdjustPageInfoMode = func(req *http.Request, mode godoc.PageInfoMode) godoc.PageInfoMode {
			switch {
			case req.URL.Path == "/pkg/", strings.HasPrefix(req.URL.Path, "/pkg/") && strings.HasSuffix(req.URL.Path, "/internal/"):
				mode |= godoc.NoFiltering
			}
			return mode
		}
		// attempt to override URLs for source code links
		// TODO fix links from source pages back to docs
		pres.URLForSrc = func(src string) string {
			// seems godoc lib documentation is incorrect here, 'src' is actually the whole package path to the file
			src = strings.TrimPrefix(src, "/")
			u := args.Linker.LinkToSource(src, source.LinkOptions{})
			return u.String()
		}
		pres.URLForSrcPos = func(src string, line, low, high int) string {
			src = strings.TrimPrefix(src, "/src/")
			u := args.Linker.LinkToSource(src, source.LinkOptions{
				Line: line,
			})
			return u.String()
		}
		pres.URLForSrcQuery = func(src, query string, line int) string {
			src = strings.TrimPrefix(src, "/src/")
			u := args.Linker.LinkToSource(src, source.LinkOptions{
				Line: line,
			})
			return u.String()
		}
		funcs := pres.FuncMap()
		addGoPagesFuncs(funcs, args.ModulePackage, args.Args)
		readTemplates(pres, funcs, ns)
		args.Presentation = pres
		return args
	})

var genStaticPipe = pipe.New(pipe.Options{}).
	Append(func(args []interface{}) (billy.Filesystem, string, string) {
		fs := args[0].(billy.Filesystem)
		outputPath := args[1].(string)
		name := args[2].(string)
		return fs, outputPath, name
	}).
	Append(func(fs billy.Filesystem, outputPath, name string) (billy.Filesystem, string, string, error) {
		path := filepath.Join(outputPath, "lib", "godoc", name)
		content := static.Files[name]
		return fs, path, content, fs.MkdirAll(filepath.Dir(path), scrapeDirPermission)
	}).
	Append(func(fs billy.Filesystem, path, content string) error {
		return util.WriteFile(fs, path, []byte(content), scrapeFilePermission)
	})

var crawlerStaticPipe = pipe.New(pipe.Options{}).
	Append(func(args []interface{}) docsArgs {
		return args[0].(docsArgs)
	}).
	Append(func(args docsArgs) (docsArgs, error) {
		// Generate main index to redirect to actual content page. Important to separate from 'lib' top-level dir.
		return args, util.WriteFile(args.FS, filepath.Join(args.OutputPath, "index.html"), []byte(redirect("pkg/"+args.ModulePackage)), scrapeFilePermission)
	}).
	Append(func(args docsArgs) (docsArgs, []byte, error) {
		// Generate a custom 404 page as a catch-all
		custom404, err := genericPage(args.Presentation, "Page not found", notFoundHTML)
		return args, custom404, err
	}).
	Append(func(args docsArgs, custom404 []byte) (docsArgs, error) {
		return args, util.WriteFile(args.FS, filepath.Join(args.OutputPath, "404.html"), custom404, scrapeFilePermission)
	})

var crawlerPipe = pipe.New(pipe.Options{}).
	Append(func(args []interface{}) docsArgs {
		return args[0].(docsArgs)
	}).
	Append(func(args docsArgs) (docsArgs, []string, error) {
		packagePaths, err := getPackagePaths(args.ModulePackage)
		return args, packagePaths, err
	})

var packageGenPipe = pipe.New(pipe.Options{}).
	Append(func(args []interface{}) (docsArgs, string, string) {
		dArgs := args[0].(docsArgs)
		packagePath := args[1].(string)
		base := args[2].(string)
		return dArgs, packagePath, base
	}).
	Append(func(args docsArgs, packagePath, base string) (docsArgs, string, string, error) {
		return args, packagePath, base, writePackageIndex(args.FS, args.Presentation, packagePath, args.OutputPath)
	}).
	Append(func(args docsArgs, packagePath, base string) (docsArgs, string, string, error) {
		return args, packagePath, base, writeSourceFile(args.FS, args.Presentation, args.BaseURL, packagePath, true, base, args.OutputPath, args.Linker)
	}).
	Append(func(args docsArgs, packagePath, base string) error {
		return writeSourceFile(args.FS, args.Presentation, args.BaseURL, packagePath, true, base, args.OutputPath, args.Linker)
	})

var (
	errSkipFile  = fmt.Errorf("skip this file")
	walkFilePipe = pipe.New(pipe.Options{}).
			Append(func(args []interface{}) (docsArgs, string, bool) {
			dArgs := args[0].(docsArgs)
			file := args[1].(string)
			isDir := args[2].(bool)
			return dArgs, file, isDir
		}).
		Append(func(args docsArgs, file string, isDir bool) (docsArgs, bool, string, bool, error) {
			// skip dot dirs, e.g. '.git'
			shouldHide := strings.HasPrefix(filepath.Base(file), ".")
			return args, shouldHide, file, isDir, pipe.CheckError(shouldHide && isDir, filepath.SkipDir)
		}).
		Append(func(args docsArgs, shouldHide bool, file string, isDir bool) (docsArgs, string, bool, error) {
			// skip dot files, e.g. '.gitignore'
			return args, file, isDir, pipe.CheckError(shouldHide, errSkipFile)
		}).
		Append(func(args docsArgs, file string, isDir bool) (docsArgs, string, bool, error) {
			// skip the destination directory if it's set to avoid infinite recursion
			return args, file, isDir, pipe.CheckError(isDir && args.OutputPath != "" && strings.TrimPrefix(file, "/") == args.OutputPath, filepath.SkipDir)
		}).
		Append(func(args docsArgs, file string, isDir bool) (docsArgs, string, bool, error) {
			// only scrape directories and Go files
			scrapable := isDir || filepath.Ext(file) == ".go"
			return args, file, isDir, pipe.CheckError(!scrapable, errSkipFile)
		}).
		Append(func(args docsArgs, file string, isDir bool) error {
			dir, base := filepath.Split(file)
			packagePath := path.Join(args.ModulePackage, dir)
			return writeSourceFile(args.FS, args.Presentation, args.BaseURL, packagePath, isDir, base, args.OutputPath, args.Linker)
		})
)

var packageCrawlerPipe = pipe.New(pipe.Options{}).
	Append(func(args []interface{}) (docsArgs, string) {
		dArgs := args[0].(docsArgs)
		path := args[1].(string)
		return dArgs, path
	}).
	Append(func(args docsArgs, path string) (docsArgs, string, error) {
		return args, path, writePackageIndex(args.FS, args.Presentation, path, args.OutputPath)
	}).
	Append(func(args docsArgs, packagePath string) error {
		multiArgs := make([][]interface{}, 0, strings.Count(packagePath, "/")+1)
		var base string // collect the last base path (should always be the same after a loop iteration
		for base = ""; packagePath != ""; packagePath, base = path.Split(packagePath) {
			packagePath = path.Clean(packagePath)
			multiArgs = append(multiArgs, []interface{}{args, packagePath, base})
		}
		multiArgs = append(multiArgs, []interface{}{args, "", base})
		_, err := pipe.Map(packageGenPipe, multiArgs)
		return err
	})

var crawlerWalkPipe = pipe.New(pipe.Options{}).
	Append(func(args []interface{}) docsArgs {
		return args[0].(docsArgs)
	}).
	Append(func(args docsArgs) (docsArgs, error) {
		// For each source file and directory inside the module, generate a src page
		return args, walkFiles(args.SrcRoot, "/", func(file string, isDir bool) error {
			_, err := walkFilePipe.Do(args, file, isDir)
			err = errors.Unwrap(err) // remove pipe wrapper
			return pipe.CheckError(!errors.Is(err, errSkipFile), err)
		})
	}).
	Append(func(args docsArgs) (docsArgs, error) {
		// Generate root package index, displaying all packages
		return args, writePackageIndex(args.FS, args.Presentation, "", args.OutputPath)
	}).
	Append(func(args docsArgs) error {
		// Generate root src index, displaying all top level files
		return writeSourceFile(args.FS, args.Presentation, args.BaseURL, "", true, "", args.OutputPath, args.Linker)
	})

// Docs generates documentation pages for the given package
func Docs(modulePath, modulePackage string, src, fs billy.Filesystem, args flags.Args, linker source.Linker) error {
	dArgs := docsArgs{
		Args:          args,
		ModulePath:    modulePath,
		ModulePackage: modulePackage,
		Src:           src,
		FS:            fs,
		Linker:        linker,
	}
	docsPipe := pipe.New(pipe.Options{}).
		Concat(makePresentationPipe).
		Append(func(args docsArgs) (docsArgs, error) {
			// Generate all static assets and save to /lib/godoc
			var multiArgs [][]interface{}
			for name := range static.Files {
				multiArgs = append(multiArgs, []interface{}{
					fs, args.OutputPath, name,
				})
			}
			_, err := pipe.Map(genStaticPipe, multiArgs)
			return args, err
		}).
		Concat(crawlerStaticPipe).
		Concat(crawlerPipe).
		Append(func(args docsArgs, packagePaths []string) (docsArgs, error) {
			multiArgs := make([][]interface{}, len(packagePaths))
			for i, p := range packagePaths {
				multiArgs[i] = []interface{}{args, p}
			}
			_, err := pipe.Map(packageCrawlerPipe, multiArgs)
			return args, err
		}).
		Concat(crawlerWalkPipe)
	_, err := docsPipe.Do(dArgs)
	return err
}

func doRequest(do func(w http.ResponseWriter)) ([]byte, error) {
	recorder := httptest.NewRecorder()
	do(recorder)
	result := recorder.Result()
	defer result.Body.Close()
	return recorder.Body.Bytes(), pipe.CheckErrorf(
		result.StatusCode != http.StatusOK,
		"Error generating page: [%d]\n%s\n%s", result.StatusCode, recorder.Header(), recorder.Body.String(),
	)
}

func getPage(pres *godoc.Presentation, path string) ([]byte, error) {
	u := &url.URL{Path: path}
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

var writePackageIndexPipe = pipe.New(pipe.Options{}).
	Append(func(args []interface{}) (billy.Filesystem, *godoc.Presentation, string, string) {
		fs := args[0].(billy.Filesystem)
		pres := args[1].(*godoc.Presentation)
		packagePath := args[2].(string)
		outputBasePath := args[3].(string)
		return fs, pres, packagePath, outputBasePath
	}).
	Append(func(fs billy.Filesystem, pres *godoc.Presentation, packagePath, outputBasePath string) (billy.Filesystem, string, string, []byte, error) {
		p := path.Join("/pkg", packagePath)
		p = pagePath(true, p)
		page, err := getPage(pres, p)
		return fs, packagePath, outputBasePath, page, err
	}).
	Append(func(fs billy.Filesystem, packagePath, outputBasePath string, page []byte) (billy.Filesystem, string, []byte, error) {
		outputComponents := append([]string{outputBasePath, "pkg"}, pathSplit(packagePath)...)
		outputComponents = append(outputComponents, "index.html")
		outputPath := filepath.Join(outputComponents...)
		return fs, outputPath, page, fs.MkdirAll(filepath.Dir(outputPath), scrapeDirPermission)
	}).
	Append(func(fs billy.Filesystem, outputPath string, page []byte) error {
		return util.WriteFile(fs, outputPath, page, scrapeFilePermission)
	})

func writePackageIndex(fs billy.Filesystem, pres *godoc.Presentation, packagePath, outputBasePath string) error {
	_, err := writePackageIndexPipe.Do(fs, pres, packagePath, outputBasePath)
	return err
}

func pagePath(dir bool, path string) string {
	if dir {
		path += "/"
	}
	return path
}

type writeSourceArgs struct {
	FS             billy.Filesystem
	Pres           *godoc.Presentation
	BaseURL        string
	PackagePath    string
	IsDir          bool
	FileName       string
	OutputBasePath string
	Linker         source.Linker
}

var writeSourceFilePipe = pipe.New(pipe.Options{}).
	Append(func(args []interface{}) writeSourceArgs {
		return args[0].(writeSourceArgs)
	}).
	Append(func(args writeSourceArgs) (writeSourceArgs, error) {
		scrapeLinker, ok := args.Linker.(source.ScrapeChecker)
		return args, pipe.CheckError(ok && !scrapeLinker.ShouldScrapePackage(args.PackagePath), errSkipFile)
	}).
	Append(func(args writeSourceArgs) (writeSourceArgs, []byte, error) {
		p := path.Join("/src", args.PackagePath, args.FileName)
		p = pagePath(args.IsDir, p)
		page, err := getPage(args.Pres, p)
		return args, page, err
	}).
	Append(func(args writeSourceArgs, page []byte) (writeSourceArgs, []byte, error) {
		page, err := customizeSourceCodePage(args.BaseURL, page)
		return args, page, err
	}).
	Append(func(args writeSourceArgs, page []byte) (writeSourceArgs, []byte, string, error) {
		outputComponents := append([]string{args.OutputBasePath, "src"}, pathSplit(args.PackagePath)...)
		outputComponents = append(outputComponents, args.FileName)
		if args.IsDir {
			outputComponents = append(outputComponents, "index")
		}
		outputPath := filepath.Join(outputComponents...) + ".html"
		return args, page, outputPath, args.FS.MkdirAll(filepath.Dir(outputPath), scrapeDirPermission)
	}).
	Append(func(args writeSourceArgs, page []byte, outputPath string) error {
		return util.WriteFile(args.FS, outputPath, page, scrapeFilePermission)
	})

func writeSourceFile(fs billy.Filesystem, pres *godoc.Presentation, baseURL, packagePath string, isDir bool, fileName, outputBasePath string, linker source.Linker) error {
	_, err := writeSourceFilePipe.Do(writeSourceArgs{
		FS:             fs,
		Pres:           pres,
		BaseURL:        baseURL,
		PackagePath:    packagePath,
		IsDir:          isDir,
		FileName:       fileName,
		OutputBasePath: outputBasePath,
		Linker:         linker,
	})
	return pipe.CheckError(!errors.Is(err, errSkipFile), err)
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
	err := template.Must(template.New("").Parse(redirectHTML)).Execute(&buf, map[string]interface{}{
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
	return pipe.CheckError(!errors.Is(err, filepath.SkipDir), err)
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
		if errors.Is(err, filepath.SkipDir) {
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
	err := fs.MkdirAll(path, scrapeDirPermission)
	if err != nil {
		panic(err)
	}
	return fs
}
