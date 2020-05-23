// This file is part of Zap, a tool for embedding files into Go source.
// Copyright (C) 2020 Jordan Ocokoljic.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

// As an exception, you may distribute programs that contain code generated
// with or copied into by this program under terms of your choice.

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

// Files returns the names of all files embedded into the Directory sorted in
// alphabetical order.
func (dir *Directory) Files() []string {
	var files []string

	for fname := range dir.files {
		files = append(files, fname)
	}

	return files
}

// Directories returns the names of all directories embedded into the Directory
// sorted in alphabetical order.
func (dir *Directory) Directories() []string {
	var directories []string

	for dname := range dir.directories {
		directories = append(directories, dname)
	}

	return directories
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
