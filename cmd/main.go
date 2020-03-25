package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	var action string
	if len(os.Args) >= 2 {
		action = os.Args[1]
	}

	switch action {
	case "init":
	case "build":
	case "stub":
	case "help", "":
		usage()
	default:
		fmt.Printf("option: %s not recognised\n", action)
	}
}

func usage() {
	fmt.Println(strings.TrimSpace(`
usage: zap <action>

There are three recognised actions in zap:

init	Creates the root level package with functions for being able to
	read data from the embedded files.
	
build	Scans the source for calls to the library functions, and reads the
	files and embeds the byte representation of those files into go 
	source code so they can be built into an executable.

stub	Scans the source for calls to the library functions, and generates
	functions that read the contents of those files as they exist on
	the file system for testing and development.

help	Shows this manual.
`))

	os.Exit(0)
}
