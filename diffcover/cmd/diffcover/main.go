package main

import (
	"fmt"
	"io"
	"os"

	"github.com/fatih/color"
	osfs "github.com/hack-pad/hackpadfs/os"
)

var (
	osExiter           = os.Exit
	osErr    io.Writer = os.Stderr
)

func main() {
	if os.Getenv("CI") == "true" {
		color.NoColor = false
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
