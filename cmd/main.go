package main

import (
	"fmt"
	"os"
	"strings"
	"zap"
)

func main() {
	var action string
	if len(os.Args) >= 2 {
		action = os.Args[1]
	}

	wd, err := os.Getwd()
	if err != nil {
		fmt.Printf("error getting current directory: %s", err.Error())
		os.Exit(1)
	}

	err = run(action, wd)
	if err != nil {
		fmt.Printf("an error occured: %s", err.Error())
		os.Exit(1)
	}
}

// run takes the action provided in the command line, and the current working
// directory as determined by the program and executes the specified operation.
func run(action string, wd string) error {
	var err error

	switch action {
	case "init":
		err = zap.OperationInit(wd)
	case "build":
		err = zap.OperationBuild(wd)
	case "stub":
		err = zap.OperationStub(wd)
	case "help", "":
		usage()
	default:
		err = fmt.Errorf("action %s not recognized", action)
	}

	return err
}

// usage prints out the usage guide explaining how Zap should be called.
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
