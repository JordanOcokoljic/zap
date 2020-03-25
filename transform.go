package zap

import (
	"go/build"
	"os"
)

// getPackageAtPath returns the *build.Package for a given path.
func getPackageAtPath(path string) (*build.Package, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	return build.Import(path, pwd, 0)
}
