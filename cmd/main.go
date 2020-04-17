package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"zap"
	"zap/zapped"
)

func main() {
	// Setup development mode flag.
	var devMode = flag.Bool(
		"devMode",
		false,
		"whether or not zapped should run in development mode.",
	)

	flag.Parse()

	// Get the working directory of the program.
	wd, err := os.Getwd()
	if err != nil {
		fmt.Printf(
			"an error occured while getting the working directory: %s",
			err.Error())

		os.Exit(1)
	}

	// Check if the zapped directory exists, if it doesn't, create it. Relies
	// on the fact that if the directory already exists, Mkdir is a no-op.
	zappedPath := filepath.Join(wd, "zapped")
	os.Mkdir(zappedPath, os.ModePerm)

	// Refresh the zapped library with the most recent version.
	zappedResource, err := zapped.Resource("ZAP_RESOURCE", "../zapped")
	if err != nil {
		fmt.Printf(
			"an error occured while accessing the zapped resource: %s",
			err.Error(),
		)

		os.Exit(1)
	}

	zappedLib, err := zappedResource.File("zapped.go")
	if err != nil {
		fmt.Printf(
			"an error occured while reading zapped.go: %s",
			err.Error(),
		)

		os.Exit(1)
	}

	zappedLibPath := filepath.Join(zappedPath, "zapped.go")
	err = ioutil.WriteFile(zappedLibPath, zappedLib.Bytes(), os.ModePerm)
	if err != nil {
		fmt.Printf(
			"an error occured while writing zapped.go: %s",
			err.Error(),
		)

		os.Exit(1)
	}

	// Get all the packages in the project.
	packages, err := zap.GetPackagesInProject(wd)
	if err != nil {
		fmt.Printf(
			"an error occured while getting packages in project: %s",
			err.Error(),
		)

		os.Exit(1)
	}

	// Get resources in all the packages.
	var resources []zap.Resource
	for _, pkg := range packages {
		packageResources, err := zap.GetResourcesInPackage(pkg)
		if err != nil {
			fmt.Printf(
				"an error occured while getting resources in package %s: %s",
				pkg.Name,
				err.Error(),
			)

			os.Exit(1)
		}

		resources = append(resources, packageResources...)
	}

	// Embed the directories.
	embeddedDirectories, err := zap.EmbedDirectories(resources)
	if err != nil {
		fmt.Printf(
			"an error occured while embedding resources: %s",
			err.Error(),
		)

		os.Exit(1)
	}

	// Generate the code.
	code, err := zap.GenerateCode(embeddedDirectories, *devMode)
	if err != nil {
		fmt.Printf(
			"an error occured while generating code: %s",
			err.Error(),
		)

		os.Exit(1)
	}

	// Write the code to the file
	embedPath := filepath.Join(zappedPath, "zap.embed.go")
	err = ioutil.WriteFile(embedPath, code, os.ModePerm)
	if err != nil {
		fmt.Printf(
			"an error occured while writing code: %s",
			err.Error(),
		)

		os.Exit(1)
	}
}
