// Package module finds a module's package path for a given file path.
package module

import (
	"os"
	"path/filepath"

	"github.com/johnstarich/go/pipe"
	"golang.org/x/mod/modfile"
)

var packagePipe = pipe.New(pipe.Options{}).
	Append(func(args []interface{}) string {
		modulePath := args[0].(string)
		return modulePath
	}).
	Append(func(modulePath string) (string, error) {
		goMod := filepath.Join(modulePath, "go.mod")
		_, err := os.Stat(goMod)
		return goMod, pipe.CheckErrorf(os.IsNotExist(err), "go.mod not found in the current directory")
	}).
	Append(func(goMod string) (string, string, error) {
		buf, err := os.ReadFile(goMod)
		modulePackage := modfile.ModulePath(buf)
		return goMod, modulePackage, err
	}).
	Append(func(goMod, modulePackage string) (string, error) {
		return modulePackage, pipe.CheckErrorf(modulePackage == "", "Unable to find module package name in go.mod file: %s", goMod)
	})

// Package returns a module's package path for the given module file path
func Package(modulePath string) (string, error) {
	out, err := packagePipe.Do(modulePath)
	var modulePackage string
	if err == nil {
		modulePackage = out[0].(string)
	}
	return modulePackage, err
}
