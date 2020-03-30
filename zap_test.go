package zap

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func parseGoString(t *testing.T, code string) (*ast.File, *token.FileSet) {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", code, 0)
	if err != nil {
		t.Fatal(err.Error())
	}

	return f, fset
}

func assertString(t *testing.T, expected string, actual string) {
	t.Helper()

	if expected != actual {
		t.Errorf("Expected %s, but was %s", expected, actual)
	}
}

func TestCheckDirectoryExistsWithValidDirectory(t *testing.T) {
	result := checkDirectoryExists("testdata/testpackage")
	if result != true {
		t.Error("checkDirectoryExists returned false when directory existed")
	}
}

func TestCheckDirectoryExistsWithInvalidDirectory(t *testing.T) {
	result := checkDirectoryExists("testdata/notdir")
	if result != false {
		t.Error("checkDirectoryExists returned true when directory does not exist")
	}
}

func TestGetPackageInDirectory(t *testing.T) {
	pkg, err := getPackageInDirectory("testdata/testpackage")
	if err != nil {
		t.Fatal(err.Error())
	}

	if pkg.Name != "testpackage" && pkg.GoFiles[0] != "testpackage.go" {
		t.Error("getPackageInDirectory did not find correct package")
	}
}

func TestGetPackagesInProject(t *testing.T) {
	pkgs, err := getPackagesInProject("testdata/testpackage", nil)
	if err != nil {
		t.Fatal(err.Error())
	}

	if len(pkgs) != 2 {
		t.Fatal("getPackagesInProject did not find all packages in project")
	}

	if pkgs[0].Name != "testpackage" && pkgs[1].Name != "otherpackage" {
		t.Fatal("getPackagesInProject did not find the correct packages")
	}
}

func TestGetPackagesInProjectWithSkipList(t *testing.T) {
	pkgs, err := getPackagesInProject("testdata/testpackage", []string{"otherpackage"})
	if err != nil {
		t.Fatal(err.Error())
	}

	if len(pkgs) != 1 {
		t.Fatal("getPackagesInProject did not skip otherpackage")
	}

	if pkgs[0].Name != "testpackage" {
		t.Fatal("getPackagesInProject did not find the correct package")
	}
}

func TestGetZappedImportName(t *testing.T) {
	importTests := []struct {
		code     string
		expected string
	}{
		{"package sample\nimport (\n\"zapped\"\n)", "zapped"},
		{"package sample\nimport (\n. \"zapped\"\n)", "."},
		{"package sample\nimport (\n_ \"zapped\"\n)", "_"},
		{"package sample\nimport (\nz \"zapped\"\n)", "z"},
		{"package sample\nimport (\n\"os\"\n)", ""},
	}

	for _, subtest := range importTests {
		t.Run("Import "+subtest.expected, func(s *testing.T) {
			f, _ := parseGoString(s, subtest.code)
			importName := getZappedImportName(f)
			assertString(s, subtest.expected, importName)
		})
	}
}

func TestGetBoxesInFile(t *testing.T) {
	code := strings.TrimSpace(`
package test

import "zapped"

func Test() {
	zapped.OpenBox("a")
	zapped.OpenBox("b")
	zapped.OpenBox("c")
}
`)

	f, _ := parseGoString(t, code)
	boxes := getBoxesInFile(f, "zapped")
	assertString(t, "a", boxes[0])
	assertString(t, "b", boxes[1])
	assertString(t, "c", boxes[2])
}

func TestGetBoxesInFileWithDotImport(t *testing.T) {
	code := strings.TrimSpace(`
package test
	
import . "zapped"
	
func Test() {
	OpenBox("a")
	OpenBox("b")
	OpenBox("c")
}
`)

	f, _ := parseGoString(t, code)
	boxes := getBoxesInFile(f, ".")
	assertString(t, "a", boxes[0])
	assertString(t, "b", boxes[1])
	assertString(t, "c", boxes[2])
}

func TestGetBoxesInFileWithRenamedImport(t *testing.T) {
	code := strings.TrimSpace(`
package test
	
import z "zapped"
	
func Test() {
	z.OpenBox("a")
	z.OpenBox("b")
	z.OpenBox("c")
}
`)

	f, _ := parseGoString(t, code)
	boxes := getBoxesInFile(f, "z")
	assertString(t, "a", boxes[0])
	assertString(t, "b", boxes[1])
	assertString(t, "c", boxes[2])
}
