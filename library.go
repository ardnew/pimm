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
	"github.com/HouzuoGuo/tiedot/db"

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

	newMedia     chan *Discovery // media discovery
	newDirectory chan *Discovery // subdirectory discovery
	newAuxiliary chan *Discovery // support file discovery

	scanComplete chan bool      // synchronization lock
	scanStart    chan time.Time // counting semaphore to limit number of concurrent scanners
	scanElapsed  time.Duration  // measures time elapsed for scan to complete (use internally, not thread-safe!)

	loadComplete chan bool      // synchronization lock
	loadStart    chan time.Time // counting semaphore to limit number of concurrent loaders
	loadElapsed  time.Duration  // measures time elapsed for load to complete (use internally, not thread-safe!)
}

// type PathHandler represents a function that accepts a *Library, file path,
// and variable number of additional arguments. this is intended for use by the
// function scanDive() when it encounters files and directories.
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
func init() {}

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

	// read all content of the root directory in the library file system.
	_, err = fds.Readdir(0)
	fds.Close()
	if nil != err {
		return nil, rcInvalidLibrary.specf(
			"newLibrary(%q, %q): Readdir(): %s", dat, lib, err)
	}

	// open or create the library database if it doesn't exist.
	db, ret := newDatabase(opt, abs, dat)
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
		db:      db,

		// channels for communicating scanner data to the main thread.
		newMedia:     make(chan *Discovery),
		newDirectory: make(chan *Discovery),
		newAuxiliary: make(chan *Discovery),

		scanComplete: make(chan bool),
		scanStart:    make(chan time.Time, maxLibraryScanners),
		scanElapsed:  0,

		loadComplete: make(chan bool),
		loadStart:    make(chan time.Time, maxLibraryScanners),
		loadElapsed:  0,
	}, nil
}

// function String() creates a string representation of the Library for easy
// identification in logs.
func (l *Library) String() string {
	return fmt.Sprintf("{%q,%q,%s}", l.name, l.absPath, l.db)
}

// function scanDive() is the recursive step for the file system traversal,
// invoked initially by function scan(). error codes generated in this routine
// will be returned to the caller of scanDive() -and- the caller of scan().
func (l *Library) scanDive(absPath string, depth, count uint, sh *ScanHandler) (uint, *ReturnCode) {

	// get a path to the file relative to the library root dir (useful for
	// displaying diagnostic info to the user).
	relPath, err := filepath.Rel(l.absPath, absPath)
	if nil != err {
		return count, rcInvalidPath.specf(
			"scanDive(%q, %d): filepath.Rel(%q): %s", absPath, depth, l.absPath, err)
	}

	// for concision, show the relative path by default in any diagnostics/logs.
	//dispPath := absPath
	dispPath := relPath

	// read fs attributes to determine how we handle the file.
	fileInfo, err := os.Lstat(absPath)
	if nil != err {
		return count, rcInvalidStat.specf(
			"scanDive(%q, %d): os.Lstat(): %s", dispPath, depth, err)
	}
	mode := fileInfo.Mode()

	// operate on the file based on its file mode.
	switch {
	case (mode & os.ModeDir) > 0:
		// file is directory, scanDive its contents unless we are at max depth.
		if depthUnlimited != l.maxDepth && depth > l.maxDepth {
			return count, rcDirDepth.specf(
				"scanDive(%q, %d): limit = %d", dispPath, depth, l.maxDepth)
		}
		dir, err := os.Open(absPath)
		if nil != err {
			return count, rcDirOpen.specf(
				"scanDive(%q, %d): os.Open(): %s", dispPath, depth, err)
		}
		dirName, err := dir.Readdirnames(0)
		dir.Close()
		if nil != err {
			return count, rcDirOpen.specf(
				"scanDive(%q, %d): dir.Readdirnames(): %s", dispPath, depth, err)
		}

		// don't add the library path itself to its list of subdirectories.
		if depth > 1 {
			// fire the on-enter-directory event handler.
			if nil != sh && nil != sh.handleEnter {
				sh.handleEnter(l, absPath, nil)
			}
		}

		// recursively scan all of this subdirectory's contents.
		var scanErr *ReturnCode
		for _, name := range dirName {
			count, scanErr = l.scanDive(path.Join(absPath, name), depth+1, count, sh)
			if nil != scanErr {
				// a file/subdir of the current directory threw an error.
				warnLog.verbose(scanErr)
			}
		}
		// fire the on-exit-directory event handler.
		if nil != sh && nil != sh.handleExit {
			sh.handleExit(l, absPath, nil)
		}
		return count, nil

	case (mode & os.ModeSymlink) > 0:
		// symlinks currently unhandled.
		return count, rcInvalidFile.specf(
			"scanDive(%q, %d): symlinks not supported! (skipping)", dispPath, depth)

	case (mode & (os.ModeDevice | os.ModeNamedPipe | os.ModeSocket | os.ModeCharDevice)) > 0:
		// file is not a regular file, not supported.
		return count, rcInvalidFile.specf(
			"scanDive(%q, %d): not a regular file (skipping)", dispPath, depth)

	default:

		// function seenFile() checks if the file specified by path and kind of
		// media exists in the associated collection of this library's database.
		seenFile := func(lib *Library, class EntityClass, kind int, path string) (bool, error) {

			var col *db.Col
			var query map[string]interface{}

			switch class {
			case ecMedia:
				col = lib.db.mediaCol[kind]
				query = map[string]interface{}{
					"eq": path,
					"in": []interface{}{mediaIndex[mxPath][0]},
				}
			case ecSupport:
				col = lib.db.supportCol[kind]
				query = map[string]interface{}{
					"eq": path,
					"in": []interface{}{supportIndex[sxPath][0]},
				}
			}
			result := make(map[int]struct{})
			if err := db.EvalQuery(query, col, &result); nil != err {
				return false, err
			}
			return len(result) > 0, nil
		}

		// first extract the file name extension. this is how we determine file
		// type; not very intelligible, but fast and mostly reliable for media
		// files (~my~ media files, at least).
		ext := path.Ext(absPath)

		// check if it looks like a regular media file.
		switch kind, extName := mediaKindOfFileExt(ext); kind {
		case mkAudio:

			ac := l.db.mediaCol[mkAudio]
			seen, err := seenFile(l, ecMedia, int(kind), absPath)
			if err != nil {
				return count, rcInvalidFile.specf(
					"scanDive(%q, %d): failed to evaluate query: %s (skipping)", dispPath, depth, err)
			}
			if !seen {
				audio := newAudioMedia(l, absPath, relPath, ext, extName, fileInfo)
				if rec, recErr := audio.toRecord(); nil == recErr {
					if id, insErr := ac.Insert(*rec); nil == insErr {
						count++
						infoLog.tracef("discovered audio: %s", audio)
						if nil != sh && nil != sh.handleMedia {
							sh.handleMedia(l, absPath, audio, id)
						}
					} else {
						return count, rcDatabaseError.specf(
							"scanDive(%q, %d): failed to insert record: %s (skipping)", dispPath, depth, insErr)
					}
				} else {
					return count, recErr
				}
			}

		case mkVideo:

			vc := l.db.mediaCol[mkVideo]
			seen, err := seenFile(l, ecMedia, int(kind), absPath)
			if err != nil {
				return count, rcInvalidFile.specf(
					"scanDive(%q, %d): failed to evaluate query: %s (skipping)", dispPath, depth, err)
			}
			if !seen {
				video := newVideoMedia(l, absPath, relPath, ext, extName, fileInfo)
				if rec, recErr := video.toRecord(); nil == recErr {
					if id, insErr := vc.Insert(*rec); nil == insErr {
						count++
						infoLog.tracef("discovered video: %s", video)
						if nil != sh && nil != sh.handleMedia {
							sh.handleMedia(l, absPath, video, id)
						}
					} else {
						return count, rcDatabaseError.specf(
							"scanDive(%q, %d): failed to insert record: %s (skipping)", dispPath, depth, insErr)
					}
				} else {
					return count, recErr
				}
			}

		default:
			// doesn't have an extension typically associated with media files.
			// check if it is a media-supporting file.
			switch kind, extName := supportKindOfFileExt(ext); kind {
			case skSubtitles:
				sc := l.db.supportCol[skSubtitles]
				seen, err := seenFile(l, ecSupport, int(kind), absPath)
				if err != nil {
					return count, rcInvalidFile.specf(
						"scanDive(%q, %d): failed to evaluate query: %s (skipping)", dispPath, depth, err)
				}
				if !seen {
					subs := newSubtitles(l, absPath, relPath, ext, extName, fileInfo)
					if rec, recErr := subs.toRecord(); nil == recErr {
						if id, insErr := sc.Insert(*rec); nil == insErr {
							infoLog.tracef("discovered subtitles: %s", subs)
							if nil != sh && nil != sh.handleAux {
								sh.handleAux(l, absPath, skSubtitles, subs, id)
							}
						} else {
							return count, rcDatabaseError.specf(
								"scanDive(%q, %d): failed to insert record: %s (skipping)", dispPath, depth, insErr)
						}
					} else {
						return count, recErr
					}
				}

			default:
				// cannot identify the file, probably an undesirable piece of
				// trash. well-suited for being ignored.
				if nil != sh && nil != sh.handleOther {
					sh.handleOther(l, absPath)
				}
			}
		}
		return count, nil
	}

	// we should never reach here
	return count, nil
}

// function scan() is the entry point for initiating a scan on the library's
// root file system. currently, the scan is dispatched and cannot be safely
// interruped. you must wait for the scan to finish before restarting.
func (l *Library) scan(handler *ScanHandler) *ReturnCode {

	var (
		count uint = 0 // number of -new- files discovered on file system
		err   *ReturnCode
	)

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
		count, err = l.scanDive(l.absPath, 1, 0, handler)
		l.scanElapsed = time.Since(<-l.scanStart)
		infoLog.verbosef(
			"finished scanning: %q (%d media files in %s)",
			l.name, count, l.scanElapsed.Round(time.Millisecond))
	default:
		// if the write failed, we fall back to this default case. the only
		// reason it should fail is if the buffer is already filled to capacity,
		// meaning we already have the max allowed number of goroutines scanning
		// this library's file system.
		err = rcLibraryBusy.specf(
			"scan(): max number of scanners reached: %q (max = %d)",
			l.absPath, maxLibraryScanners)
	}
	return err
}

func (l *Library) loadDive(class EntityClass, kind int) (uint, *ReturnCode) {

	var count uint = 0

	switch class {
	case ecMedia:
		l.db.mediaCol[kind].ForEachDoc(
			func(id int, data []byte) (willMoveOn bool) {
				switch MediaKind(kind) {
				case mkAudio:
					audio := &AudioMedia{}
					audio.fromRecord(data)
					infoLog.tracef("loaded audio (ID = %d): %s", id, audio)
				case mkVideo:
					video := &VideoMedia{}
					video.fromRecord(data)
					infoLog.tracef("loaded video (ID = %d): %s", id, video)
				default:
				}
				count++
				return true // move on to the next document OR
			})

	case ecSupport:
		l.db.supportCol[kind].ForEachDoc(
			func(id int, data []byte) (willMoveOn bool) {
				switch SupportKind(kind) {
				case skSubtitles:
					subs := &Subtitles{}
					subs.fromRecord(data)
					infoLog.tracef("loaded subtitles (ID = %d): %s", id, subs)
				default:
				}
				count++
				return true // move on to the next document OR
			})
	default:
	}

	return count, nil
}

// function load() is the entry point for initiating a load on the library's
// backing data store. currently, the load is dispatched and cannot be safely
// interruped. you must wait for the load to finish before restarting.
func (l *Library) load() *ReturnCode {

	var (
		mediaTotal   uint = 0 // number of -existing-loaded from database
		supportTotal uint = 0 // ditto
		mediaCount        = [mkCOUNT]uint{0, 0}
		supportCount      = [skCOUNT]uint{0}
		err          *ReturnCode
	)

	//
	// the loadStart channel is buffered so that we can limit the number of
	// goroutines concurrently reading this library's database:
	//
	//     writes to the channel will fail and fallback on the default select
	//     case if the max number of loaders is reached -- which sets an error
	//     code that is returned to the caller -- so be sure to check
	//     the return value when calling function load()!
	//

	// try writing to the buffered channel. this will succeed if and only if it
	// isn't already filled to capacity.
	select {
	case l.loadStart <- time.Now():
		// the write succeeded, so we can initiate loading. keep track of the
		// time at which we began so that the time elapsed can be calculated and
		// notified to the user.
		infoLog.verbosef("loading: %q", l.name)

		for k := range mediaColName {
			mediaCount[k], err = l.loadDive(ecMedia, k)
			if nil != err {
				return err
			}
			mediaTotal += mediaCount[k]
		}

		for k := range supportColName {
			supportCount[k], err = l.loadDive(ecSupport, k)
			if nil != err {
				return err
			}
			supportTotal += supportCount[k]
		}

		l.loadElapsed = time.Since(<-l.loadStart)
		infoLog.verbosef(
			"finished loading: %q (%d media files, %d support files in %s)",
			l.name, mediaTotal, supportTotal, l.loadElapsed.Round(time.Millisecond))
	default:
		// if the write failed, we fall back to this default case. the only
		// reason it should fail is if the buffer is already filled to capacity,
		// meaning we already have the max allowed number of goroutines loading
		// this library's database.
		err = rcLibraryBusy.specf(
			"load(): max number of loaders reached: %q (max = %d)",
			l.absPath, maxLibraryScanners)
	}
	return err
}
