# Zap
Zap is a tool for embedding files into Go projects. During development, it can
read from the filesystem, so that you don't need to wait for files to be 
embedded to test your application.

## Building Zap
Zap requires Zap. To build Zap from source, clone the repo and then:
``` bash
go run cmd/main.go
go build -o zap cmd/main.go
```

## Usage
Using Zap is very simple, to get started, simply run `zap` in the package of
your project. Zap will then add the `zap` library to your project. 

You can then use the `zap.Resource` function to access resources from the file
system. Note that `zap.Resource` calls are relative to the package containing
the file from which the call is being made.

Once there are some calls to `zap.Resource`, running `zap` will go through and
parse the code within the packages in the project, extracing the keys and
the paths provided to these calls. Note that because the scanning takes place 
before execution, the arguments passed to calls to `zap.Resource` must be 
string literals, otherwise the tool cannot find which paths to embed. Once all
the calls and the parameters of these calls have been identified, the
directories specified in the calls are embedded into Go source and written to
a file named `zap.embed.go` that will be in the same directory as the `zap`
library.

If you want to run tests with Zap, or have it able to read from your filesystem
during development, you can ruin `zap` with the `-devMode` flag, which will
allow it to read files from the filesystem instead of the embedded files.

### Examples
Using Zap for the first time in a project:
``` bash
zap
```

Embedding directories and files into your project based on calls to 
`zap.Resource`:
```bash
zap
```

Using Zap during development or testing to allow it to read off the filesystem
instead of files it has embedded:
```bash
zap -devMode
```

## Anatomy of a Resource
A call to `zap.Resource` has two parts, a `Key` and a `Path`. The `Path` is the
directory that should be embedded into the application. All subdirectories of
paths specified in calls to `zap.Resource` will be embedded. The other part of
a call to `zap.Resource` is the `Key` which should be unique across the entire
project, any string value can be used provided it meets this constraint.

## Licensing
Zap itself is licensed under the GPLv3 license. However, because it both copies
a portion of its code (contained in `zapped/zapped.go`) as well as generating
code and placing it into projects that use it, an exception has been made to 
allow projects that contain those files to still be distributed under any terms
that projects maintainer chooses. If you have any questions around this, or are
unsure if you can use Zap, please do not hesitiate to contact me.