package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/pkg/errors"
	"golang.org/x/mod/modfile"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/godoc"
	"golang.org/x/tools/godoc/static"
	"golang.org/x/tools/godoc/vfs"
	"golang.org/x/tools/godoc/vfs/mapfs"
)

type Args struct {
	BaseURL    string
	OutputPath string
}

func main() {
	var args Args
	flag.StringVar(&args.OutputPath, "out", "dist", "Output path for static files")
	flag.StringVar(&args.BaseURL, "base", "", "Base URL to use for static assets")
	flag.Parse()

	err := run(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func run(args Args) error {
	modulePath, err := os.Getwd()
	if err != nil {
		return err
	}

	goMod := filepath.Join(modulePath, "go.mod")
	if _, err := os.Stat(goMod); os.IsNotExist(err) {
		return errors.New("go.mod not found in the current directory")
	}

	buf, err := ioutil.ReadFile(goMod)
	if err != nil {
		return err
	}

	modulePackage := modfile.ModulePath(buf)
	if modulePackage == "" {
		return errors.Errorf("Unable to find module package name in go.mod file: %s", goMod)
	}

	if err := os.RemoveAll(args.OutputPath); err != nil {
		return err
	}
	if err := os.MkdirAll(args.OutputPath, 0700); err != nil {
		return err
	}

	fmt.Println("Generating godoc static pages for module...", modulePackage)

	fs := vfs.NewNameSpace()
	fs.Bind("/lib/godoc", mapfs.New(static.Files), "/", vfs.BindReplace)
	modFS := vfs.OS(modulePath)
	fs.Bind(path.Join("/src", modulePackage), modFS, "/", vfs.BindReplace)

	corpus := godoc.NewCorpus(fs)
	corpus.Init()

	presentation := godoc.NewPresentation(corpus)
	readTemplates(args, presentation, fs)

	for name, content := range static.Files {
		path := filepath.Join(args.OutputPath, "lib", "godoc", name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			return err
		}
		err := ioutil.WriteFile(path, []byte(content), 0600)
		if err != nil {
			return err
		}
	}

	err = ioutil.WriteFile(filepath.Join(args.OutputPath, "index.html"), []byte(redirect("pkg/")), 0600)
	if err != nil {
		return err
	}

	paths, err := getPackagePaths(modulePackage)
	if err != nil {
		return err
	}
	for _, path := range paths {
		err = scrape(presentation, modulePackage, path, filepath.Join(args.OutputPath, "pkg"))
		if err != nil {
			return err
		}
	}
	return nil
}

func scrape(p *godoc.Presentation, moduleRoot, packagePath, outputPath string) error {
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

	recorder := httptest.NewRecorder()
	p.ServeHTTP(recorder, &http.Request{
		URL: &url.URL{
			Path: path.Join("/pkg", packagePath) + "/",
		},
	})
	if recorder.Result().StatusCode != http.StatusOK {
		return errors.Errorf("Error scraping doc: [%d]\n%s", recorder.Result().StatusCode, recorder.Body.String())
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0700); err != nil {
		return err
	}
	return ioutil.WriteFile(outputPath, recorder.Body.Bytes(), 0600)
}

func readTemplates(args Args, p *godoc.Presentation, fs vfs.FileSystem) {
	funcs := p.FuncMap()
	funcs["baseURL"] = func() string {
		return args.BaseURL
	}

	p.CallGraphHTML = readTemplate(funcs, fs, "callgraph.html")
	p.DirlistHTML = readTemplate(funcs, fs, "dirlist.html")
	p.ErrorHTML = readTemplate(funcs, fs, "error.html")
	p.ExampleHTML = readTemplate(funcs, fs, "example.html")
	p.GodocHTML = parseTemplate(funcs, "godoc.html", godocHTML)
	p.ImplementsHTML = readTemplate(funcs, fs, "implements.html")
	p.MethodSetHTML = readTemplate(funcs, fs, "methodset.html")
	p.PackageHTML = readTemplate(funcs, fs, "package.html")
	p.PackageRootHTML = readTemplate(funcs, fs, "packageroot.html")
	// Disable search, since that requires a server
	p.SearchHTML = readTemplate(funcs, fs, "search.html")
	p.SearchDocHTML = readTemplate(funcs, fs, "searchdoc.html")
	p.SearchCodeHTML = readTemplate(funcs, fs, "searchcode.html")
	p.SearchTxtHTML = readTemplate(funcs, fs, "searchtxt.html")
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
