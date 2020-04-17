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
			"an error occured while getting the working directory: %s\n",
			err.Error())

		os.Exit(1)
	}

	// Check if the zapped directory exists, if it doesn't, create it. Also
	// record if the directory was created, because this indicates if it was
	// the first time Zap was run in this project.
	firstRun := false
	zappedPath := filepath.Join(wd, "zapped")
	if _, err := os.Stat(zappedPath); os.IsNotExist(err) {
		os.Mkdir(zappedPath, os.ModePerm)
		firstRun = true
	}

	// Refresh the zapped library with the most recent version.
	zappedResource, err := zapped.Resource("ZAP_RESOURCE", "../zapped")
	if err != nil {
		fmt.Printf(
			"an error occured while accessing the zapped resource: %s\n",
			err.Error(),
		)

		os.Exit(1)
	}

	zappedLib, err := zappedResource.File("zapped.go")
	if err != nil {
		fmt.Printf(
			"an error occured while reading zapped.go: %s\n",
			err.Error(),
		)

		os.Exit(1)
	}

	zappedLibPath := filepath.Join(zappedPath, "zapped.go")
	err = ioutil.WriteFile(zappedLibPath, zappedLib.Bytes(), 0666)
	if err != nil {
		fmt.Printf(
			"an error occured while writing zapped.go: %s\n",
			err.Error(),
		)

		os.Exit(1)
	}

	// If this is the first time that Zap has been run, or if zap has been run
	// in devMode, terminate here - it isn't necessary to actually embed files
	// either way.
	if firstRun || *devMode {
		os.Exit(0)
	}

	// Get all the packages in the project.
	packages, err := zap.GetPackagesInProject(wd)
	if err != nil {
		fmt.Printf(
			"an error occured while getting packages in project: %s\n",
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
				"an error occured while getting resources in package %s: %s\n",
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
			"an error occured while embedding resources: %s\n",
			err.Error(),
		)

		os.Exit(1)
	}

	// Generate the code.
	code, err := zap.GenerateCode(embeddedDirectories, *devMode)
	if err != nil {
		fmt.Printf(
			"an error occured while generating code: %s\n",
			err.Error(),
		)

		os.Exit(1)
	}

	// Write the code to the file
	embedPath := filepath.Join(zappedPath, "zap.embed.go")
	err = ioutil.WriteFile(embedPath, code, 0666)
	if err != nil {
		fmt.Printf(
			"an error occured while writing code: %s\n",
			err.Error(),
		)

		os.Exit(1)
	}
}
