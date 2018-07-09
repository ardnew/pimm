package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
)

type Library struct {
	workingDir string
	path       string
	name       string
	depthLimit uint
	ignored    []string
	subdir     chan string
	mediaChan  chan *Media
	media      map[string][]*Media
}

const (
	LibraryDepthUnlimited = 0
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

	return &Library{
		workingDir: dir,
		path:       libPath,
		name:       path.Base(libPath),
		depthLimit: LibraryDepthUnlimited,
		ignored:    ignore,
		subdir:     make(chan string),
		mediaChan:  make(chan *Media),
		media:      make(map[string][]*Media),
	}, nil
}

func (l *Library) String() string {
	return fmt.Sprintf("%q:{ depthLimit:%d ignored:%v }", l.path, l.depthLimit, l.ignored)
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

func (l *Library) Subdir() chan string {
	return l.subdir
}

func (l *Library) MediaChan() chan *Media {
	return l.mediaChan
}

func (l *Library) Media() map[string][]*Media {
	return l.media
}

func (l *Library) Walk(currPath string, depth uint) *ErrorCode {

	// TODO: don't continue if file matches an ignore pattern
	/*
		for _, p := range l.Ignored() {
			match, err := filepath.Match(p, currPath)
			if nil != err {
				return NewErrorCode(EInvalidOption, fmt.Sprintf("invalid match pattern=%q skipping: %q", p, currPath))
			}
			if match {
				return NewErrorCode(EFileIgnore, fmt.Sprintf("ignore pattern=%q skipping: %q", p, currPath))
			}
		}
	*/

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
		// don't add the library root as a subdir to itself
		if depth > 1 {
			l.subdir <- currPath
		}
		l.media[currPath] = []*Media{}
		for _, info := range contentInfo {
			err := l.Walk(path.Join(currPath, info.Name()), depth+1)
			if nil != err {
				warnLog.Log(err)
			}
		}
		return nil

	case (mode & os.ModeSymlink) > 0:
		return NewErrorCode(EInvalidFile, fmt.Sprintf("not following symlinks, skipping: %q", relPath))

	case (mode & (os.ModeDevice | os.ModeNamedPipe | os.ModeSocket | os.ModeCharDevice)) > 0:
		// file is not a regular file, not supported
		return NewErrorCode(EInvalidFile, fmt.Sprintf("%q", relPath))
	}

	// if we made it here, we have a regular file. add it as a media candidate
	media, errCode := NewMedia(fileInfo, currPath)
	if nil != errCode {
		return errCode
	}

	mediaDir := path.Dir(currPath)
	l.media[mediaDir] = append(l.media[mediaDir], media)
	l.mediaChan <- media

	return nil
}

func (l *Library) Scan() *ErrorCode {

	err := l.Walk(l.path, 1)
	if nil != err {
		switch err.Code {
		default:
			infoLog.Log("walk(%q): %s", l.path, err)
		}
		return err
	}
	return nil
}
