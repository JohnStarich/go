package flags

import (
	"errors"
	"testing"
	"text/template"

	"github.com/johnstarich/go/gopages/internal/generate/source"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGoPagesLinker(t *testing.T) {
	t.Parallel()
	const someBaseURL = "/some/base"
	linker := newGoPagesLinker(someBaseURL)
	assert.Equal(t, &GoPagesLinker{baseURL: someBaseURL}, linker)
}

func TestGoPagesLinkToSource(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		description string
		baseURL     string
		pkgPath     string
		options     source.LinkOptions
		expectLink  string
	}{
		{
			description: "simple path",
			pkgPath:     "github.com/org/repo/mypkg/myfile.go",
			expectLink:  "/src/github.com/org/repo/mypkg/myfile.go",
		},
		{
			description: "base URL",
			baseURL:     "/some/base",
			pkgPath:     "github.com/org/repo/mypkg/myfile.go",
			expectLink:  "/some/base/src/github.com/org/repo/mypkg/myfile.go",
		},
		{
			description: "line number",
			pkgPath:     "github.com/org/repo/mypkg/myfile.go",
			options:     source.LinkOptions{Line: 10},
			expectLink:  "/src/github.com/org/repo/mypkg/myfile.go#L10",
		},
	} {
		tc := tc // enable parallel sub-tests
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()
			linker := newGoPagesLinker(tc.baseURL)
			url := linker.LinkToSource(tc.pkgPath, tc.options)
			assert.Equal(t, tc.expectLink, url.String())
		})
	}
}

func TestNewTemplateLinker(t *testing.T) {
	t.Parallel()
	const (
		someModulePkg = "https://github.com/johnstarich/go"
		someTemplate  = "https://github.com/johnstarich/go/blob/master/gopages/{{.Path}}{{if .Line}}#L{{.Line}}{{end}}"
	)
	linker, err := newTemplateLinker(someModulePkg, someTemplate)
	assert.NoError(t, err)
	assert.Equal(t, &TemplateLinker{
		modulePackage: "github.com/johnstarich/go",
		template:      template.Must(template.New("").Parse(someTemplate)),
	}, linker)
}

func TestTemplateLinkToSource(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		description string
		modulePkg   string
		template    string
		pkgPath     string
		options     source.LinkOptions
		expectLink  string
	}{
		{
			description: "simple path",
			modulePkg:   "github.com/org/repo",
			template:    "{{.Path}}",
			pkgPath:     "github.com/org/repo/mypkg/myfile.go",
			expectLink:  "mypkg/myfile.go",
		},
		{
			description: "line number",
			modulePkg:   "github.com/org/repo",
			template:    "{{.Path}}#L{{.Line}}",
			pkgPath:     "github.com/org/repo/mypkg/myfile.go",
			options:     source.LinkOptions{Line: 10},
			expectLink:  "mypkg/myfile.go#L10",
		},
		{
			description: "non-module path",
			modulePkg:   "github.com/org/repo",
			template:    "{{.Path}}#L{{.Line}}",
			pkgPath:     "example.com/org/repo/mypkg/myfile.go",
			expectLink:  "",
		},
	} {
		tc := tc // enable parallel sub-tests
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()
			linker, err := newTemplateLinker(tc.modulePkg, tc.template)
			require.NoError(t, err)
			url := linker.LinkToSource(tc.pkgPath, tc.options)
			assert.Equal(t, tc.expectLink, url.String())
		})
	}
}

func TestTemplateShouldScrapePackage(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		description  string
		modulePkg    string
		pkgPath      string
		expectScrape bool
	}{
		{
			description:  "same package",
			modulePkg:    "github.com/org/repo",
			pkgPath:      "github.com/org/repo/mypkg/myfile.go",
			expectScrape: true,
		},
		{
			description:  "different package",
			modulePkg:    "github.com/org/repo",
			pkgPath:      "example.com/org/repo/mypkg/myfile.go",
			expectScrape: false,
		},
	} {
		tc := tc // enable parallel sub-tests
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()
			linker, err := newTemplateLinker(tc.modulePkg, "")
			require.NoError(t, err)
			shouldScrape := linker.ShouldScrapePackage(tc.pkgPath)
			assert.Equal(t, tc.expectScrape, shouldScrape)
		})
	}
}

func TestPanicIfErr(t *testing.T) {
	t.Parallel()
	assert.PanicsWithError(t, "some error", func() {
		panicIfErr(errors.New("some error"))
	})

	assert.NotPanics(t, func() {
		panicIfErr(nil)
	})
}
