package zap

import (
	"errors"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"reflect"
	"strings"
	"testing"
)

func assertStringSliceMatch(t *testing.T, expected []string, actual []string) {
	t.Helper()
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Expected %v got %v", expected, actual)
	}
}

func assertString(t *testing.T, expected string, actual string) {
	t.Helper()
	if expected != actual {
		t.Errorf("Expected %s got %s", expected, actual)
	}
}

func parseGo(t *testing.T, src string) (*ast.File, *token.FileSet) {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "main.go", src, 0)
	if err != nil {
		t.Fatalf(err.Error())
	}

	return file, fset
}

func assertResourceSliceMatch(t *testing.T, exp []Resource, act []Resource) {
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
		{Key: "KEY1", Path: "project/scripts/"},
		{Key: "KEY2", Path: "project/sql/"},
		{Key: "KEY3", Path: "project/html/"},
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
		{Key: "KEY", Path: "testdata/PATH/"},
	}

	assertResourceSliceMatch(t, expected, resources)
}
