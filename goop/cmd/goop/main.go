package main

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/urfave/cli/v2"
)

//nolint:gochecknoglobals // This minimal set of globals enables testing of main().
var (
	osArgs             = os.Args
	osExiter           = os.Exit
	osOut    io.Writer = os.Stdout
	osErr    io.Writer = os.Stderr
)

func main() {
	err := run(osArgs, osOut, osErr)
	if err != nil {
		fmt.Fprintln(osErr, err)
		osExiter(exitCode(err))
		return
	}
}

func exitCode(err error) int {
	code := 1
	var exitCoder cli.ExitCoder
	if errors.As(err, &exitCoder) {
		code = exitCoder.ExitCode()
	}
	return code
}
