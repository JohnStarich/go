package flags

import (
	"bytes"
	"flag"
)

// Args contains all command-line options for gopages
type Args struct {
	BaseURL          string
	GitHubPages      bool
	GitHubPagesToken string
	GitHubPagesUser  string
	OutputPath       string
	SiteDescription  string
	SiteTitle        string
}

func Parse(osArgs ...string) (Args, string, error) {
	var args Args
	commandLine := flag.NewFlagSet("gopages", flag.ContinueOnError)
	commandLine.StringVar(&args.OutputPath, "out", "dist", "Output path for static files")
	commandLine.StringVar(&args.BaseURL, "base", "", "Base URL to use for static assets")
	commandLine.StringVar(&args.SiteTitle, "brand-title", "", "Branding title in the top left of documentation")
	commandLine.StringVar(&args.SiteDescription, "brand-description", "", "Branding description in the top left of documentation")
	commandLine.BoolVar(&args.GitHubPages, "gh-pages", false, "Automatically commit the output path to the gh-pages branch. The current branch must be clean.")
	commandLine.StringVar(&args.GitHubPagesUser, "gh-pages-user", "", "The Git username to push with")
	commandLine.StringVar(&args.GitHubPagesToken, "gh-pages-token", "", "The Git token to push with. Usually this is an API key.")
	var output bytes.Buffer
	commandLine.SetOutput(&output)
	err := commandLine.Parse(osArgs) // prints usage if fails
	return args, output.String(), err
}
