package server

import (
	"net/http"
	"os"
	"strings"
)

// A FileSystem which which can optionally:
//
// - serve .htm(l) files for the corresponding path without suffix, if the
//  original path is not found
//
// - hide dotfiles
type FileSystem struct {
	http.FileSystem

	ResolveHTML  bool
	HideDotFiles bool
	Root         string
}

// NewFileSystem returns a FileSystem with the specified configuration.
func NewFileSystem(root string, resolveHTML bool, hideDotFiles bool) FileSystem {
	return FileSystem{
		FileSystem:   http.Dir(root),
		ResolveHTML:  resolveHTML,
		HideDotFiles: hideDotFiles,
		Root:         string(http.Dir(root)),
	}
}

// Open returns a File object for the specified path under the FileSystem
// directory.
func (fs FileSystem) Open(name string) (http.File, error) {
	if fs.HideDotFiles && containsDotFile(name) {
		// Even if the file exists, return 404
		return nil, os.ErrNotExist
	}

	file, err := fs.FileSystem.Open(name)
	if fs.HideDotFiles {
		file = dotfileHidingFile{file}
	}

	if !os.IsNotExist(err) || !fs.ResolveHTML {
		return file, err
	}

	if !(strings.HasSuffix(name, ".html") || strings.HasSuffix(name, ".htm")) {
		for _, suffix := range []string{".html", ".htm"} {
			newName := name + suffix
			if file, err := fs.OpenFile(newName); err == nil {
				return file, nil
			}
		}
	}

	// return the result of the original call
	return file, err
}

// OpenFile returns a File object for the specified path under the FileSystem
// directory if it esists and it's not a directory.
func (fs FileSystem) OpenFile(name string) (http.File, error) {
	if file, err := fs.FileSystem.Open(name); err == nil {
		if fileInfo, err := file.Stat(); err == nil && !fileInfo.IsDir() {
			return file, nil
		}
	}
	return nil, os.ErrNotExist
}

// dotfileHidingFile wraps the Readdir method of http.File so to remove files
// and directories that start with a period from its output.
type dotfileHidingFile struct {
	http.File
}

// Readdir is a wrapper around the Readdir method of the embedded File that
// filters out all files that start with a period in their name.
func (f dotfileHidingFile) Readdir(n int) (fileInfos []os.FileInfo, err error) {
	files, err := f.File.Readdir(n)
	if err != nil {
		return
	}
	for _, file := range files {
		if !strings.HasPrefix(file.Name(), ".") {
			fileInfos = append(fileInfos, file)
		}
	}
	return
}

// containsDotFile reports whether name contains a path element starting with a
// period.  The name is assumed to be a delimited by forward slashes, as
// guaranteed by the http.FileSystem interface.
func containsDotFile(name string) bool {
	parts := strings.Split(name, "/")
	for _, part := range parts {
		if strings.HasPrefix(part, ".") {
			return true
		}
	}
	return false
}
