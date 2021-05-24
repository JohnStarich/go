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

type GoPagesLinker struct {
	baseURL string
}

func newGoPagesLinker(baseURL string) *GoPagesLinker {
	return &GoPagesLinker{
		baseURL: baseURL,
	}
}

func (l *GoPagesLinker) LinkToSource(packagePath string, options source.LinkOptions) url.URL {
	u := url.URL{
		Path: path.Join(l.baseURL, "/src", packagePath),
	}
	if options.Line > 0 {
		u.Fragment = fmt.Sprintf("L%d", options.Line)
	}
	return u
}

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

func (l *TemplateLinker) ShouldScrapePackage(packagePath string) bool {
	return strings.HasPrefix(packagePath, l.modulePackage+"/")
}

func panicIfErr(err error) {
	if err != nil {
		panic(err)
	}
}
