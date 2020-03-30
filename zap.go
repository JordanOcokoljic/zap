package zap

import (
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// checkDirectoryExists determines if a directory at the specified path exists
// and if so, returns true. It will return false if the item at the specified
// path does not exist, or is anything other than a directory.
func checkDirectoryExists(check string) bool {
	stat, err := os.Stat(check)
	if err != nil {
		return false
	}

	return !os.IsNotExist(err) && stat.IsDir()
}

// getPackageInDirectory returns the representation of a package at the given
// directory.
func getPackageInDirectory(dir string) (*build.Package, error) {
	return build.ImportDir(dir, 0)
}

// getPackagesInProject will begin at the root of the project and identify all
// directories, and get the packages contained within those directories
// recursively. It will not follow symbolic links because it uses Walk to
// find the directories.
func getPackagesInProject(root string, jmp []string) ([]*build.Package, error) {
	var packages []*build.Package

	walkProject := func(fp string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if jmp != nil {
			for _, toSkip := range jmp {
				if info.Name() == toSkip {
					return filepath.SkipDir
				}
			}
		}

		if info.IsDir() {
			pkg, err := getPackageInDirectory(fp)
			if err != nil {
				return err
			}

			packages = append(packages, pkg)
		}

		return nil
	}

	err := filepath.Walk(root, walkProject)
	return packages, err
}

// getZappedImportName returns the name zapped was imported under, or an empty
// string if it's not imported.
func getZappedImportName(f *ast.File) string {
	var importedAs string

	for _, imp := range f.Imports {
		if !strings.HasSuffix(imp.Path.Value, "zapped\"") {
			continue
		}

		switch imp.Name {
		case nil:
			importedAs = "zapped"
		default:
			importedAs = imp.Name.Name
		}

		break
	}

	return importedAs
}

// getBoxesInFile scans through a provided *ast.File and identifies all calls
// to zapped.Box
func getBoxesInFile(f *ast.File, importName string) []string {
	var boxes []string

	// State based parsing just saves us having to use a million diffent
	// variables to track what everything is doing.
	type State uint8
	var state State
	const (
		InitialState State = iota
		ExpectingBox
		ExpectingPath
	)

	ast.Inspect(f, func(node ast.Node) bool {
		if node == nil {
			return false
		}

		switch n := node.(type) {
		case *ast.Ident:
			if state == InitialState {
				if n.Name == importName {
					state = ExpectingBox
				}

				if importName == "." && n.Name == "OpenBox" {
					state = ExpectingPath
				}
			}

			if state == ExpectingBox && n.Name == "OpenBox" {
				state = ExpectingPath
			}

		case *ast.BasicLit:
			if state == ExpectingPath && n.Kind == token.STRING {
				state = InitialState

				// Because the parser will spit out strings with the quotations
				// included we need to strip them out.
				boxes = append(boxes, n.Value[1:len(n.Value)-1])
			}
		}

		return true
	})

	return boxes
}

// getBoxesInPackage will return a list of directory paths that are read
// requested in the source code.
func getBoxesInPackage(pkg *build.Package) ([]string, error) {
	var boxes []string

	for _, filename := range pkg.GoFiles {
		fullPath := filepath.Join(pkg.Dir, filename)

		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, fullPath, nil, 0)
		if err != nil {
			return boxes, err
		}

		importedName := getZappedImportName(f)
		if importedName == "" || importedName == "_" {
			continue
		}

		boxesInFile := getBoxesInFile(f, importedName)
		boxes = append(boxes, boxesInFile...)
	}

	return boxes, nil
}
