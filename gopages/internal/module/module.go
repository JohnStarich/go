package module

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/johnstarich/go/gopages/internal/pipe"
	"github.com/pkg/errors"
	"golang.org/x/mod/modfile"
)

func Package(modulePath string) (string, error) {
	goMod := filepath.Join(modulePath, "go.mod")
	var modulePackage string
	err := pipe.ChainFuncs(
		func() error {
			_, err := os.Stat(goMod)
			return pipe.ErrIf(os.IsNotExist(err), errors.New("go.mod not found in the current directory"))
		},
		func() error {
			buf, err := ioutil.ReadFile(goMod)
			modulePackage = modfile.ModulePath(buf)
			return err
		},
		func() error {
			return pipe.ErrIf(modulePackage == "", errors.Errorf("Unable to find module package name in go.mod file: %s", goMod))
		},
	).Do()
	return modulePackage, err
}
