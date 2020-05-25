package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/util"
	"github.com/johnstarich/go/gopages/internal/flags"
	"github.com/pkg/errors"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/godoc"
	"golang.org/x/tools/godoc/static"
	"golang.org/x/tools/godoc/vfs"
	"golang.org/x/tools/godoc/vfs/mapfs"
)

func readTemplates(pres *godoc.Presentation, funcs template.FuncMap, fs vfs.FileSystem) {
	pres.CallGraphHTML = readTemplate(funcs, fs, "callgraph.html")
	pres.DirlistHTML = readTemplate(funcs, fs, "dirlist.html")
	pres.ErrorHTML = readTemplate(funcs, fs, "error.html")
	pres.ExampleHTML = readTemplate(funcs, fs, "example.html")
	pres.GodocHTML = parseTemplate(funcs, "godoc.html", godocHTML)
	pres.ImplementsHTML = readTemplate(funcs, fs, "implements.html")
	pres.MethodSetHTML = readTemplate(funcs, fs, "methodset.html")
	pres.PackageHTML = readTemplate(funcs, fs, "package.html")
	pres.PackageRootHTML = readTemplate(funcs, fs, "packageroot.html")
}

func readTemplate(funcs template.FuncMap, fs vfs.FileSystem, name string) *template.Template {
	// use underlying file system fs to read the template file
	// (cannot use template ParseFile functions directly)
	data, err := vfs.ReadFile(fs, path.Join("lib/godoc", name))
	if err != nil {
		panic(err)
	}
	return parseTemplate(funcs, name, string(data))
}

func parseTemplate(funcs template.FuncMap, name, data string) *template.Template {
	t, err := template.New(name).Funcs(funcs).Parse(data)
	if err != nil {
		panic(err)
	}
	return t
}

func addGoPagesFuncs(funcs template.FuncMap, args flags.Args) {
	var longTitle string
	if args.SiteTitle != "" && args.SiteDescription != "" {
		longTitle = fmt.Sprintf("%s | %s", args.SiteTitle, args.SiteDescription)
	}
	values := map[string]interface{}{
		"BaseURL":       args.BaseURL,
		"SiteTitle":     args.SiteTitle,
		"SiteTitleLong": longTitle,
	}
	funcs["gopages"] = func(defaultValue, firstKey string, keys ...string) (string, error) {
		keys = append([]string{firstKey}, keys...) // require at least one key
		for _, key := range keys {
			value, ok := values[key]
			if !ok {
				return "", errors.Errorf("Unknown gopages key: %q", key)
			}
			valueStr, isString := value.(string)
			if !isString {
				return "", errors.Errorf("gopages key %q is not a string", key)
			}
			if valueStr != "" {
				return template.HTMLEscapeString(valueStr), nil
			}
		}
		return defaultValue, nil
	}
}

func generateDocs(modulePath, modulePackage string, args flags.Args, src, fs billy.Filesystem) error {
	if err := util.RemoveAll(fs, args.OutputPath); err != nil {
		return errors.Wrap(err, "Failed to clean output directory")
	}
	if err := fs.MkdirAll(args.OutputPath, 0700); err != nil {
		return errors.Wrap(err, "Failed to create output directory")
	}

	ns := vfs.NewNameSpace()
	ns.Bind("/lib/godoc", mapfs.New(static.Files), "/", vfs.BindReplace)
	srcRoot, err := src.Chroot(modulePath)
	if err != nil {
		return errors.Wrapf(err, "Failed to chroot the source file system to %q", modulePath)
	}
	modFS := &filesystemOpener{Filesystem: srcRoot}
	ns.Bind(path.Join("/src", modulePackage), modFS, "/", vfs.BindReplace)

	corpus := godoc.NewCorpus(ns)
	if err := corpus.Init(); err != nil {
		return errors.Wrap(err, "Failed to initialize corpus")
	}

	pres := godoc.NewPresentation(corpus)
	funcs := pres.FuncMap()
	addGoPagesFuncs(funcs, args)
	readTemplates(pres, funcs, ns)

	// Generate all static assets and save to /lib/godoc
	for name, content := range static.Files {
		path := filepath.Join(args.OutputPath, "lib", "godoc", name)
		if err := fs.MkdirAll(filepath.Dir(path), 0700); err != nil {
			return err
		}
		err := util.WriteFile(fs, path, []byte(content), 0600)
		if err != nil {
			return err
		}
	}

	// Generate main index to redirect to actual content page. Important to separate from 'lib' top-level dir.
	err = util.WriteFile(fs, filepath.Join(args.OutputPath, "index.html"), []byte(redirect("pkg/")), 0600)
	if err != nil {
		return err
	}

	custom404, err := genericPage(pres, "Page not found", `
<p>
<span class="alert" style="font-size:120%">
Oops, this page doesn't exist.
</span>
</p>
<p>If something should be here, <a href="https://github.com/JohnStarich/go/issues/new" target="_blank">open an issue</a>.</p>
`)
	if err != nil {
		return err
	}
	err = util.WriteFile(fs, filepath.Join(args.OutputPath, "404.html"), custom404, 0600)
	if err != nil {
		return err
	}

	// For each package, generate an index page
	paths, err := getPackagePaths(modulePackage)
	if err != nil {
		return err
	}
	for _, path := range paths {
		err = scrapePackage(fs, pres, modulePackage, path, filepath.Join(args.OutputPath, "pkg"))
		if err != nil {
			return err
		}
	}
	return nil
}

func doRequest(do func(w http.ResponseWriter)) ([]byte, error) {
	recorder := httptest.NewRecorder()
	do(recorder)
	if recorder.Result().StatusCode != http.StatusOK {
		return nil, errors.Errorf("Error generating page: [%d]\n%s", recorder.Result().StatusCode, recorder.Body.String())
	}
	return recorder.Body.Bytes(), nil
}

func getPage(pres *godoc.Presentation, path string) ([]byte, error) {
	return doRequest(func(w http.ResponseWriter) {
		pres.ServeHTTP(w, &http.Request{
			URL: &url.URL{Path: path},
		})
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

func scrapePackage(fs billy.Filesystem, pres *godoc.Presentation, moduleRoot, packagePath, outputPath string) error {
	if moduleRoot != packagePath && !strings.HasPrefix(packagePath, moduleRoot+"/") {
		return errors.Errorf("Package path %q must be rooted by module: %q", packagePath, moduleRoot)
	}
	var packageRelPath string
	if moduleRoot != packagePath {
		packageRelPath = strings.TrimPrefix(packagePath, moduleRoot+"/")
	}
	outputComponents := filepath.SplitList(outputPath)
	if packageRelPath != "" {
		outputComponents = append(outputComponents, strings.Split(packageRelPath, "/")...)
	}
	outputComponents = append(outputComponents, "index.html")
	outputPath = filepath.Join(outputComponents...)

	page, err := getPage(pres, path.Join("/pkg", packagePath)+"/")
	if err != nil {
		return err
	}
	if err := fs.MkdirAll(filepath.Dir(outputPath), 0700); err != nil {
		return err
	}
	return util.WriteFile(fs, outputPath, page, 0600)
}

func getPackagePaths(modulePackage string) ([]string, error) {
	pkgs, err := packages.Load(&packages.Config{
		Mode: packages.NeedName,
	}, modulePackage+"/...")
	if err != nil {
		return nil, err
	}

	paths := make([]string, len(pkgs))
	for i, pkg := range pkgs {
		paths[i] = pkg.PkgPath
	}
	return paths, nil
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
