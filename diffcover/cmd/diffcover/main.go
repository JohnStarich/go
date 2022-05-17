package main

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	osfs "github.com/hack-pad/hackpadfs/os"
)

var osExiter = os.Exit

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
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		osExiter(1)
		return
	}
}
