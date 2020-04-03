package zap_test

import (
	"reflect"
	"testing"
	"zap"
)

func AssertStringSliceMatch(t *testing.T, expected []string, actual []string) {
	t.Helper()
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Expected %v got %v", expected, actual)
	}
}

func TestGetPackagesInProject(t *testing.T) {
	pkgs, err := zap.GetPackagesInProject(".", []string{".git"})
	if err != nil {
		t.Fatal(err.Error())
	}

	var names []string
	for _, pkg := range pkgs {
		names = append(names, pkg.Name)
	}

	AssertStringSliceMatch(t, []string{"zap", "main", "zapped"}, names)
}
