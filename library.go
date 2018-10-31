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
	newAuxiliary chan *Discovery // support file discovery

	scanComplete chan bool      // synchronization lock
	scanStart    chan time.Time // counting semaphore to limit number of concurrent scanners
	scanElapsed  time.Duration  // measures time elapsed for scan to complete (use internally, not thread-safe!)
}

// type PathHandler represents a function that accepts a *Library, file path,
// and variable number of additional arguments. this is intended for use by the
// function walk() when it encounters files and directories.
type PathHandler func(*Library, string, ...interface{})
type ScanHandler struct {
	handleEnter, handleExit, handleMedia, handleAux, handleOther PathHandler
}

// var defaultScanHandler is a simple event handler that essentially delegates
// all significant work onto the discovery channel polling routines.
var defaultScanHandler *ScanHandler

// type Discovery represents any sort of file entity discovered during a file
// system traversal of the library; we can capture here any other useful info
// describing the state of the file system traversal / search at the exact
// moment in time in which it was discovered.
type Discovery struct {
	time time.Time
	data []interface{}
}

// function newDiscovery() constructs a new instance of a Discovery struct
// with the current time and the provided data.
func newDiscovery(d ...interface{}) *Discovery {
	return &Discovery{time: time.Now(), data: d}
}

// unexported constants
const (
	depthUnlimited     = 0
	maxLibraryScanners = 1
)

// function init() initializes all of the locally-declared data for use both
// locally and globally
func init() {
	defaultScanHandler = &ScanHandler{
		// the scanner entered a subdirectory of the library's file system.
		handleEnter: func(l *Library, p string, v ...interface{}) {
			l.newDirectory <- newDiscovery(p)
		},
		// the scanner exited a subdirectory of the library's file system.
		handleExit: func(l *Library, p string, v ...interface{}) {
		},
		// the scanner identified some file in a subdirectory of the library's
		// file system as a media file.
		handleMedia: func(l *Library, p string, v ...interface{}) {
			l.newMedia <- newDiscovery(v...)
		},
		// the scanner identified some file in a subdirectory of the library's
		// file system as a supporting auxiliary file to a known or as-of-yet
		// unknown media file.
		handleAux: func(l *Library, p string, v ...interface{}) {
			l.newAuxiliary <- newDiscovery(v...)
		},
		// the scanner identified some file in a subdirectory of the library's
		// file system as an undesirable piece of trash.
		handleOther: func(l *Library, p string, v ...interface{}) {
		},
	}
}

// function newLibrary() creates and initializes a new Library ready to scan.
// the library database is also created if one doesn't already exist, otherwise
// it is opened for businesss.
func newLibrary(opt *Options, lib string, lim uint, curr []*Library) (*Library, *ReturnCode) {

	// pull only the relevant info we need from the Options struct.
	dat := opt.LibData.string

	// determine the user's current working dir -- from where they invoked us.
	dir, err := os.Getwd()
	if nil != err {
		return nil, rcInvalidLibrary.specf(
			"newLibrary(%q, %q): os.Getwd(): %s", dat, lib, err)
	}

	// determine the absolute path to the directory tree containing media.
	abs, err := filepath.Abs(lib)
	if nil != err {
		return nil, rcInvalidLibrary.specf(
			"newLibrary(%q, %q): filepath.Abs(): %s", dat, lib, err)
	}

	// verify we haven't already seen this path in our library list.
	for _, p := range curr {
		if p.absPath == abs {
			return nil, rcDuplicateLibrary.specf(
				"newLibrary(%q, %q): library already exists (skipping): %q", dat, lib, abs)
		}
	}

	// open the root directory of the library file system for reading.
	fds, err := os.Open(abs)
	if nil != err {
		return nil, rcInvalidLibrary.specf(
			"newLibrary(%q, %q): os.Open(): %s", dat, lib, err)
	}
	defer fds.Close()

	// read all content of the root directory in the library file system.
	_, err = fds.Readdir(0)
	if nil != err {
		return nil, rcInvalidLibrary.specf(
			"newLibrary(%q, %q): Readdir(): %s", dat, lib, err)
	}

	// open or create the library database if it doesn't exist.
	dba, ret := newDatabase(opt, abs, dat)
	if nil != ret {
		return nil, ret
	}

	return &Library{
		workingDir: dir,
		absPath:    abs,
		name:       path.Base(abs),
		maxDepth:   lim,

		// path to the library database directory.
		dataDir: dat,
		data:    dba,

		// channels for communicating scanner data to the main thread.
		newMedia:     make(chan *Discovery),
		newDirectory: make(chan *Discovery),
		newAuxiliary: make(chan *Discovery),

		scanComplete: make(chan bool),
		scanStart:    make(chan time.Time, maxLibraryScanners),
		scanElapsed:  0,
	}, nil
}

// creates a string representation of the Library for easy identification in
// logs.
func (l *Library) String() string {
	return fmt.Sprintf("{%q,%q,%s}", l.name, l.absPath, l.data)
}

// function walk() is the recursive step for the file system traversal, invoked
// initially by function scan(). error codes generated in this routine will be
// returned to the caller of walk() -and- the caller of scan().
func (l *Library) walk(absPath string, depth uint, sh *ScanHandler) *ReturnCode {

	// get a path to the file relative to the current working dir (useful for
	// displaying diagnostic info to the user).
	relPath, err := filepath.Rel(l.workingDir, absPath)
	if nil != err {
		return rcInvalidPath.specf(
			"walk(%q, %d): filepath.Rel(%q): %s", absPath, depth, l.workingDir, err)
	}

	// the purpose of showing the relative path is concision -- readability. if
	// that relative path is longer (e.g. a bunch of "../../..") than the
	// absolute path, then just show the absolute path instead.
	dispPath := absPath // absPath wins the tiebreak if equal length
	if len(relPath) < len(absPath) {
		dispPath = relPath
	}

	// read fs attributes to determine how we handle the file.
	fileInfo, err := os.Lstat(absPath)
	if nil != err {
		return rcInvalidStat.specf(
			"walk(%q, %d): os.Lstat(): %s", dispPath, depth, err)
	}
	mode := fileInfo.Mode()

	// operate on the file based on its file mode.
	switch {
	case (mode & os.ModeDir) > 0:

		// file is directory, walk its contents unless we are at max depth.
		if depthUnlimited != l.maxDepth && depth > l.maxDepth {
			return rcDirDepth.specf(
				"walk(%q, %d): limit = %d", dispPath, depth, l.maxDepth)
		}
		dir, err := os.Open(absPath)
		if nil != err {
			return rcDirOpen.specf(
				"walk(%q, %d): os.Open(): %s", dispPath, depth, err)
		}
		contentInfo, err := dir.Readdir(0)
		if nil != err {
			return rcDirOpen.specf(
				"walk(%q, %d): dir.Readdir(): %s", dispPath, depth, err)
		}
		dir.Close()
		// don't add the library path itself to its list of subdirectories.
		if depth > 1 {
			// fire the on-enter-directory event handler.
			if nil != sh && nil != sh.handleEnter {
				sh.handleEnter(l, absPath)
			}
		}
		// recursively walk all of this subdirectory's contents.
		for _, info := range contentInfo {
			err := l.walk(path.Join(absPath, info.Name()), depth+1, sh)
			if nil != err {
				// a file/subdir of the current directory threw an error.
				warnLog.verbose(err)
			}
		}
		// fire the on-exit-directory event handler.
		if nil != sh && nil != sh.handleExit {
			sh.handleExit(l, absPath)
		}
		return nil

	case (mode & os.ModeSymlink) > 0:
		// symlinks currently unhandled.
		return rcInvalidFile.specf(
			"walk(%q, %d): symlinks not supported! (skipping)", dispPath, depth)

	case (mode & (os.ModeDevice | os.ModeNamedPipe | os.ModeSocket | os.ModeCharDevice)) > 0:
		// file is not a regular file, not supported.
		return rcInvalidFile.specf(
			"walk(%q, %d): not a regular file (skipping)", dispPath, depth)

	default:

		// first extract the file name extension. this is how we determine file
		// type; not very intelligible, but fast and mostly reliable for media
		// files.
		ext := path.Ext(absPath)

		// check if it looks like a regular media file.
		switch kind, extName := mediaKindOfFileExt(ext); kind {
		case mkAudio:
			if nil != sh && nil != sh.handleMedia {
				media, errCode := newAudioMedia(l, absPath, relPath, ext, extName, fileInfo)
				if nil != errCode {
					return errCode
				}
				sh.handleMedia(l, absPath, media)
			}
		case mkVideo:
			if nil != sh && nil != sh.handleMedia {
				media, errCode := newVideoMedia(l, absPath, relPath, ext, extName, fileInfo)
				if nil != errCode {
					return errCode
				}
				sh.handleMedia(l, absPath, media)
			}
		default:
			// doesn't have an extension typically associated with media files.
			// check if it is a media-supporting file.
			switch kind, extName := mediaSupportKindOfFileExt(ext); kind {
			case mskSubtitles:
				if nil != sh && nil != sh.handleAux {
					sh.handleAux(l, absPath, mskSubtitles, ext, extName, fileInfo)
				}
			default:
				// cannot identify the file, probably an undesirable piece of
				// trash. well-suited for being ignored.
				if nil != sh && nil != sh.handleOther {
					sh.handleOther(l, absPath)
				}
			}
		}
		return nil
	}

	// we should never reach here
	return nil
}

// function scan() is the entry point for initiating a scan on the library's
// root file system. currently, the scan is dispatched and cannot be safely
// interruped. you must wait for the scan to finish before restarting.
func (l *Library) scan(handler *ScanHandler) *ReturnCode {

	var err *ReturnCode

	//
	// the scanStart channel is buffered so that we can limit the number of
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
	case l.scanStart <- time.Now():
		// the write succeeded, so we can initiate scanning. keep track of the
		// time at which we began so that the time elapsed can be calculated and
		// notified to the user.
		infoLog.verbosef("scanning: %q", l.name)
		err = l.walk(l.absPath, 1, handler)
		l.scanElapsed = time.Since(<-l.scanStart)
		infoLog.verbosef(
			"finished scanning: %q (%s)",
			l.name, l.scanElapsed.Round(time.Millisecond))
	default:
		// if the write failed, we fall back to this default case. the only
		// reason it should fail is if the buffer is already filled to capacity,
		// meaning we already have the max allowed number of goroutines scanning
		// this library's file system.
		err = rcLibraryBusy.specf(
			"scan(): max number of scanners reached: %q (max = %d)",
			l.absPath, maxLibraryScanners)
	}
	l.scanComplete <- true
	return err
}
