package zapped

import "fmt"

// A File represents an embedded file.
type File struct {
	contents []byte
}

// Bytes return the contents of the file as a byte slice.
func (file *File) Bytes() []byte {
	return file.contents
}

// String returns the contents of the file as a string.
func (file *File) String() string {
	return string(file.contents)
}

// A Directory represents an embedded directory.
type Directory struct {
	files       map[string]*File
	directories map[string]*Directory
}

// File searches for a File with a name that matches the provided one. If a
// file with the provided name cannot be found, an error will be returned.
func (dir *Directory) File(name string) (*File, error) {
	file, ok := dir.files[name]

	if !ok {
		err := fmt.Errorf("a file with name %s could not be found", name)
		return nil, err
	}

	return file, nil
}

// Directory searches for a Directory with a name that matches the provided
// one. If a directory with a matching name cannot be found, an error will be
// returned.
func (dir *Directory) Directory(name string) (*Directory, error) {
	dir, ok := dir.directories[name]

	if !ok {
		err := fmt.Errorf("a directory with name %s could not be found", name)
		return nil, err
	}

	return dir, nil
}

// resources is used to store all of the directories that are embedded within
// the application. The string used to refer to them is the Key provided to the
// call to Resource().
var resources = make(map[string]*Directory)

// Resource attempts to locate an embedded resource with the provided key, and
// return it. If the resource cannot be found, an error will be returned.
func Resource(key string, dir string) (*Directory, error) {
	res, ok := resources[key]

	if !ok {
		return nil, fmt.Errorf("resource with key %s could not be found", key)
	}

	return res, nil
}
