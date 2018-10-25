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

// variable colName maps the MediaKind enum values to the string name of their
// corresponding collection in the database.
var (
	colName = [mkCOUNT]string{
		"Audio", // 0 = mkAudio
		"Video", // 1 = mkVideo
	}
)

// TBD: create tiedot cache/index JSON config file
//import (
//    "encoding/json"
//    "fmt"
//    "io/ioutil"
//)
//
//type Rankings struct {
//    Keyword  string
//    GetCount uint32
//    Engine   string
//    Locale   string
//    Mobile   bool
//}
//
//func main() {
//    var jsonBlob = []byte(`
//        {"keyword":"hipaa compliance form", "get_count":157, "engine":"google", "locale":"en-us", "mobile":false}
//    `)
//    rankings := Rankings{}
//    err := json.Unmarshal(jsonBlob, &rankings)
//    if err != nil {
//        // nozzle.printError("opening config file", err.Error())
//    }
//
//    rankingsJson, _ := json.Marshal(rankings)
//    err = ioutil.WriteFile("output.json", rankingsJson, 0644)
//    fmt.Printf("%+v", rankings)
//}

// type JSONDataConfig defines all of tiedot's configurable paramters for
// initial index amd cache sizes
type JSONDataConfig struct {
	docMaxRoom    int  // <=- maximum size of a single document that will ever be accepted into database.
	colFileGrowth int  // <=- size (in bytes) to grow collection data file when new documents have to fit in.
	perBucket     int  // number of entries pre-allocated to each hash table bucket.
	htFileGrowth  int  // size (in bytes) to grow hash table file to fit in more entries.
	hashBits      uint // number of bits to consider for hashing indexed key, also determines the initial number of buckets in a hash table file.

	initialBuckets int    `json:"-"` // number of buckets initially allocated in a hash table file.
	padding        string `json:"-"` // pre-allocated filler (space characters) for new documents.
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
func newDatabase(abs string, dat string) (*Database, *ReturnCode) {

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
