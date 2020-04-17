package zap

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"go/ast"
	"go/build"
	"go/format"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
)

// Resource is used to track each unique Key passed to a call to Resource() and
// the path specified in the call.
type Resource struct {
	Key  string
	Path string
}

// aggregateError is a collection of errors that fullfils the error interface,
// so it can be passed up, while still containing all the information.
type aggregateError struct {
	errors []error
}

// Add adds a new error to the list of errors.
func (ag *aggregateError) Add(err error) {
	ag.errors = append(ag.errors, err)
}

// SafeReturn will check if there are any errors in the slice and return the
// error if there are. If there aren't, nil will be returned.
func (ag aggregateError) SafeReturn() error {
	if len(ag.errors) != 0 {
		return ag
	}

	return nil
}

// Returns the value of the aggregateError, which is just a big collection of
// all the other errors. Fulfils the Error interface.
func (ag aggregateError) Error() string {
	var str string

	for i, err := range ag.errors {
		if i == 0 {
			str = err.Error()
			continue
		}

		str = fmt.Sprintf("%s\n%s", str, err.Error())
	}

	return str
}

// Directory represents an embedded directory. Only the absolute paths of the
// subdirectories are stored so that they are not embedded mulitple times.
type Directory struct {
	SubDirs []string
	Files   map[string][]byte
}

// GetPackagesInProject will return the package in the current directory, as
// well as all the subdirectories under it. Under the hood it uses Walk, so
// it won't follow symbolic links.
func GetPackagesInProject(wd string) ([]*build.Package, error) {
	var packages []*build.Package

	fn := func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			return nil
		}

		// Respecting the Go tools ignoring this directory.
		if info.Name() == "testdata" {
			return filepath.SkipDir
		}

		pkg, err := build.ImportDir(path, 0)

		if err != nil {
			if !strings.HasPrefix(err.Error(), "no buildable Go source") {
				return err
			}

			return filepath.SkipDir
		}

		packages = append(packages, pkg)

		return nil
	}

	err := filepath.Walk(wd, fn)
	return packages, err
}

// getZappedImportName returns the name the Zapped library was imported under,
// returning empty string if it is not imported.
func getZappedImportName(file *ast.File) string {
	var name string

	for _, imp := range file.Imports {
		if strings.HasSuffix(imp.Path.Value, "zapped\"") {
			name = "zapped"

			if imp.Name != nil {
				name = imp.Name.Name
			}

			break
		}
	}

	return name
}

// Track the types of errors that could occur so that generateParseError knows
// what message to use.
const (
	errorUnknownCall uint8 = iota
	errorBadType
)

// generateParseError will return an error with correct formatting describing
// what was incorrect about the scanned source.
func generateParseError(fset *token.FileSet, p token.Pos, err uint8) error {
	pos := fset.Position(p)
	filename := pos.Filename

	// Reduces unecessary duplication of code.
	format := func(msg string) string {
		return fmt.Sprintf("%s:%d:%d: %s", filename, pos.Line, pos.Column, msg)
	}

	var msg string
	switch err {
	case errorUnknownCall:
		msg = format("expected Resource() but was something else")
	case errorBadType:
		msg = format("calls to Resource() require string literals")
	}

	return fmt.Errorf(msg)
}

// parse will walk the AST and identify calls to Resource() and extract
// the key and the path from them. It will return an error
func parse(f *ast.File, fset *token.FileSet, imp string) ([]Resource, error) {
	var resources []Resource
	var errors aggregateError

	// State based parsing let's us solve this problem without needing to have
	// a million different variables tracking everything.
	var state uint8
	const (
		NilState uint8 = iota
		ExpectingResourceCall
		ExpectingKey
		ExpectingPath
	)

	// Error handling code being pulled out into it's own closure reduces
	// repetition of simple code.
	handleError := func(node ast.Node, errorType uint8) {
		err := generateParseError(fset, node.Pos(), errorType)
		errors.Add(err)
	}

	// Pulling the code out of the switch improves the readability, and ensures
	// clarity of logic.
	resolveIdent := func(node *ast.Ident) uint8 {
		switch state {
		case NilState:
			if node.Name == imp {
				return ExpectingResourceCall
			}

		case ExpectingResourceCall:
			if node.Name != "Resource" {
				handleError(node, errorUnknownCall)
				return NilState
			}

			return ExpectingKey

		case ExpectingKey, ExpectingPath:
			handleError(node, errorBadType)
			return NilState
		}

		return state
	}

	resolveBasicLit := func(node *ast.BasicLit) uint8 {
		switch state {
		case ExpectingKey:
			if node.Kind != token.STRING {
				handleError(node, errorBadType)
				return NilState
			}

			res := Resource{Key: node.Value[1 : len(node.Value)-1]}
			resources = append(resources, res)
			return ExpectingPath

		case ExpectingPath:
			if node.Kind != token.STRING {
				handleError(node, errorBadType)
				return NilState
			}

			res := &resources[len(resources)-1]
			res.Path = node.Value[1 : len(node.Value)-1]
			return NilState
		}

		return state
	}

	// Walk the AST.
	ast.Inspect(f, func(node ast.Node) bool {
		if node == nil {
			return false
		}

		switch n := node.(type) {
		case *ast.Ident:
			state = resolveIdent(n)

		case *ast.BasicLit:
			state = resolveBasicLit(n)

		default:
			if state == ExpectingKey || state == ExpectingPath {
				handleError(node, errorBadType)
			}

			state = NilState
		}

		return true
	})

	return resources, errors.SafeReturn()
}

// GetResourcesInFile will parse through a file, and identify all calls to
// Resource(), it will return a slice of Boxes so that Zap and pack these into
// Go source.
func GetResourcesInFile(fpath string) ([]Resource, error) {
	var resources []Resource

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, fpath, nil, 0)
	if err != nil {
		return nil, err
	}

	importName := getZappedImportName(f)
	if importName == "" || importName == "_" || importName == "." {
		return resources, nil
	}

	return parse(f, fset, importName)
}

// CorrectlyPathResources takes a collection of resources that have relative
// paths, and fills them in with their full path based on the parent package
// of the file they were pulled out of.
func CorrectlyPathResources(pkgPath string, base []Resource) []Resource {
	var resources []Resource

	for _, res := range base {
		resources = append(resources, Resource{
			Key:  res.Key,
			Path: filepath.Join(pkgPath, res.Path),
		})
	}

	return resources
}

// GetResourcesInPackage will return a slice of Resources that are correctly
// pathed.
func GetResourcesInPackage(pkg *build.Package) ([]Resource, error) {
	var resources []Resource
	var errors aggregateError

	for _, file := range pkg.GoFiles {
		if file == "zap.embed.go" {
			continue
		}

		fpath := filepath.Join(pkg.Dir, file)
		res, err := GetResourcesInFile(fpath)
		if err != nil {
			errors.Add(err)
			continue
		}

		resources = append(resources, CorrectlyPathResources(pkg.Dir, res)...)
	}

	return resources, errors.SafeReturn()
}

// EmbedDirectories will return a map of directories containg the contents of
// the files within them.
func EmbedDirectories(resources []Resource) (map[string]*Directory, error) {
	var errors aggregateError

	dirs := make(map[string]*Directory)

	var dfn func(string) (*Directory, error)
	dfn = func(dpath string) (*Directory, error) {
		var dnfErrors aggregateError
		dir := Directory{}
		dir.Files = make(map[string][]byte)

		files, err := ioutil.ReadDir(dpath)
		if err != nil {
			return nil, err
		}

		for _, file := range files {
			// In case someone has version controlled the folder they store
			// embeddable assets in - stops the tool getting stuck on this
			// potentially massive file.
			if file.Name() == ".git" || file.Name() == "zap.embed.go" {
				continue
			}

			fpath := filepath.Join(dpath, file.Name())

			switch file.IsDir() {
			case true:
				subdir, err := dfn(fpath)
				if err != nil {
					dnfErrors.Add(err)
					continue
				}

				dirs[fpath] = subdir
				dir.SubDirs = append(dir.SubDirs, fpath)
			case false:
				bytes, err := ioutil.ReadFile(fpath)
				if err != nil {
					dnfErrors.Add(err)
					continue
				}

				dir.Files[file.Name()] = bytes
			}
		}

		return &dir, dnfErrors.SafeReturn()
	}

	for _, res := range resources {
		if _, exists := dirs[res.Path]; exists {
			continue
		}

		dir, err := dfn(res.Path)
		if err != nil {
			errors.Add(err)
			continue
		}

		dirs[res.Path] = dir
	}

	return dirs, errors.SafeReturn()
}

// GenerateCode will return a slice of bytes containing the code that should be
// written so that file contents can be accessed within the binary. The output
// has been run through the Go formatter.
func GenerateCode(dirs map[string]*Directory, devMode bool) ([]byte, error) {
	var buf bytes.Buffer
	var sortedDirs []string
	var errors aggregateError
	hashMap := make(map[string]string)

	for dpath := range dirs {
		sortedDirs = append(sortedDirs, dpath)
	}

	sort.Slice(sortedDirs, func(i, j int) bool {
		ic := strings.Count(sortedDirs[i], string(os.PathSeparator))
		jc := strings.Count(sortedDirs[j], string(os.PathSeparator))
		return ic > jc
	})

	type TmplDir struct {
		Name  string
		Hash  string
		Files map[string][]byte
		Dirs  map[string]string
	}

	type TmplData struct {
		DevMode bool
		Dirs    []TmplDir
	}

	tmplData := TmplData{DevMode: devMode}

	for _, path := range sortedDirs {
		dir := dirs[path]
		hash := fmt.Sprintf("_%x", sha1.Sum([]byte(path)))
		hashMap[path] = hash

		dt := TmplDir{
			Name:  path,
			Hash:  hash,
			Files: dir.Files,
			Dirs:  make(map[string]string),
		}

		for _, subd := range dir.SubDirs {
			tpath, err := filepath.Rel(path, subd)
			if err != nil {
				errors.Add(err)
				continue
			}

			dt.Dirs[tpath] = hashMap[subd]
		}

		tmplData.Dirs = append(tmplData.Dirs, dt)
	}

	tmpl := template.Must(template.New("tmpl").Parse(strings.TrimSpace(`
package zapped
	
func init() {
	developmentMode = {{ printf "%t" .DevMode }}

{{ range $dir := .Dirs }}
	// {{ $dir.Name }}
	{{ $dir.Hash }} := Directory{
		directories: make(map[string]*Directory),
		files: make(map[string]File),
	}
	{{ range $path, $hash := $dir.Dirs }}
	{{ $dir.Hash }}.directories["{{ $path }}"] = &{{ $hash }}
	{{- end -}}
	{{ range $name, $body := $dir.Files }}
	{{ $dir.Hash }}.files["{{ $name }}"] = File{ {{ printf "%#v" $body }} }
	{{- end }}
{{ end -}}
}
`)))

	tmpl.Execute(&buf, tmplData)

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		errors.Add(err)
	}

	return formatted, errors.SafeReturn()
}
