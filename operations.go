package zap

// OperationInit creates a new folder containg the zap library for use within
// projects that use the tool.
func OperationInit(path string) error {
	return nil
}

// OperationBuild walks through the source code in the project, and identifies
// the directories that need to be embedded into Boxes, collects all the files
// within them and makes them accessible to the library created so the project
// can be built.
func OperationBuild(path string) error {
	return nil
}

// OperationStub creates stub functions that allow calls from the library that
// just reach into the filesystem, rather than looking for the embedded files
// for testing and development purposes.
func OperationStub(path string) error {
	return nil
}
