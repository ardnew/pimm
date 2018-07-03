package main

import (
	"fmt"
	"os"
	"path"
)

type Library struct {
	path       string
	depthLimit uint
	ignored    []string
}

const (
	LibraryDepthUnlimited = 0
)

func NewLibrary(p string) (*Library, *ErrorCode) {

	p = path.Clean(p)

	f, err := os.Open(p)

	if nil != err {
		return nil, NewErrorCode(EInvalidLibrary, fmt.Sprintf("%s: %q", err, p))
	}
	defer f.Close()

	_, err = f.Readdir(0)

	if nil != err {
		return nil, NewErrorCode(EInvalidLibrary, fmt.Sprintf("%s: %q", err, p))
	}

	return &Library{path: p, depthLimit: LibraryDepthUnlimited, ignored: make([]string, 0)}, nil
}

func (l *Library) String() string {
	return fmt.Sprintf("%q:{ depthLimit:%d ignored:%v }", l.path, l.depthLimit, l.ignored)
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

func (l *Library) walk(currPath string, depth uint) *ErrorCode {

	// read fs attributes to determine how we handle the file
	fileInfo, err := os.Lstat(currPath)
	if nil != err {
		return NewErrorCode(EFileStat, fmt.Sprintf("%s: %q", err, currPath))
	}
	mode := fileInfo.Mode()

	switch {
	case (mode & os.ModeDir) > 0:
		// file is directory, walk its contents unless we are at max depth
		if l.IsDepthLimited() && depth > l.depthLimit {
			return NewErrorCode(EDirDepth, fmt.Sprintf("exceeded limit=%d, skipping: %q", l.depthLimit, currPath))
		}
		dir, err := os.Open(currPath)
		if nil != err {
			return NewErrorCode(EDirOpen, fmt.Sprintf("%s: %q", err, currPath))
		}
		contentInfo, err := dir.Readdir(0)
		dir.Close()
		if nil != err {
			return NewErrorCode(EDirOpen, fmt.Sprintf("%s: %q", err, currPath))
		}
		for _, info := range contentInfo {
			err := l.walk(path.Join(currPath, info.Name()), depth+1)
			if nil != err {
				WarnLog.Log(err)
			}
		}
		return nil

	case (mode & os.ModeSymlink) > 0:
		return NewErrorCode(EInvalidFile, fmt.Sprintf("not following symlinks, skipping: %q", currPath))

	case (mode & (os.ModeDevice | os.ModeNamedPipe | os.ModeSocket | os.ModeCharDevice)) > 0:
		// file is not a regular file, not supported
		return NewErrorCode(EInvalidFile, fmt.Sprintf("%q", currPath))
	}

	// if we made it here, we have a regular file. add it as a media candidate
	InfoLog.Log("adding: ", currPath)

	return nil
}

func (l *Library) Scan() *ErrorCode {

	err := l.walk(l.path, 1)
	if nil != err {
		switch err.Code {
		default:
			InfoLog.Log("walk(%q): %s", l.path, err)
		}
		return err
	}
	return nil
}
