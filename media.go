package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/cespare/xxhash"
)

type Media struct {
	dir   string
	name  string
	path  string
	size  int64
	mtime time.Time
	hash  uint64
}

func NewMedia(info os.FileInfo, absPath string) (*Media, *ErrorCode) {

	f, err := os.Open(absPath)
	if nil != err {
		return nil, NewErrorCode(EFileHash, fmt.Sprintf("failed to open file for hashing: %q", absPath))
	}
	defer f.Close()

	b, err := ioutil.ReadAll(f)
	if nil != err {
		return nil, NewErrorCode(EFileHash, fmt.Sprintf("failed to compute file hash: %q", absPath))
	}

	// TBD: spawn the checsum calculation off on its own
	done := make(chan uint64)
	go func(b []byte) {
		done <- xxhash.Sum64(b)
	}(b)
	hash := <-done

	dir := path.Dir(absPath)
	name := path.Base(absPath)

	return &Media{
		dir:   dir,
		name:  name,
		path:  absPath,
		size:  info.Size(),
		mtime: info.ModTime().Local(),
		hash:  hash,
	}, nil
}

func (m *Media) String() string {
	return fmt.Sprintf("%s [%x]", m.name, m.hash)
}

func (m *Media) Columns() []string {
	return []string{
		m.name,
		m.MTimeStr(),
		m.SizeStr(),
		m.HashStr(),
	}
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

func (m *Media) SizeStr() string {
	return fmt.Sprintf("%d", m.size)
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
