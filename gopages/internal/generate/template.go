package generate

import (
	"fmt"
	"path"
	"strings"
	"text/template"
	"time"

	"github.com/johnstarich/go/gopages/internal/flags"
	"github.com/johnstarich/go/pipe"
	"github.com/pkg/errors"
	"golang.org/x/tools/godoc"
	"golang.org/x/tools/godoc/vfs"
)

var (
	errTemplateValueEmpty  = fmt.Errorf("empty template value")
	goPagesTemplateVarPipe = pipe.New(pipe.Options{}).
				Append(func(args []interface{}) (string, interface{}, bool) {
			templateKey := args[0].(string)
			templateValue := args[1]
			templateValueExists := args[2].(bool)
			return templateKey, templateValue, templateValueExists
		}).
		Append(func(key string, value interface{}, exists bool) (string, interface{}, error) {
			return key, value, pipe.CheckErrorf(!exists, "Unknown gopages key: %q", key)
		}).
		Append(func(key string, value interface{}) (string, error) {
			valueStr, isString := value.(string)
			return valueStr, pipe.CheckErrorf(!isString, "gopages key %q is not a string", key)
		}).
		Append(func(value string) (string, error) {
			return value, pipe.CheckError(value == "", errTemplateValueEmpty)
		}).
		Append(template.HTMLEscapeString)
)

func addGoPagesFuncs(funcs template.FuncMap, modulePackage string, args flags.Args) {
	funcs["node_html"] = nodeHTML(funcs["node_html"].(node_htmlFunc), args.BaseURL)

	longTitle := fmt.Sprintf("%s | %s", args.SiteTitle, args.SiteDescription)
	if args.SiteTitle == "" || args.SiteDescription == "" {
		longTitle = ""
	}
	values := map[string]interface{}{
		"BaseURL":       args.BaseURL,
		"ModuleURL":     path.Join(args.BaseURL, "/pkg", modulePackage) + "/",
		"SiteTitle":     args.SiteTitle,
		"SiteTitleLong": longTitle,
	}
	funcs["gopages"] = func(defaultValue, firstKey string, keys ...string) (string, error) {
		keys = append([]string{firstKey}, keys...) // require at least one key
		multiArgs := make([][]interface{}, len(keys))
		for i, key := range keys {
			value, ok := values[key]
			multiArgs[i] = []interface{}{key, value, ok, defaultValue}
		}
		out, err := pipe.Filter(goPagesTemplateVarPipe, multiArgs)
		result := defaultValue
		if err == nil {
			result = out[0][0].(string)
		}
		if errors.Is(err, errTemplateValueEmpty) {
			err = nil
		}
		return result, err
	}
	funcs["gopagesWatchScript"] = func() string {
		script := fmt.Sprintf(`
<script>
const startDate = %q
const timeoutMillis = 2000
const poll = () => {
	fetch(window.location).then(resp => {
		const newDate = resp.headers.get("GoPages-Last-Updated")
		if (newDate != startDate) {
			window.location.reload()
		}
	})
}
window.setInterval(poll, timeoutMillis)
</script>
`, time.Now().Format(time.RFC3339))
		if !args.Watch {
			script = ""
		}
		return script
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
	return template.Must(template.New(name).Funcs(funcs).Parse(data))
}

type node_htmlFunc = func(info *godoc.PageInfo, node interface{}, linkify bool) string

// nodeHTML runs the original 'node_html' template func, then rewrites any links inside it
func nodeHTML(original node_htmlFunc, baseURL string) node_htmlFunc {
	pkgURL := path.Join(baseURL, "/pkg")
	return func(info *godoc.PageInfo, node interface{}, linkify bool) string {
		s := original(info, node, linkify)
		return strings.ReplaceAll(s, `<a href="/pkg/`, fmt.Sprintf(`<a href="%s/`, pkgURL))
	}
}
