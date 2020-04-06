package main

import (
	"fmt"
	"os"
	"zap"
)

func main() {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	pkgs, err := zap.GetPackagesInProject(wd)
	if err != nil {
		panic(err)
	}

	for _, pkg := range pkgs {
		fmt.Println(pkg.Dir)
	}
}
