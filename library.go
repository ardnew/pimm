package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
)

type Library struct {
	workingDir string              // user's CWD
	path       string              // absolute path to library
	name       string              // logical filename portion of path
	depthLimit uint                // recursive traversal depth
	ignored    []string            // patterns of ignored directories
	media      map[string][]*Media // map of each subdirectory to slice of its files
	totalMedia int64               // number of media files discovered
	totalSize  int64               // total size of all media files discovered
	sigDir     chan string         // subdirectory discovery
	sigMedia   chan *Media         // media discovery
	sigWork    chan bool           // indicates scanner activity
}

const (
	LibraryDepthUnlimited = 0
	MaxScannersPerLibrary = 1
)

func NewLibrary(libName string, ignore []string) (*Library, *ErrorCode) {

	dir, err := os.Getwd()
	if nil != err {
		return nil, NewErrorCode(EInvalidLibrary, fmt.Sprintf("%s: %q", err, libName))
	}

	libPath, err := filepath.Abs(libName)
	if nil != err {
		return nil, NewErrorCode(EInvalidLibrary, fmt.Sprintf("%s: %q", err, libName))
	}

	hdl, err := os.Open(libPath)
	if nil != err {
		return nil, NewErrorCode(EInvalidLibrary, fmt.Sprintf("%s: %q", err, libName))
	}
	defer hdl.Close()

	_, err = hdl.Readdir(0)

	if nil != err {
		return nil, NewErrorCode(EInvalidLibrary, fmt.Sprintf("%s: %q", err, libName))
	}

	library := Library{
		workingDir: dir,
		path:       libPath,
		name:       path.Base(libPath),
		depthLimit: LibraryDepthUnlimited,
		ignored:    ignore,
		media:      make(map[string][]*Media),
		totalMedia: 0,
		totalSize:  0,
		sigDir:     make(chan string),
		sigMedia:   make(chan *Media),
		sigWork:    make(chan bool, MaxScannersPerLibrary),
	}
	return &library, nil
}

func (l *Library) String() string {
	return fmt.Sprintf("%q{%s}[%d]", l.name, l.path, len(l.media))
}

func (l *Library) WorkingDir() string {
	return l.workingDir
}

func (l *Library) Name() string {
	return l.name
}

func (l *Library) SetName(n string) {
	l.name = n
}

func (l *Library) Path() string {
	return l.path
}

func (l *Library) SetPath(p string) {
	l.path = p
}

func (l *Library) IsDepthLimited() bool {
	return LibraryDepthUnlimited != l.depthLimit
}

func (l *Library) DepthLimit() uint {
	return l.depthLimit
}

func (l *Library) SetDepthLimit(d uint) {
	l.depthLimit = d
}

func (l *Library) Ignored() []string {
	return l.ignored
}

func (l *Library) SetIgnored(i []string) {
	l.ignored = make([]string, len(i))
	copy(l.ignored, i)
}

func (l *Library) AddIgnored(i ...string) {
	l.ignored = append(l.ignored, i...)
}

func (l *Library) Media() map[string][]*Media {
	return l.media
}

func (l *Library) TotalMedia() int64 {
	return l.totalMedia
}

func (l *Library) TotalSize() int64 {
	return l.totalSize
}

func (l *Library) SigDir() chan string {
	return l.sigDir
}

func (l *Library) SigMedia() chan *Media {
	return l.sigMedia
}

func (l *Library) SigWork() chan bool {
	return l.sigWork
}

func (l *Library) Walk(currPath string, depth uint) *ErrorCode {

	// TODO: don't continue if file matches an ignore pattern
	// ...

	currDir := path.Dir(currPath)

	// get a path to the library relative to current working dir (useful for
	// displaying diagnostic info to the user)
	relPath, err := filepath.Rel(l.WorkingDir(), currPath)
	if nil != err {
		return NewErrorCode(EFileStat, fmt.Sprintf("%s: %q", err, currPath))
	}

	// read fs attributes to determine how we handle the file
	fileInfo, err := os.Lstat(currPath)
	if nil != err {
		return NewErrorCode(EFileStat, fmt.Sprintf("%s: %q", err, relPath))
	}
	mode := fileInfo.Mode()

	// operate on the file based on its file mode
	switch {
	case (mode & os.ModeDir) > 0:
		// file is directory, walk its contents unless we are at max depth
		if l.IsDepthLimited() && depth > l.depthLimit {
			return NewErrorCode(EDirDepth, fmt.Sprintf("exceeded limit=%d, skipping: %q", l.depthLimit, relPath))
		}
		dir, err := os.Open(currPath)
		if nil != err {
			return NewErrorCode(EDirOpen, fmt.Sprintf("%s: %q", err, relPath))
		}
		contentInfo, err := dir.Readdir(0)
		dir.Close()
		if nil != err {
			return NewErrorCode(EDirOpen, fmt.Sprintf("%s: %q", err, relPath))
		}

		// initialize this subdirectory's list of content/filenames
		l.media[currPath] = []*Media{}

		// don't add the library path itself to its list of subdirectories
		if depth > 1 {
			l.sigDir <- currPath
		}

		// recursively walk all of this subdirectory's contents
		for _, info := range contentInfo {
			err := l.Walk(path.Join(currPath, info.Name()), depth+1)
			if nil != err {
				warnLog.Log(err)
			}
		}
		return nil

	case (mode & os.ModeSymlink) > 0:
		// symlinks currently unhandled
		return NewErrorCode(EInvalidFile, fmt.Sprintf("not following symlinks (unhandled), skipping: %q", relPath))

	case (mode & (os.ModeDevice | os.ModeNamedPipe | os.ModeSocket | os.ModeCharDevice)) > 0:
		// file is not a regular file, not supported
		return NewErrorCode(EInvalidFile, fmt.Sprintf("%q", relPath))
	}

	// if we made it here, we have a regular file. add it as a media candidate
	// only if we were able to successfully os.Lstat() it
	media, errCode := NewMedia(l, fileInfo, currPath)
	if nil != errCode {
		return errCode
	}

	// append the media to this subdirectory's list of content/filenames
	l.media[currDir] = append(l.media[currDir], media)

	// update library counters
	l.totalMedia += 1
	l.totalSize += media.Size()

	// finally, notify the consumer of our media discovery
	l.sigMedia <- media

	return nil
}

func (l *Library) Scan() *ErrorCode {

	var err *ErrorCode = nil

	// the sigWork channel is buffered so that we can limit the number of
	// scanners concurrently operating on this library. the writes will fail and
	// fallback on the default select case if another scanner has already pushed
	// a value into the sigWork channel. so be sure to check the return value
	// when calling Scan()!
	select {
	case l.sigWork <- true:
		err = l.Walk(l.path, 1)
		<-l.sigWork
	default:
		err = NewErrorCode(ELibraryBusy, fmt.Sprintf("cannot scan library until current scan finishes %q", l.path))
	}
	return err
}
