package zap

import (
	"go/build"
	"os"
	"path/filepath"
)

// GetPackagesInProject will return the package in the current directory, as
// well as all the subdirectories under it. Under the hood it uses Walk, so
// it won't follow symbolic links.
func GetPackagesInProject(wd string, skip []string) ([]*build.Package, error) {
	var packages []*build.Package

	fn := func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			return nil
		}

		for _, name := range skip {
			if info.Name() == name {
				return filepath.SkipDir
			}
		}

		pkg, err := build.ImportDir(path, 0)
		if err != nil {
			return err
		}

		packages = append(packages, pkg)

		return nil
	}

	err := filepath.Walk(wd, fn)
	return packages, err
}
