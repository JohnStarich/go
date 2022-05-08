package main

import (
	"fmt"
	"os"
	"path"

	"github.com/fatih/color"
	osfs "github.com/hack-pad/hackpadfs/os"
	"github.com/johnstarich/go/diffcover/internal/fspath"
)

func main() {
	if os.Getenv("CI") == "true" {
		color.NoColor = false
	}
	err := run(
		os.Args[1:],
		os.Stdin,
		os.Stdout,
		os.Stderr,
		osfs.NewFS(),
		path.Clean(fspath.ToFSPath(os.TempDir())), // TODO ensure tempdir exists
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
		return
	}
}
