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
	data    *Database // database containing all known media in this library

	newMedia     chan *Discovery // media discovery
	newDirectory chan *Discovery // subdirectory discovery

	scanTime chan time.Time // counting semaphore to limit number of concurrent scanners
}

// type PathHandler represents a function that accepts a *Library, file path,
// and variable number of additional arguments. this is intended for use by the
// function walk() when it encounters files and directories.
type PathHandler func(*Library, string, ...interface{})
type ScanHandler struct {
	handleEnter, handleExit, handleMedia, handleAux, handleOther PathHandler
}

// type Discovery represents any sort of file entity discovered during a file
// system traversal of the library; we can capture here any other useful info
// describing the state of the file system traversal / search at the exact
// moment in time in which it was discovered.
type Discovery struct {
	time time.Time
	data []interface{}
}

func newDiscovery(d ...interface{}) *Discovery {
	return &Discovery{time: time.Now(), data: d}
}

// unexported constants
const (
	depthUnlimited     = 0
	maxLibraryScanners = 1
)

// function newLibrary() creates and initializes a new Library ready to scan.
// the library database is also created if one doesn't already exist, otherwise
// it is opened for businesss.
func newLibrary(opt *Options, lib string, lim uint, curr []*Library) (*Library, *ReturnCode) {

	// pull only the relevant info we need from the Options struct
	dat := opt.LibData.string

	// determine the user's current working dir -- from where they invoked us
	dir, err := os.Getwd()
	if nil != err {
		return nil, rcInvalidLibrary.withInfof(
			"newLibrary({libdata:%q}, %q, %q): %s", dat, lib, err)
	}

	// determine the absolute path to the directory tree containing media
	abs, err := filepath.Abs(lib)
	if nil != err {
		return nil, rcInvalidLibrary.withInfof(
			"newLibrary({libdata:%q}, %q, %q): %s", dat, lib, err)
	}

	// verify we haven't already seen this path in our library list
	for _, p := range curr {
		if p.absPath == abs {
			return nil, rcDuplicateLibrary.withInfof(
				"newLibrary({libdata:%q}, %q, %q): library already exists (skipping): %q", dat, lib, abs)
		}
	}

	// open the root directory of the library file system for reading
	fds, err := os.Open(abs)
	if nil != err {
		return nil, rcInvalidLibrary.withInfof(
			"newLibrary({libdata:%q}, %q, %q): %s", dat, lib, err)
	}
	defer fds.Close()

	// read all content of the root directory in the library file system
	_, err = fds.Readdir(0)
	if nil != err {
		return nil, rcInvalidLibrary.withInfof(
			"newLibrary({libdata:%q}, %q, %q): %s", dat, lib, err)
	}

	// open or create the library database if it doesn't exist
	dba, ret := newDatabase(opt, abs, dat)
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
		data:    dba,

		// channels for communicating scanner data to the main thread
		newMedia:     make(chan *Discovery),
		newDirectory: make(chan *Discovery),
		scanTime:     make(chan time.Time, maxLibraryScanners),
	}, nil
}

// creates a string representation of the Library for easy identification in
// logs.
func (l *Library) String() string {
	return fmt.Sprintf("{%q,%q,%s}", l.name, l.absPath, l.data)
}

func (l *Library) Path() string {
	return l.absPath
}

// function walk() is the recursive step for the file system traversal, invoked
// initially by function scan(). error codes generated in this routine will be
// returned to the caller of walk() -and- the caller of scan().
func (l *Library) walk(absPath string, depth uint, sh *ScanHandler) *ReturnCode {

	// get a path to the file relative to the current working dir (useful for
	// displaying diagnostic info to the user)
	relPath, err := filepath.Rel(l.workingDir, absPath)
	if nil != err {
		return rcInvalidPath.withInfof(
			"walk(%q, %d): filepath.Rel(%q): %s", absPath, depth, l.workingDir, err)
	}

	// read fs attributes to determine how we handle the file
	fileInfo, err := os.Lstat(absPath)
	if nil != err {
		// NOTE: we show relPath for readability, -should- be equivalent :^)
		return rcInvalidStat.withInfof(
			"walk(%q, %d): os.Lstat(): %s", relPath, depth, err)
	}
	mode := fileInfo.Mode()

	// operate on the file based on its file mode
	switch {
	case (mode & os.ModeDir) > 0:
		// file is directory, walk its contents unless we are at max depth
		if depthUnlimited != l.maxDepth && depth > l.maxDepth {
			return rcDirDepth.withInfof(
				"walk(%q, %d): limit = %d", relPath, depth, l.maxDepth)
		}
		dir, err := os.Open(absPath)
		if nil != err {
			return rcDirOpen.withInfof(
				"walk(%q, %d): os.Open(): %s", relPath, depth, err)
		}
		contentInfo, err := dir.Readdir(0)
		dir.Close()
		if nil != err {
			return rcDirOpen.withInfof(
				"walk(%q, %d): dir.Readdir(): %s", relPath, depth, err)
		}

		// don't add the library path itself to its list of subdirectories
		if depth > 1 {
			// fire the on-enter-directory event handler
			if nil != sh && nil != sh.handleEnter {
				sh.handleEnter(l, absPath)
			}
		}

		// recursively walk all of this subdirectory's contents
		for _, info := range contentInfo {
			err := l.walk(path.Join(absPath, info.Name()), depth+1, sh)
			if nil != err {
				// a file/subdir of the current directory threw an error
				warnLog.verbose(err)
			}
		}

		// fire the on-exit-directory event handler
		if nil != sh && nil != sh.handleExit {
			sh.handleExit(l, absPath)
		}
		return nil

	case (mode & os.ModeSymlink) > 0:
		// symlinks currently unhandled
		return rcInvalidFile.withInfof(
			"walk(%q, %d): symlinks not supported! (skipping)", relPath, depth)

	case (mode & (os.ModeDevice | os.ModeNamedPipe | os.ModeSocket | os.ModeCharDevice)) > 0:
		// file is not a regular file, not supported
		return rcInvalidFile.withInfof(
			"walk(%q, %d): not a regular file (skipping)", relPath, depth)
	}

	// if we made it here, we have a regular file. check if it is a Media file,
	// one of its auxiliary/support files, or an undesirable piece of trash.

	if /* the file discovered is a media file */ true {

		// create a Media struct object, and try analyzing the media content
		media, errCode := newMedia(l, absPath, relPath, fileInfo)
		if nil != errCode {
			return errCode
		}

		// fire the event handler for new Media discovery
		if nil != sh && nil != sh.handleMedia {
			sh.handleMedia(l, absPath, media)
		}

	} else {
		if /* the file is an auxiliary/support file */ false {
			// we encountered a file that supports a Media file we already know
			// about or one we haven't yet encountered. associate it with the
			// Media struct object.
			if nil != sh && nil != sh.handleAux {
				sh.handleAux(l, absPath)
			}

		} else {
			// we encountered an undesirable piece of trash. handle it as such.
			if nil != sh && nil != sh.handleOther {
				sh.handleOther(l, absPath)
			}
		}
	}

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

	// try writing to the buffered channel. this will succeed if and only if it
	// isn't already filled to capacity.
	select {
	case l.scanTime <- time.Now():
		// the write succeeded, so we can initiate scanning. keep track of the
		// time at which we began so that the time elapsed can be calculated and
		// notified to the user.
		infoLog.verbosef("scanning: %q", l.name)
		err = l.walk(l.absPath, 1, handler)
		elapsed := time.Since(<-l.scanTime)
		infoLog.verbosef(
			"finished scanning: %q (%s)", l.name, elapsed.Round(time.Millisecond))
	default:
		// if the write failed, we fall back to this default case. the only
		// reason it should fail is if the buffer is already filled to capacity,
		// meaning we already have the max allowed number of goroutines scanning
		// this library's file system.
		err = rcLibraryBusy.withInfof(
			"scan(): max number of scanners reached: %q (max = %d)", l.absPath, maxLibraryScanners)
	}
	return err
}
