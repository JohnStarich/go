package main

import (
	"errors"
	"fmt"
	"io"
	"os"
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

type exitCoder interface {
	error
	ExitCode() int
}

func exitCode(err error) int {
	code := 1
	var exitErr exitCoder
	if errors.As(err, &exitErr) {
		code = exitErr.ExitCode()
	}
	return code
}
