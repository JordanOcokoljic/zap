package zap

import (
	"errors"
	"fmt"
	"os"
	"path"
	"zap/zapped"
)

var (
	// ErrorInitNoOp is returned when OperationInit is called in a directory
	// where the zap library has already been initialized.
	ErrorInitNoOp = errors.New("zap library already initialized")
)

// OperationInit creates a new folder containg the zap library for use within
// projects that use the tool.
func OperationInit(wd string) error {
	if checkDirectoryExists(path.Join(wd, "zapped")) {
		return ErrorInitNoOp
	}

	err := os.Mkdir("zapped", os.ModePerm)
	if err != nil {
		return err
	}

	zapped.OpenBox("zapped")

	return nil
}

// OperationBuild walks through the source code in the project, and identifies
// the directories that need to be embedded into Boxes, collects all the files
// within them and makes them accessible to the library created so the project
// can be built.
func OperationBuild(wd string) error {
	skip := []string{
		".git",
		"testdata",
		"zapped",
	}

	packages, err := getPackagesInProject(wd, skip)
	if err != nil {
		return err
	}

	var boxes []string
	for _, pkg := range packages {
		found, err := getBoxesInPackage(pkg)
		if err != nil {
			return err
		}

		boxes = append(boxes, found...)
	}

	fmt.Printf("%v\n", boxes)

	return nil
}

// OperationStub creates stub functions that allow calls from the library that
// just reach into the filesystem, rather than looking for the embedded files
// for testing and development purposes.
func OperationStub(wd string) error {
	return nil
}
