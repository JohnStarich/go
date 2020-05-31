package generate

import (
	"fmt"
	"path"
	"text/template"

	"github.com/johnstarich/go/gopages/internal/flags"
	"github.com/johnstarich/go/gopages/internal/pipe"
	"github.com/pkg/errors"
	"golang.org/x/tools/godoc"
	"golang.org/x/tools/godoc/vfs"
)

func addGoPagesFuncs(funcs template.FuncMap, modulePackage string, args flags.Args) {
	var longTitle string
	if args.SiteTitle != "" && args.SiteDescription != "" {
		longTitle = fmt.Sprintf("%s | %s", args.SiteTitle, args.SiteDescription)
	}
	values := map[string]interface{}{
		"BaseURL":       args.BaseURL,
		"ModuleURL":     path.Join(args.BaseURL, "pkg", modulePackage) + "/",
		"SiteTitle":     args.SiteTitle,
		"SiteTitleLong": longTitle,
	}
	funcs["gopages"] = func(defaultValue, firstKey string, keys ...string) (string, error) {
		keys = append([]string{firstKey}, keys...) // require at least one key
		for _, key := range keys {
			value, ok := values[key]
			var valueStr string
			err := pipe.ChainFuncs(
				func() error {
					return pipe.ErrIf(!ok, errors.Errorf("Unknown gopages key: %q", key))
				},
				func() error {
					var isString bool
					valueStr, isString = value.(string)
					return pipe.ErrIf(!isString, errors.Errorf("gopages key %q is not a string", key))
				},
			).Do()
			if err != nil || valueStr != "" {
				return template.HTMLEscapeString(valueStr), err
			}
		}
		return defaultValue, nil
	}
}

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
