package zapped

import (
	"fmt"
	"io/ioutil"
	"path"
	"path/filepath"
	"runtime"
)

// developmentMode indicates if the library has been run with the --dev flag,
// which should allow the files to be read from the filesystem rather than from
// the embedded source.
var developmentMode = true

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
	directories map[string]*Directory
	files       map[string]File
	devPath     string
}

// File searches for a File with a name that matches the provided one. If a
// file with the provided name cannot be found, an error will be returned.
func (dir *Directory) File(name string) (File, error) {
	var file File

	switch developmentMode {
	case false:
		f, ok := dir.files[name]

		if !ok {
			err := fmt.Errorf("a file with name %s could not be found", name)
			return File{}, err
		}

		file = f
	case true:
		fpath := filepath.Join(dir.devPath, name)
		bytes, err := ioutil.ReadFile(fpath)

		if err != nil {
			return File{}, err
		}

		file = File{contents: bytes}
	}

	return file, nil
}

// Directory searches for a Directory with a name that matches the provided
// one. If a directory with a matching name cannot be found, an error will be
// returned.
func (dir *Directory) Directory(name string) (*Directory, error) {
	var directory *Directory

	switch developmentMode {
	case false:
		dir, ok := dir.directories[name]

		if !ok {
			err := fmt.Errorf(
				"a directory with name %s could not be found",
				name)

			return nil, err
		}

		directory = dir
	case true:
		directory = &Directory{devPath: filepath.Join(dir.devPath, name)}
	}

	return directory, nil
}

// resources is used to store all of the directories that are embedded within
// the application. The string used to refer to them is the Key provided to the
// call to Resource().
var resources = make(map[string]*Directory)

// Resource attempts to locate an embedded resource with the provided key, and
// return it. If the resource cannot be found, an error will be returned.
func Resource(key string, dir string) (*Directory, error) {
	var resource *Directory

	switch developmentMode {
	case false:
		res, ok := resources[key]

		if !ok {
			return nil, fmt.Errorf(
				"resource with key %s could not be found",
				key)
		}

		resource = res
	case true:
		_, fn, _, ok := runtime.Caller(1)
		if !ok {
			return nil, fmt.Errorf(
				"was unable to get relative path for directory %s",
				dir)
		}

		resource = &Directory{devPath: filepath.Join(path.Dir(fn), dir)}
	}

	return resource, nil
}
