package zap

import (
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
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
	file, err := parser.ParseFile(fset, "", src, 0)
	if err != nil {
		t.Fatalf(err.Error())
	}

	return file, fset
}

func TestGetPackagesInProject(t *testing.T) {
	pkgs, err := GetPackagesInProject(".", []string{".git"})
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
