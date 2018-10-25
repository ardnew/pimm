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

	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

const (
	dataConfigFileName  = "data-config.json"
	dataConfigFilePerms = 0644

	mebiBytes           = 1048576
	defaultDBRecMaxSize = 2 * mebiBytes
	defaultDBBufferSize = 32 * mebiBytes
	defaultDBBucketSize = 16
	defaultDBHashGrowth = 32 * mebiBytes
	defaultDBNumBuckets = 16
)

// variable colName maps the MediaKind enum values to the string name of their
// corresponding collection in the database.
var (
	colName = [mkCOUNT]string{
		"Audio", // 0 = mkAudio
		"Video", // 1 = mkVideo
	}
)

// type JSONDataConfig defines all of tiedot's configurable paramters for
// initial index and cache sizes
type JSONDataConfig struct {
	DBRecMaxSize int  `json:"DocMaxRoom"`    // <=- maximum size of a single document that will ever be accepted into database.
	DBBufferSize int  `json:"ColFileGrowth"` // <=- size (in bytes) of each collection's pre-allocated files.
	DBBucketSize int  `json:"PerBucket"`     // number of entries pre-allocated to each hash table bucket.
	DBHashGrowth int  `json:"HTFileGrowth"`  // size (in bytes) to grow hash table file to fit in more entries.
	DBNumBuckets uint `json:"HashBits"`      // number of bits to consider for hashing indexed key, also determines the initial number of buckets in a hash table file.
}

// creates a string representation of the Database for easy identification in
// logs.
func (c *JSONDataConfig) String() string {
	r := strings.NewReplacer("\r", "", "\n", "")
	return r.Replace(fmt.Sprintf("%#v", c))
}

// function newJSONDataConfig() creates the struct that configures tiedot's
// index/cache sizing options. this struct is intended to be marshalled into
// a json string and stored in a file read by the tiedot runtime.
func newJSONDataConfig(opt *Options) (*JSONDataConfig, *ReturnCode) {

	if nil == opt {
		return nil, rcInvalidJSONData.withInfo(
			"newJSONDataConfig(): cannot encode JSON object: &Options{} is nil")
	}
	return &JSONDataConfig{
		DBRecMaxSize: opt.DBRecMaxSize.int,
		DBBufferSize: opt.DBBufferSize.int,
		DBBucketSize: opt.DBBucketSize.int,
		DBHashGrowth: opt.DBHashGrowth.int,
		DBNumBuckets: opt.DBNumBuckets.uint,
	}, nil
}

// function marshal() marshals a configuration struct into a json string capable
// of being read from a file by the tiedot runtime.
func (c *JSONDataConfig) marshal(indent bool) ([]byte, *ReturnCode) {

	var (
		js  []byte
		err error
	)
	if indent {
		js, err = json.MarshalIndent(c, "", "  ")
	} else {
		js, err = json.Marshal(c)
	}
	if nil != err {
		// TBD --
		return nil, rcInvalidJSONData.withInfof(
			"marshal(%s): cannot marshal struct into JSON object: %s", c, err)
	}
	return js, nil
}

// type Database represents an abstraction from the internal persistant storage
// mechanism used for maintaining an index of all known libraries and their
// respective media content.
type Database struct {
	absPath string // absolute path to database directory
	libPath string // absolute path to library
	name    string // libPath checksum (name of database directory)
	dataDir string // directory containing all known library databases

	store *db.DB // interactive database object
}

// function newDatabase() creates a new high-level database object through
// which all of the persistent storage operations should be performed.
func newDatabase(opt *Options, abs string, dat string) (*Database, *ReturnCode) {

	// compute an identifying checksum from the absolute path to the library,
	// and use that to build a path to the database directory.
	sum := strings.ToLower(goutil.MD5(abs))
	path := filepath.Join(dat, sum)

	// verify or create the database directory if it doesn't exist.
	if exists, _ := goutil.PathExists(path); !exists {
		if err := os.MkdirAll(path, os.ModePerm); nil != err {
			return nil, rcInvalidDatabase.withInfof(
				"newDatabase(%q, %q): os.MkdirAll(%q): %s", abs, dat, path, err)
		}
		infoLog.tracef("created database: %q (%q)", sum, abs)
	}

	// configure tiedot's index/cache sizes from our Options struct
	jdc, ret := newJSONDataConfig(opt)
	if nil != ret {
		return nil, ret
	}

	// marshal the configuration struct into a json string for handing over to
	// the tiedot runtime.
	js, ret := jdc.marshal(true)
	if nil != ret {
		return nil, ret
	}

	jdcPath := filepath.Join(path, dataConfigFileName)
	if exists, _ := goutil.PathExists(jdcPath); !exists {
		if err := ioutil.WriteFile(jdcPath, js, dataConfigFilePerms); nil != err {
			return nil, rcInvalidDatabase.withInfof(
				"newDatabase(%q, %q): ioutil.WriteFile(%q, %s, %d): %s",
				abs, dat, jdcPath, js, dataConfigFilePerms, err)
		}
		infoLog.tracef("created database configuration file: %q", dataConfigFileName)
	}

	// open the database, creating it if it doesn't already exist.
	store, err := db.OpenDB(path)
	if nil != err {
		return nil, rcDatabaseError.withInfof(
			"newDatabase(%q, %q): db.OpenDB(%q): %s", abs, dat, path, err)
	}

	// initialize the new struct object.
	dbase := &Database{
		absPath: path,
		libPath: abs,
		name:    sum,
		dataDir: dat,
		store:   store,
	}

	// initialize the backing data store by creating the required collections;
	// returns to the caller any error it may have encountered.
	if ok, ret := dbase.initialize(); !ok {
		return nil, ret
	}

	// no errors caused an early return, so return the new struct object and a
	// nil ReturnCode to indicate success.
	return dbase, nil
}

// creates a string representation of the Database for easy identification in
// logs.
func (d *Database) String() string {
	return fmt.Sprintf("{%q,%s}", d.dataDir, d.name)
}

// function close() closes the backing data store. returns true on success, and
// returns false with a diagnostic ReturnCode on failure.
func (d *Database) close() (bool, *ReturnCode) {

	err := d.store.Close()
	if nil != err {
		return false, rcDatabaseError.withInfof("close(%s): %s", d, err)
	}
	return true, nil
}

// function initialize() creates the required collections in the backing data
// store. returns true on success, and returns false with a diagnostic
// ReturnCode on failure.
func (d *Database) initialize() (bool, *ReturnCode) {

	// iterate over all required collection names
	for _, name := range colName {
		// verify it is available
		if !d.store.ColExists(name) {
			// otherwise, collection doesn't exist -- create it
			infoLog.tracef("creating database collection: %q (%s)", name, d.name)
			if err := d.store.Create(name); nil != err {
				return false, rcDatabaseError.withInfof(
					"initialize(): %s: Create(%q): %s", d, name, err)
			}
			infoLog.tracef("finished creating database collection: %q (%s)", name, d.name)
		}
	}
	return true, nil
}

func (d *Database) scrub() {
	for _, name := range colName {
		if d.store.ColExists(name) {
			d.store.Scrub(name)
		}
	}
}
