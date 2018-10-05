// =============================================================================
//  PROJ: pimmp
//  AUTH: ardnew
//  DATE: 26 Sept 2018
//  FILE: library.go
// -----------------------------------------------------------------------------
//
//  DESCRIPTION
//    (TBD)
//
// =============================================================================

package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
)

// type Library represents a collection of a specified kind of media files
// together with a rooted search path from which all media file discovery
// is performed.
type Library struct {
	cdir string // current working directory
	path string // absolute path to library
	name string // library name (default: basename of path)
}

func NewLibrary(lib string) (*Library, *ErrorCode) {

	dir, err := os.Getwd()
	if nil != err {
		info := fmt.Sprintf("NewLibrary(%q): %s", lib, err)
		return nil, EInvalidLibrary.WithInfo(info)
	}

	abs, err := filepath.Abs(lib)
	if nil != err {
		info := fmt.Sprintf("NewLibrary(%q): %s", lib, err)
		return nil, EInvalidLibrary.WithInfo(info)
	}

	fds, err := os.Open(abs + "x")
	if nil != err {
		info := fmt.Sprintf("NewLibrary(%q): %s", lib, err)
		return nil, EInvalidLibrary.WithInfo(info)
	}
	defer fds.Close()

	_, err = fds.Readdir(0)
	if nil != err {
		info := fmt.Sprintf("NewLibrary(%q): %s", lib, err)
		return nil, EInvalidLibrary.WithInfo(info)
	}

	library := Library{
		cdir: dir,
		path: abs,
		name: path.Base(abs),
	}
	return &library, nil
}
