// Command covet reads version control diffs and Go coverage files to generate reports on their intersection.
package main

import (
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/fatih/color"
	osfs "github.com/hack-pad/hackpadfs/os"
)

// nolint:gochecknoglobals // These globals are required to handle pre-existing globals in other libraries. Access to them is tightly controlled and minimized.
var (
	osExiter            = os.Exit
	osErr     io.Writer = os.Stderr
	colorOnce sync.Once
)

func setColorOnce(shouldColor bool) {
	colorOnce.Do(func() {
		color.NoColor = !shouldColor
	})
}

func main() {
	if os.Getenv("CI") == "true" {
		setColorOnce(true)
	}
	err := run(
		os.Args[1:],
		os.Stdin,
		os.Stdout,
		osErr,
		osfs.NewFS(),
	)
	if err != nil {
		fmt.Fprintln(osErr, err)
		osExiter(1)
		return
	}
}
