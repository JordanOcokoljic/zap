package zap

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// Resource is used to track each unique Key passed to a call to Resource() and
// the path specified in the call.
type Resource struct {
	Key  string
	Path string
}

// aggregateError is a collection of errors that fullfils the error interface,
// so it can be passed up, while still containing all the information.
type aggregateError struct {
	errors []error
}

// Add adds a new error to the list of errors.
func (ag *aggregateError) Add(err error) {
	ag.errors = append(ag.errors, err)
}

// Returns the value of the aggregateError, which is just a big collection of
// all the other errors. Fulfils the Error interface.
func (ag aggregateError) Error() string {
	var str string

	for i, err := range ag.errors {
		if i == 0 {
			str = err.Error()
			continue
		}

		str = fmt.Sprintf("%s\n%s", str, err.Error())
	}

	return str
}

// GetPackagesInProject will return the package in the current directory, as
// well as all the subdirectories under it. Under the hood it uses Walk, so
// it won't follow symbolic links.
func GetPackagesInProject(wd string) ([]*build.Package, error) {
	var packages []*build.Package

	fn := func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			return nil
		}

		pkg, err := build.ImportDir(path, 0)

		if err != nil {
			if !strings.HasPrefix(err.Error(), "no buildable Go source") {
				return err
			}

			return filepath.SkipDir
		}

		packages = append(packages, pkg)

		return nil
	}

	err := filepath.Walk(wd, fn)
	return packages, err
}

// getZappedImportName returns the name the Zapped library was imported under,
// returning empty string if it is not imported.
func getZappedImportName(file *ast.File) string {
	var name string

	for _, imp := range file.Imports {
		if strings.HasSuffix(imp.Path.Value, "zapped\"") {
			name = "zapped"

			if imp.Name != nil {
				name = imp.Name.Name
			}

			break
		}
	}

	return name
}

// Track the types of errors that could occur so that generateParseError knows
// what message to use.
const (
	errorUnknownCall uint8 = iota
	errorBadType
)

// generateParseError will return an error with correct formatting describing
// what was incorrect about the scanned source.
func generateParseError(fset *token.FileSet, p token.Pos, err uint8) error {
	pos := fset.Position(p)
	filename := pos.Filename

	// Reduces unecessary duplication of code.
	format := func(msg string) string {
		return fmt.Sprintf("%s:%d:%d: %s", filename, pos.Line, pos.Column, msg)
	}

	var msg string
	switch err {
	case errorUnknownCall:
		msg = format("expected Resource() but was something else")
	case errorBadType:
		msg = format("calls to Resource() require string literals")
	}

	return fmt.Errorf(msg)
}

// parse will walk the AST and identify calls to Resource() and extract
// the key and the path from them. It will return an error
func parse(f *ast.File, fset *token.FileSet, imp string) ([]Resource, error) {
	var resources []Resource
	var errors aggregateError

	// State based parsing let's us solve this problem without needing to have
	// a million different variables tracking everything.
	var state uint8
	const (
		NilState uint8 = iota
		ExpectingResourceCall
		ExpectingKey
		ExpectingPath
	)

	// Error handling code being pulled out into it's own closure reduces
	// repetition of simple code.
	handleError := func(node ast.Node, errorType uint8) {
		err := generateParseError(fset, node.Pos(), errorType)
		errors.Add(err)
	}

	// Pulling the code out of the switch improves the readability, and ensures
	// clarity of logic.
	resolveIdent := func(node *ast.Ident) uint8 {
		switch state {
		case NilState:
			if node.Name == imp {
				return ExpectingResourceCall
			}

		case ExpectingResourceCall:
			if node.Name != "Resource" {
				handleError(node, errorUnknownCall)
				return NilState
			}

			return ExpectingKey

		case ExpectingKey, ExpectingPath:
			handleError(node, errorBadType)
			return NilState
		}

		return state
	}

	resolveBasicLit := func(node *ast.BasicLit) uint8 {
		switch state {
		case ExpectingKey:
			if node.Kind != token.STRING {
				handleError(node, errorBadType)
				return NilState
			}

			res := Resource{Key: node.Value[1 : len(node.Value)-1]}
			resources = append(resources, res)
			return ExpectingPath

		case ExpectingPath:
			if node.Kind != token.STRING {
				handleError(node, errorBadType)
				return NilState
			}

			res := resources[len(resources)-1]
			res.Path = node.Value
			return NilState
		}

		return state
	}

	// Walk the AST.
	ast.Inspect(f, func(node ast.Node) bool {
		if node == nil {
			return false
		}

		switch n := node.(type) {
		case *ast.Ident:
			state = resolveIdent(n)

		case *ast.BasicLit:
			state = resolveBasicLit(n)

		default:
			if state == ExpectingKey || state == ExpectingPath {
				handleError(node, errorBadType)
			}

			state = NilState
		}

		return true
	})

	if errors.Error() != "" {
		return resources, errors
	}

	return resources, nil
}

// CorrectlyPathResources takes a collection of resources that have relative
// paths, and fills them in with their full path based on the parent package
// of the file they were pulled out of.
func CorrectlyPathResources(pkgPath string, base []Resource) []Resource {
	var resources []Resource

	for _, res := range base {
		resources = append(resources, Resource{
			Key:  res.Key,
			Path: filepath.Join(pkgPath, res.Path),
		})
	}

	return resources
}

// GetResourcesInFile will parse through a file, and identify all calls to
// Resource(), it will return a slice of Boxes so that Zap and pack these into
// Go source.
func GetResourcesInFile(fpath string) ([]Resource, error) {
	var resources []Resource

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, fpath, nil, 0)
	if err != nil {
		return nil, err
	}

	importName := getZappedImportName(f)
	if importName == "" || importName == "_" || importName == "." {
		return resources, nil
	}

	return parse(f, fset, importName)
}
