// =============================================================================
//  PROJ: pimmp
//  AUTH: ardnew
//  DATE: 26 Sept 2018
//  FILE: library.go
// -----------------------------------------------------------------------------
//
//  DESCRIPTION
//    defines types and operations for interacting with library databases,
//    traversing file systems, and managing user-defined collections of media.
//
// =============================================================================

package main

import (
	"ardnew.com/goutil"

	"fmt"
	"os"
	"path"
	"path/filepath"
	"time"
)

// type Library represents a collection of a specified kind of media files
// together with a rooted search path from which all media file discovery
// is performed.
type Library struct {
	workingDir string // current working directory
	absPath    string // absolute path to library
	name       string // library name (default: basename of path)
	maxDepth   uint   // maximum traversal depth (unlimited: 0)

	dataDir string    // directory containing all known library databases
	db      *Database // database containing all known media in this library

	newMedia     chan *Media // media discovery
	newDirectory chan string // subdirectory discovery

	scanTime chan time.Time // counting semaphore to limit number of concurrent scanners
}

// unexported constants
const (
	defaultDataDir     = "library.db"
	depthUnlimited     = 0
	maxLibraryScanners = 1
)

// function newLibrary() creates and initializes a new Library ready to scan.
// the library database is also created if one doesn't already exist, otherwise
// it is opened for businesss.
func newLibrary(lib, dat string, lim uint) (*Library, *ReturnCode) {

	// common logic for constructing and throwing an error ReturnCode
	invalidLibrary := func(l, d string, e error) *ReturnCode {
		info := fmt.Sprintf("newLibrary(%q, %q): %s", l, d, e)
		return rcInvalidLibrary.withInfo(info)
	}

	// determine the user's current working dir -- from where they invoked us
	dir, err := os.Getwd()
	if nil != err {
		return nil, invalidLibrary(lib, dat, err)
	}

	// determine the absolute path to the directory tree containing media
	abs, err := filepath.Abs(lib)
	if nil != err {
		return nil, invalidLibrary(lib, dat, err)
	}

	// open the root directory of the library file system for reading
	fds, err := os.Open(abs)
	if nil != err {
		return nil, invalidLibrary(lib, dat, err)
	}
	defer fds.Close()

	// read all content of the root directory in the library file system
	_, err = fds.Readdir(0)
	if nil != err {
		return nil, invalidLibrary(lib, dat, err)
	}

	// verify or create the database storage location if it doesn't exist
	if exists, _ := goutil.PathExists(dat); !exists {
		if err := os.MkdirAll(dat, os.ModePerm); nil != err {
			return nil, invalidLibrary(lib, dat, err)
		}
		infoLog.vlog(fmt.Sprintf("created data directory: %q", dat))
	}

	// open or create the library database if it doesn't exist
	dba, ret := newDatabase(abs, dat)
	if nil != ret {
		return nil, ret
	}

	return &Library{
		workingDir: dir,
		absPath:    abs,
		name:       path.Base(abs),
		maxDepth:   lim,

		// path to the library database directory
		dataDir: dat,
		db:      dba,

		// channels for communicating scanner data to the main thread
		newMedia:     make(chan *Media),
		newDirectory: make(chan string),
		scanTime:     make(chan time.Time, maxLibraryScanners),
	}, nil
}

// creates a string representation of the Library for easy identification in
// logs.
func (l *Library) String() string {
	return fmt.Sprintf("{%q,%q,%s}", l.name, l.absPath, l.db)
}

// function walk() is the recursive step for the file system traversal, invoked
// initially by function scan(). error codes generated in this routine will be
// returned to the caller of walk() -and- the caller of scan().
func (l *Library) walk(absPath string, depth uint) *ReturnCode {

	// get a path to the file relative to the current working dir (useful for
	// displaying diagnostic info to the user)
	relPath, err := filepath.Rel(l.workingDir, absPath)
	if nil != err {
		info := fmt.Sprintf("walk(%q, %d): filepath.Rel(%q): %s", absPath, depth, l.workingDir, err)
		return rcInvalidPath.withInfo(info)
	}

	// read fs attributes to determine how we handle the file
	fileInfo, err := os.Lstat(absPath)
	if nil != err {
		// NOTE: we show relPath for readability, -should- be equivalent :^)
		info := fmt.Sprintf("walk(%q, %d): os.Lstat(): %s", relPath, depth, err)
		return rcInvalidStat.withInfo(info)
	}
	mode := fileInfo.Mode()

	// operate on the file based on its file mode
	switch {
	case (mode & os.ModeDir) > 0:
		// file is directory, walk its contents unless we are at max depth
		if depthUnlimited != l.maxDepth && depth > l.maxDepth {
			info := fmt.Sprintf("walk(%q, %d): limit = %d", relPath, depth, l.maxDepth)
			return rcDirDepth.withInfo(info)
		}
		dir, err := os.Open(absPath)
		if nil != err {
			info := fmt.Sprintf("walk(%q, %d): os.Open(): %s", relPath, depth, err)
			return rcDirOpen.withInfo(info)
		}
		contentInfo, err := dir.Readdir(0)
		dir.Close()
		if nil != err {
			info := fmt.Sprintf("walk(%q, %d): dir.Readdir(): %s", relPath, depth, err)
			return rcDirOpen.withInfo(info)
		}

		// don't add the library path itself to its list of subdirectories
		if depth > 1 {
			// but notify the consumer of a new subdirectory
			l.newDirectory <- absPath
		}

		// recursively walk all of this subdirectory's contents
		for _, info := range contentInfo {
			err := l.walk(path.Join(absPath, info.Name()), depth+1)
			if nil != err {
				// a file/subdir of the current directory threw an error
				warnLog.vlog(err)
			}
		}
		return nil

	case (mode & os.ModeSymlink) > 0:
		// symlinks currently unhandled
		info := fmt.Sprintf("walk(%q, %d): skipping, symlinks not supported!", relPath, depth)
		return rcInvalidFile.withInfo(info)

	case (mode & (os.ModeDevice | os.ModeNamedPipe | os.ModeSocket | os.ModeCharDevice)) > 0:
		// file is not a regular file, not supported
		info := fmt.Sprintf("walk(%q, %d): skipping, not a regular file", relPath, depth)
		return rcInvalidFile.withInfo(info)
	}

	// if we made it here, we have a regular file. add it as a media candidate
	// only if we were able to successfully os.Lstat() it
	media, errCode := newMedia(l, absPath, relPath, fileInfo)
	if nil != errCode {
		return errCode
	}

	// finally, notify the consumer of our media discovery
	l.newMedia <- media

	return nil
}

// function scan() is the entry point for initiating a scan on the library's
// root file system. currently, the scan is dispatched and cannot be safely
// interruped. you must wait for the scan to finish before restarting.
func (l *Library) scan() *ReturnCode {

	var err *ReturnCode

	//
	// the scanTime channel is buffered so that we can limit the number of
	// goroutines concurrently traversing this library's file system:
	//
	//     writes to the channel will fail and fallback on the default select
	//     case if the max number of scanners is reached -- which sets an error
	//     code that is returned to the caller -- so be sure to check
	//     the return value when calling function scan()!
	//

	select {
	case l.scanTime <- time.Now():
		infoLog.vlogf("initiating scan: %q", l.name)
		err = l.walk(l.absPath, 1)
		elapsed := time.Since(<-l.scanTime)
		infoLog.vlogf("finished scan: %q (%s)", l.name, elapsed.Round(time.Millisecond))
		//infoLog.vlogf("created library: %s", l)
	default:
		info := fmt.Sprintf("scan(): max number of scanners reached: %q (max = %d)", l.absPath, maxLibraryScanners)
		err = rcLibraryBusy.withInfo(info)
	}
	return err
}

func (l *Library) readAll() *ReturnCode {
	return nil
}
