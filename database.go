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

	//	"github.com/HouzuoGuo/tiedot/db"
	//  "github.com/HouzuoGuo/tiedot/dberr"

	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Database struct {
	dbPath  string // absolute path to database directory
	absPath string // absolute path to library
	name    string // absPath checksum (name of database directory)
	dataDir string // directory containing all known library databases
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

	return &Database{path, abs, sum, dat}, nil
}
