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

	newMedia     chan *MediaDiscovery  // media discovery
	newDirectory chan *SubdirDiscovery // subdirectory discovery

	scanTime chan time.Time // counting semaphore to limit number of concurrent scanners
}

// type PathHandler represents a function that accepts a *Library, file path,
// and variable number of additional arguments. this is intended for use by the
// function walk() when it encounters files and directories.
type PathHandler func(*Library, string, ...interface{})
type ScanHandler struct {
	dirEnter, dirExit, fileMedia, fileOther PathHandler
}

// type Discovery represents any sort of file entity discovered during a file
// system traversal of the library; we can capture here any other useful info
// describing the state of the file system traversal / search at the exact
// moment in time in which it was discovered.
type Discovery struct {
	time time.Time
	data interface{}
}

// type MediaDiscovery represents specifically the discovery of a Media object.
type MediaDiscovery struct {
	Discovery
	*Media
}

// function newMediaDiscovery() creates a new MediaDiscovery object with the
// time field set to current time via time.Now().
func newMediaDiscovery(m *Media, d interface{}) *MediaDiscovery {
	return &MediaDiscovery{Discovery{time.Now(), d}, m}
}

// type SubdirDiscovery represents specifically the discovery of a subdirectory.
type SubdirDiscovery struct {
	Discovery
	string
}

// function newSubdirDiscovery() creates a new SubdirDiscovery object with the
// time field set to current time via time.Now().
func newSubdirDiscovery(s string, d interface{}) *SubdirDiscovery {
	return &SubdirDiscovery{Discovery{time.Now(), d}, s}
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

	// determine the user's current working dir -- from where they invoked us
	dir, err := os.Getwd()
	if nil != err {
		return nil, rcInvalidLibrary.withInfof("newLibrary(%q, %q): %s", lib, dat, err)
	}

	// determine the absolute path to the directory tree containing media
	abs, err := filepath.Abs(lib)
	if nil != err {
		return nil, rcInvalidLibrary.withInfof("newLibrary(%q, %q): %s", lib, dat, err)
	}

	// open the root directory of the library file system for reading
	fds, err := os.Open(abs)
	if nil != err {
		return nil, rcInvalidLibrary.withInfof("newLibrary(%q, %q): %s", lib, dat, err)
	}
	defer fds.Close()

	// read all content of the root directory in the library file system
	_, err = fds.Readdir(0)
	if nil != err {
		return nil, rcInvalidLibrary.withInfof("newLibrary(%q, %q): %s", lib, dat, err)
	}

	// verify or create the database storage location if it doesn't exist
	if exists, _ := goutil.PathExists(dat); !exists {
		if err := os.MkdirAll(dat, os.ModePerm); nil != err {
			return nil, rcInvalidLibrary.withInfof("newLibrary(%q, %q): %s", lib, dat, err)
		}
		infoLog.tracef("created data directory: %q", dat)
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
		newMedia:     make(chan *MediaDiscovery),
		newDirectory: make(chan *SubdirDiscovery),
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
func (l *Library) walk(absPath string, depth uint, handler *ScanHandler) *ReturnCode {

	// get a path to the file relative to the current working dir (useful for
	// displaying diagnostic info to the user)
	relPath, err := filepath.Rel(l.workingDir, absPath)
	if nil != err {
		return rcInvalidPath.withInfof("walk(%q, %d): filepath.Rel(%q): %s", absPath, depth, l.workingDir, err)
	}

	// read fs attributes to determine how we handle the file
	fileInfo, err := os.Lstat(absPath)
	if nil != err {
		// NOTE: we show relPath for readability, -should- be equivalent :^)
		return rcInvalidStat.withInfof("walk(%q, %d): os.Lstat(): %s", relPath, depth, err)
	}
	mode := fileInfo.Mode()

	// operate on the file based on its file mode
	switch {
	case (mode & os.ModeDir) > 0:
		// file is directory, walk its contents unless we are at max depth
		if depthUnlimited != l.maxDepth && depth > l.maxDepth {
			return rcDirDepth.withInfof("walk(%q, %d): limit = %d", relPath, depth, l.maxDepth)
		}
		dir, err := os.Open(absPath)
		if nil != err {
			return rcDirOpen.withInfof("walk(%q, %d): os.Open(): %s", relPath, depth, err)
		}
		contentInfo, err := dir.Readdir(0)
		dir.Close()
		if nil != err {
			return rcDirOpen.withInfof("walk(%q, %d): dir.Readdir(): %s", relPath, depth, err)
		}

		// don't add the library path itself to its list of subdirectories
		if depth > 1 {
			// but notify the consumer of a new subdirectory
			if nil != handler && nil != handler.dirEnter {
				handler.dirEnter(l, absPath)
			}
			//l.newDirectory <- newSubdirDiscovery(absPath, nil)
		}

		// recursively walk all of this subdirectory's contents
		for _, info := range contentInfo {
			err := l.walk(path.Join(absPath, info.Name()), depth+1, handler)
			if nil != err {
				// a file/subdir of the current directory threw an error
				warnLog.verbose(err)
			}
		}
		if nil != handler && nil != handler.dirExit {
			handler.dirExit(l, absPath)
		}
		return nil

	case (mode & os.ModeSymlink) > 0:
		// symlinks currently unhandled
		return rcInvalidFile.withInfof("walk(%q, %d): skipping, symlinks not supported!", relPath, depth)

	case (mode & (os.ModeDevice | os.ModeNamedPipe | os.ModeSocket | os.ModeCharDevice)) > 0:
		// file is not a regular file, not supported
		return rcInvalidFile.withInfof("walk(%q, %d): skipping, not a regular file", relPath, depth)
	}

	// if we made it here, we have a regular file. add it as a media candidate
	// only if we were able to successfully os.Lstat() it
	media, errCode := newMedia(l, absPath, relPath, fileInfo)
	if nil != errCode {
		return errCode
	}

	// finally, notify the consumer of our media discovery
	if nil != handler && nil != handler.fileMedia {
		handler.fileMedia(l, absPath, media)
	}
	//l.newMedia <- newMediaDiscovery(media, nil)

	return nil
}

// function scan() is the entry point for initiating a scan on the library's
// root file system. currently, the scan is dispatched and cannot be safely
// interruped. you must wait for the scan to finish before restarting.
func (l *Library) scan(handler *ScanHandler) *ReturnCode {

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
		infoLog.verbosef("scanning: %q", l.name)
		err = l.walk(l.absPath, 1, handler)
		elapsed := time.Since(<-l.scanTime)
		infoLog.verbosef("finished scanning: %q (%s)", l.name, elapsed.Round(time.Millisecond))
	default:
		err = rcLibraryBusy.withInfof("scan(): max number of scanners reached: %q (max = %d)", l.absPath, maxLibraryScanners)
	}
	return err
}

func (l *Library) readAll() *ReturnCode {
	return nil
}
