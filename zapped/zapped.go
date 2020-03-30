package zapped

import "fmt"

// A Box represents a directory that has been embedded into Go source code.
type Box struct {
	files map[string][]byte
}

// We create a map here then define all the elements in the generated file.
var boxes map[string]Box

// OpenBox reads through all the embedded boxes and looks for one with the
// provided directory. If one with the provided name could not be found, an
// error will be returned.
func OpenBox(directory string) (Box, error) {
	box, ok := boxes[directory]
	if !ok {
		err := fmt.Errorf("box with name %s could not be found", directory)
		return box, err
	}

	return box, nil
}

// File will returned the named file from the Box, or return an error if the
// named file could not be found.
func (box *Box) File(name string) ([]byte, error) {
	file, ok := box.files[name]
	if !ok {
		err := fmt.Errorf("file with name %s could not be found", file)
		return nil, err
	}

	return file, nil
}
