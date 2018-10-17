// =============================================================================
//  PROJ: pimmp
//  AUTH: ardnew
//  DATE: 16 Oct 2018
//  FILE: database.go
// -----------------------------------------------------------------------------
//
//  DESCRIPTION
//    defines types and functions for interacting with an on-disk persistent
//    storage container. this is primarily used to store the media content in
//    an indexed file structure of some sort with fast read-write access.
//
// =============================================================================

package main

import (
	"ardnew.com/goutil"

	"github.com/HouzuoGuo/tiedot/db"
	//"github.com/HouzuoGuo/tiedot/dberr"

	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	colAudio = "Audio"
	colVideo = "Video"
)

type Database struct {
	absPath string // absolute path to database directory
	libPath string // absolute path to library
	name    string // libPath checksum (name of database directory)
	dataDir string // directory containing all known library databases

	db *db.DB // interactive database object
}

func newDatabase(abs string, dat string) (*Database, *ReturnCode) {

	// common logic for constructing and throwing an error ReturnCode
	invalidDatabase := func(a, d string, e error) *ReturnCode {
		info := fmt.Sprintf("newDatabase(%q, %q): %s", a, d, e)
		return rcInvalidDatabase.withInfo(info)
	}

	sum := strings.ToLower(goutil.MD5(abs))
	path := filepath.Join(dat, sum)

	// verify or create the database if it doesn't exist
	if exists, _ := goutil.PathExists(path); !exists {
		if err := os.MkdirAll(path, os.ModePerm); nil != err {
			return nil, invalidDatabase(abs, dat, err)
		}
		infoLog.vlog(fmt.Sprintf("created database: %q (%q)", sum, abs))
	}

	db, err := db.OpenDB(path)
	if nil != err {
		return nil, invalidDatabase(abs, dat, err)
	}

	var hasAudio, hasVideo bool
	for _, name := range db.AllCols() {
		n := strings.TrimSpace(name)
		hasAudio = hasAudio || n == colAudio
		hasVideo = hasVideo || n == colVideo
	}

	createCol := func(has bool, col, sum string) *ReturnCode {
		if !has {
			if err := db.Create(col); nil != err {
				return invalidDatabase(abs, dat, err) // TODO: NEW ReturnCode
			}
			infoLog.vlog(fmt.Sprintf("created collection: %q (%s)", col, sum))
		}
		return nil
	}

	if ret := createCol(hasAudio, colAudio, sum); nil != ret {
		return nil, ret
	}
	if ret := createCol(hasVideo, colVideo, sum); nil != ret {
		return nil, ret
	}

	err = db.Close()
	if nil != err {
		return nil, invalidDatabase(abs, dat, err) // TODO: NEW ReturnCode
	}

	return &Database{
		absPath: path,
		libPath: abs,
		name:    sum,
		dataDir: dat,

		db: db,
	}, nil
}

// creates a string representation of the Database for easy identification in
// logs
func (d *Database) String() string {
	return fmt.Sprintf("{%q,%s}", d.dataDir, d.name)
}
