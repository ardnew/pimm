package main

import (
	"fmt"
	//"io/ioutil"
	"os"
	"path"
	"time"
	//"github.com/cespare/xxhash"
)

type Media struct {
	dir     string    // absolute path to the parent dir
	name    string    // file name
	path    string    // absolute path to the file
	size    int64     // file size (in bytes)
	mtime   time.Time // file last modification time (local time)
	hash    uint64    // TBD: hash digest of file content
	library *Library  // library to which this media belongs
}

func NewMedia(library *Library, info os.FileInfo, absPath string) (*Media, *ErrorCode) {

	fh, err := os.Open(absPath)
	if nil != err {
		return nil, NewErrorCode(EFileHash, fmt.Sprintf("failed to open file: %q", absPath))
	}
	defer fh.Close()

	// TBD: spawn the checksum calculation off on its own
	//bytes, err := ioutil.ReadAll(fh)
	//if nil != err {
	//	return nil, NewErrorCode(EFileHash, fmt.Sprintf("failed to compute file hash: %q", absPath))
	//}
	//done := make(chan uint64)
	//go func(bytes []byte) {
	//	done <- xxhash.Sum64(bytes)
	//}(bytes)
	hash := uint64(0)

	dir := path.Dir(absPath)
	name := path.Base(absPath)

	media := Media{
		dir:     dir,
		name:    name,
		path:    absPath,
		size:    info.Size(),
		mtime:   info.ModTime().Local(),
		hash:    hash,
		library: library,
	}
	return &media, nil
}

func (m *Media) String() string {
	return fmt.Sprintf("%q{%s}#%x", m.name, m.path, m.hash)
}

func (m *Media) Dir() string {
	return m.dir
}

func (m *Media) Name() string {
	return m.name
}

func (m *Media) Path() string {
	return m.path
}

func (m *Media) Size() int64 {
	return m.size
}

func (m *Media) SizeStr(showBytes bool) string {
	return SizeStr(m.size, showBytes)
}

func (m *Media) MTime() time.Time {
	return m.mtime
}

func (m *Media) MTimeStr() string {
	return fmt.Sprintf(
		"%d-%02d-%02d %02d:%02d:%02d",
		m.mtime.Year(), m.mtime.Month(), m.mtime.Day(),
		m.mtime.Hour(), m.mtime.Minute(), m.mtime.Second(),
	)
}

func (m *Media) Hash() uint64 {
	return m.hash
}

func (m *Media) HashStr() string {
	return fmt.Sprintf("%x", m.hash)
}
