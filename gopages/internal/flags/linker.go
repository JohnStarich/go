package flags

import (
	"bytes"
	"fmt"
	"net/url"
	"path"
	"strings"
	"text/template"

	"github.com/johnstarich/go/gopages/internal/generate/source"
)

// GoPagesLinker is the default GoPages source.Linker, which links to scraped godoc source pages
type GoPagesLinker struct {
	baseURL string
}

func newGoPagesLinker(baseURL string) *GoPagesLinker {
	return &GoPagesLinker{
		baseURL: baseURL,
	}
}

// LinkToSource implements source.Linker
func (l *GoPagesLinker) LinkToSource(packagePath string, options source.LinkOptions) url.URL {
	u := url.URL{
		Path: path.Join(l.baseURL, "/src", packagePath),
	}
	if path.Ext(u.Path) == ".go" {
		u.Path += ".html"
	}
	if options.Line > 0 {
		u.Fragment = fmt.Sprintf("L%d", options.Line)
	}
	return u
}

// TemplateLinker is a custom template-based source.Linker, which links to external source pages
type TemplateLinker struct {
	template      *template.Template
	modulePackage string
}

var _ source.Linker = &TemplateLinker{}
var _ source.ScrapeChecker = &TemplateLinker{}

func newTemplateLinker(modulePackageURL, tmpl string) (*TemplateLinker, error) {
	var l TemplateLinker
	modulePackage, err := url.Parse(modulePackageURL)
	if err == nil {
		l.modulePackage = path.Join(modulePackage.Host, modulePackage.Path)
		l.template, err = template.New("").Parse(tmpl)
	}
	return &l, err
}

// LinkToSource implements source.Linker
func (l *TemplateLinker) LinkToSource(packagePath string, options source.LinkOptions) url.URL {
	filePath := strings.TrimPrefix(packagePath, l.modulePackage)
	filePath = strings.TrimPrefix(filePath, "/")
	if filePath == packagePath {
		return url.URL{}
	}
	var args = struct {
		Path string
		source.LinkOptions
	}{
		Path:        filePath,
		LinkOptions: options,
	}
	var buf bytes.Buffer
	err := l.template.Execute(&buf, args)
	panicIfErr(err)
	var u url.URL
	err = u.UnmarshalBinary(buf.Bytes())
	panicIfErr(err)
	return u
}

// ShouldScrapePackage implements source.ScrapeChecker
func (l *TemplateLinker) ShouldScrapePackage(packagePath string) bool {
	return strings.HasPrefix(packagePath, l.modulePackage+"/")
}

func panicIfErr(err error) {
	if err != nil {
		panic(err)
	}
}
