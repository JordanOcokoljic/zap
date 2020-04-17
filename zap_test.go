package zap

import (
	"errors"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// assertStringSliceMatch will assert the actual provided string slice matches
// the expected one. Currently uses reflect.DeepEqual.
func assertStringSliceMatch(t *testing.T, expected, actual []string) {
	t.Helper()

	if len(expected)+len(actual) == 0 {
		return
	}

	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Expected %v got %v", expected, actual)
	}
}

// assertString will assert that the actual string matches the expected string.
func assertString(t *testing.T, expected, actual string) {
	t.Helper()
	if expected != actual {
		t.Errorf("Expected %s got %s", expected, actual)
	}
}

// parseGo will parse a provided string of Go code and parse it, returning the
// ast.File and token.FileSet. If an error occurs in this process, the test is
// failed.
func parseGo(t *testing.T, src string) (*ast.File, *token.FileSet) {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "main.go", src, 0)
	if err != nil {
		t.Fatalf(err.Error())
	}

	return file, fset
}

// assertResourceSliceMatch will assert that the actual slice of resources
// matches the expected one.
func assertResourceSliceMatch(t *testing.T, exp, act []Resource) {
	t.Helper()

	if len(exp) != len(act) {
		t.Error("Slices did not match")
		return
	}

	for i := range exp {
		expected := exp[i]
		actual := act[i]

		if expected.Key != actual.Key || expected.Path != actual.Path {
			t.Error("Slices did not match")
			return
		}
	}
}

// assertInt will assert that the actual int provided matches the expected one.
func assertInt(t *testing.T, expected, actual int) {
	if expected != actual {
		t.Errorf("expected %d, but got %d", expected, actual)
	}
}

// getWd will return the current working directory the test resides in, if an
// error occurs as the working directory is fetched, the test is failed.
func getWd(t *testing.T) string {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf(err.Error())
	}

	return wd
}

// endIfFailed will check if the test has been marked as failing, and terminate
// it early if it has.
func endIfFailed(t *testing.T) {
	if t.Failed() {
		t.FailNow()
	}
}

func TestGetPackagesInProject(t *testing.T) {
	pkgs, err := GetPackagesInProject(".")
	if err != nil {
		t.Fatal(err.Error())
	}

	var names []string
	for _, pkg := range pkgs {
		names = append(names, pkg.Name)
	}

	assertStringSliceMatch(t, []string{"zap", "main", "zapped"}, names)
}

func TestGetZappedImportName(t *testing.T) {
	tests := []struct {
		code     string
		expected string
	}{
		{"package zap\n\nimport \"zapped\"", "zapped"},
		{"package zap\n\nimport . \"zapped\"", "."},
		{"package zap\n\nimport _\"zapped\"", "_"},
		{"package zap\n\nimport \"testing\"", ""},
	}

	for _, test := range tests {
		t.Run(test.expected, func(s *testing.T) {
			file, _ := parseGo(s, test.code)
			name := getZappedImportName(file)
			assertString(s, test.expected, name)
		})
	}
}

func TestAggregateError(t *testing.T) {
	var ag aggregateError

	assertString(t, "", ag.Error())

	ag.Add(errors.New("Test 1"))
	assertString(t, "Test 1", ag.Error())

	ag.Add(errors.New("Test 2"))
	assertString(t, "Test 1\nTest 2", ag.Error())
}

func TestGenerateParserError(t *testing.T) {
	code := "package main"

	f, fset := parseGo(t, code)

	err := generateParseError(fset, f.Pos(), errorUnknownCall)
	expected := "main.go:1:1: expected Resource() but was something else"
	assertString(t, expected, err.Error())

	err = generateParseError(fset, f.Pos(), errorBadType)
	expected = "main.go:1:1: calls to Resource() require string literals"
	assertString(t, expected, err.Error())
}

func TestParse(t *testing.T) {
	tests := []struct {
		name              string
		err               string
		expectedResources []Resource
		code              string
	}{
		{
			name: "GoodSample",
			err:  "",
			expectedResources: []Resource{
				{Key: "A", Path: "scripts/"},
				{Key: "B", Path: "sql/"},
			},
			code: `
package test

import "zapped"

func main() {
	zapped.Resource("A", "scripts/")
	zapped.Resource("B", "sql/")
}`,
		},
		{
			name: "WithRenamedImport",
			err:  "",
			expectedResources: []Resource{
				{Key: "A", Path: "scripts/"},
				{Key: "B", Path: "sql/"},
			},
			code: `
package test

import z "zapped"

func main() {
	z.Resource("A", "scripts/")
	z.Resource("B", "sql/")
}`,
		},
		{
			name: "WithIdentKeyInsteadOfLiteral",
			err:  "main.go:8:18: calls to Resource() require string literals",
			expectedResources: []Resource{
				{Key: "B", Path: "sql/"},
			},
			code: `
package test

import "zapped"

func main() {
	key := "A"

	zapped.Resource(key, "scripts/")
	zapped.Resource("B", "sql/")
}`,
		},
		{
			name: "WithIdentPathInsteadOfLiteral",
			err:  "main.go:8:23: calls to Resource() require string literals",
			expectedResources: []Resource{
				{Key: "A"},
				{Key: "B", Path: "sql/"},
			},
			code: `
package test

import "zapped"

func main() {
	key := "scripts/"

	zapped.Resource("A", key)
	zapped.Resource("B", "sql/")
}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(s *testing.T) {
			src := strings.TrimSpace(test.code)
			f, fset := parseGo(s, src)

			imp := getZappedImportName(f)
			resources, err := parse(f, fset, imp)

			if test.err == "" && err != nil {
				msg := "an error occured and isn't expected\n%s"
				s.Errorf(msg, err.Error())
			}

			if test.err != "" {
				assertString(s, test.err, err.Error())
			}

			assertResourceSliceMatch(s, test.expectedResources, resources)
		})
	}
}

func TestCorrectlyPathResources(t *testing.T) {
	resources := []Resource{
		{Key: "KEY1", Path: "scripts/"},
		{Key: "KEY2", Path: "sql/"},
		{Key: "KEY3", Path: "html/"},
	}

	fixed := CorrectlyPathResources("project/", resources)

	expected := []Resource{
		{Key: "KEY1", Path: "project/scripts"},
		{Key: "KEY2", Path: "project/sql"},
		{Key: "KEY3", Path: "project/html"},
	}

	assertResourceSliceMatch(t, expected, fixed)
}

func TestGetResourcesInPackage(t *testing.T) {
	pkg, err := build.ImportDir("testdata/", 0)
	if err != nil {
		t.Fatalf("an error occured: %s", err.Error())
	}

	resources, err := GetResourcesInPackage(pkg)
	if err != nil {
		t.Fatalf("an error occured: %s", err.Error())
	}

	expected := []Resource{
		{Key: "KEY", Path: "testdata/PATH"},
	}

	assertResourceSliceMatch(t, expected, resources)
}

func TestEmbedDirectories(t *testing.T) {
	// Because this test interacts with the filesystem, these ensure that the
	// test will use the correct files and have the correct paths, no matter
	// which machine it is being run from.
	wd := getWd(t)
	rel := func(path string) string {
		return filepath.Join(wd, path)
	}

	dirs, err := EmbedDirectories([]Resource{{"A", rel("testdata")}})
	if err != nil {
		t.Fatalf("an error occured: %s", err.Error())
	}

	// If it didn't manage to get all the directories, fail it now to avoid
	// having memory panics.
	assertInt(t, 3, len(dirs))
	endIfFailed(t)

	type file struct {
		name string
		body string
	}

	tests := []struct {
		name    string
		path    string
		subdirs []string
		files   []file
	}{
		{
			name:    "testdata",
			path:    rel("testdata"),
			subdirs: []string{rel("testdata/accounting")},
			files: []file{
				{
					name: "testdata.go",
					body: `
package testdata

import (
	"zap/zapped"
)

func main() {
	zapped.Resource("KEY", "PATH/")
}
`,
				},
			},
		},
		{
			name:    "accounting",
			path:    rel("testdata/accounting"),
			subdirs: []string{rel("testdata/accounting/clients")},
			files: []file{
				{
					name: "data.txt",
					body: `
AccountName: jordanockoljic
Balance: 143.50`,
				},
			},
		},
		{
			name:    "clients",
			path:    rel("testdata/accounting/clients"),
			subdirs: []string{},
			files: []file{
				{
					name: "a.txt",
					body: `
AccountName: A
Balance: 243512.34`,
				},
				{
					name: "b.txt",
					body: `
AccountName: B
Balance: 748362.34`,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(s *testing.T) {
			dir := dirs[test.path]

			// Check if the number of files embedded, matches the number of
			// files expected to be embedded.
			assertInt(s, len(test.files), len(dir.Files))
			endIfFailed(s)

			// Check that the subdirectories that have been embedded are the
			// ones we expect.
			assertStringSliceMatch(t, test.subdirs, dir.SubDirs)

			// Check that the files are what we expect.
			for _, file := range test.files {
				expected := strings.TrimLeft(file.body, "\n")

				var actual string
				if emb, ok := dir.Files[file.name]; ok {
					actual = string(emb)
				}

				assertString(s, expected, actual)
			}
		})
	}
}

func TestGenerateCode(t *testing.T) {
	expTmpl := `
package zapped

func init() {
	// %PROJECTPATH%/testdata/accounting/clients
	_7c0f43becbec3fcc411fb2eb4cf6781c8938c80c := Directory{
		directories: make(map[string]*Directory),
		files:       make(map[string]File),
	}

	_7c0f43becbec3fcc411fb2eb4cf6781c8938c80c.files["a.txt"] = []byte{0x41, 0x63, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x4e, 0x61, 0x6d, 0x65, 0x3a, 0x20, 0x41, 0xa, 0x42, 0x61, 0x6c, 0x61, 0x6e, 0x63, 0x65, 0x3a, 0x20, 0x32, 0x34, 0x33, 0x35, 0x31, 0x32, 0x2e, 0x33, 0x34}
	_7c0f43becbec3fcc411fb2eb4cf6781c8938c80c.files["b.txt"] = []byte{0x41, 0x63, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x4e, 0x61, 0x6d, 0x65, 0x3a, 0x20, 0x42, 0xa, 0x42, 0x61, 0x6c, 0x61, 0x6e, 0x63, 0x65, 0x3a, 0x20, 0x37, 0x34, 0x38, 0x33, 0x36, 0x32, 0x2e, 0x33, 0x34}

	// %PROJECTPATH%/testdata/accounting
	_fdcf5ee62ffce988f4735ffa040bff452a5d3e46 := Directory{
		directories: make(map[string]*Directory),
		files:       make(map[string]File),
	}

	_fdcf5ee62ffce988f4735ffa040bff452a5d3e46.directories["clients"] = &_7c0f43becbec3fcc411fb2eb4cf6781c8938c80c
	_fdcf5ee62ffce988f4735ffa040bff452a5d3e46.files["data.txt"] = []byte{0x41, 0x63, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x4e, 0x61, 0x6d, 0x65, 0x3a, 0x20, 0x6a, 0x6f, 0x72, 0x64, 0x61, 0x6e, 0x6f, 0x63, 0x6b, 0x6f, 0x6c, 0x6a, 0x69, 0x63, 0xa, 0x42, 0x61, 0x6c, 0x61, 0x6e, 0x63, 0x65, 0x3a, 0x20, 0x31, 0x34, 0x33, 0x2e, 0x35, 0x30}

	// %PROJECTPATH%/testdata
	_4d446d3a99d94af0c61e8be1147dc81ff7df9175 := Directory{
		directories: make(map[string]*Directory),
		files:       make(map[string]File),
	}

	_4d446d3a99d94af0c61e8be1147dc81ff7df9175.directories["accounting"] = &_fdcf5ee62ffce988f4735ffa040bff452a5d3e46
	_4d446d3a99d94af0c61e8be1147dc81ff7df9175.files["testdata.go"] = []byte{0x70, 0x61, 0x63, 0x6b, 0x61, 0x67, 0x65, 0x20, 0x74, 0x65, 0x73, 0x74, 0x64, 0x61, 0x74, 0x61, 0xa, 0xa, 0x69, 0x6d, 0x70, 0x6f, 0x72, 0x74, 0x20, 0x28, 0xa, 0x9, 0x22, 0x7a, 0x61, 0x70, 0x2f, 0x7a, 0x61, 0x70, 0x70, 0x65, 0x64, 0x22, 0xa, 0x29, 0xa, 0xa, 0x66, 0x75, 0x6e, 0x63, 0x20, 0x6d, 0x61, 0x69, 0x6e, 0x28, 0x29, 0x20, 0x7b, 0xa, 0x9, 0x7a, 0x61, 0x70, 0x70, 0x65, 0x64, 0x2e, 0x52, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x28, 0x22, 0x4b, 0x45, 0x59, 0x22, 0x2c, 0x20, 0x22, 0x50, 0x41, 0x54, 0x48, 0x2f, 0x22, 0x29, 0xa, 0x7d, 0xa}
}
`

	wd := getWd(t)
	expected := strings.ReplaceAll(
		strings.TrimLeft(expTmpl, "\n"),
		"%PROJECTPATH%", wd)

	resources := []Resource{
		{Key: "F", Path: filepath.Join(wd, "testdata")},
	}

	dirs, err := EmbedDirectories(resources)
	if err != nil {
		t.Fatal(err.Error())
	}

	code, err := GenerateCode(dirs)
	if err != nil {
		t.Fatal(err.Error())
	}

	assertString(t, expected, code)
}
