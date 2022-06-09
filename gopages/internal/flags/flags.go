package flags

import (
	"bytes"
	"flag"

	"github.com/johnstarich/go/gopages/internal/generate/source"
)

// Args contains all command-line options for gopages
type Args struct {
	BaseURL            string
	GitHubPages        bool
	GitHubPagesToken   string
	GitHubPagesUser    string
	IncludeInHead      FilePathContents
	SourceLinkTemplate string
	OutputPath         string
	SiteDescription    string
	SiteTitle          string
	Watch              bool // not added as a flag, only enabled when running from ./cmd/watch
}

// Parse parses the given command line arguments into Args values and returns any output to send to the user
func Parse(osArgs ...string) (Args, string, error) {
	var args Args
	commandLine := flag.NewFlagSet("gopages", flag.ContinueOnError)
	commandLine.StringVar(&args.OutputPath, "out", "dist", "Output path for static files")
	commandLine.StringVar(&args.BaseURL, "base", "", "Base URL to use for static assets")
	commandLine.StringVar(&args.SiteTitle, "brand-title", "", "Branding title in the top left of documentation")
	commandLine.StringVar(&args.SiteDescription, "brand-description", "", "Branding description in the top left of documentation")
	commandLine.StringVar(&args.SourceLinkTemplate, "source-link", "", `Custom source code link template. Disables built-in source code pages. For example, "https://github.com/johnstarich/go/blob/master/gopages/{{.Path}}{{if .Line}}#L{{.Line}}{{end}}" generates links compatible with GitHub and GitLab. Must be a valid Go template and must generate valid URLs.`)
	commandLine.Var(&args.IncludeInHead, "include-head", "Includes the given HTML file's contents in every page's '<head></head>'. Useful for including custom analytics scripts. Must be valid HTML.")

	commandLine.BoolVar(&args.GitHubPages, "gh-pages", false, "Automatically commit the output path to the gh-pages branch. The current branch must be clean.")
	commandLine.StringVar(&args.GitHubPagesUser, "gh-pages-user", "", "The Git username to push with")
	commandLine.StringVar(&args.GitHubPagesToken, "gh-pages-token", "", "The Git token to push with. Usually this is an API key.")
	var output bytes.Buffer
	commandLine.SetOutput(&output)
	err := commandLine.Parse(osArgs) // prints usage if fails
	return args, output.String(), err
}

// Linker returns an appropriate source.Linker for the given command line args
func (a Args) Linker(modulePackage string) (source.Linker, error) {
	if a.SourceLinkTemplate != "" {
		return newTemplateLinker(modulePackage, a.SourceLinkTemplate)
	}
	return newGoPagesLinker(a.BaseURL), nil
}
