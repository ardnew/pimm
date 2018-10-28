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
	"math"
	"os"
	"path/filepath"
	"strings"
)

const (
	dataConfigFileName  = "data-config.json"
	dataConfigFilePerms = 0644

	kibiBytes = 1024
	mebiBytes = 1048576

	// see type JSONDataConfig for a description of these items
	defaultDBRecMaxSize = 1 * mebiBytes
	defaultDBBufferSize = 4 * defaultDBRecMaxSize
	defaultDBBucketSize = 16
	defaultDBHashGrowth = 4 * mebiBytes // 32 * mebiBytes
	defaultDBHashedBits = 13            // 16
	defaultDBNumBuckets = 8192          // 65536
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
	options      *Options `json:"-"`              // not stored in the json data
	DBRecMaxSize int      `json:"DocMaxRoom"`     // <=- maximum size of a single document that will ever be accepted into database.
	DBBufferSize int      `json:"ColFileGrowth"`  // <=- size (in bytes) of each collection's pre-allocated files.
	DBBucketSize int      `json:"PerBucket"`      // number of entries pre-allocated to each hash table bucket.
	DBHashGrowth int      `json:"HTFileGrowth"`   // size (in bytes) to grow hash table file to fit in more entries.
	DBHashedBits uint     `json:"HashBits"`       // number of bits to consider for hashing indexed key, also determines the initial number of buckets in a hash table file.
	DBNumBuckets int      `json:"InitialBuckets"` // number of buckets initially allocated in a hash table file.
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

	bits := uint(math.Log2(float64(opt.DBHashSize.int) / 512.0))
	buckets := 1 << bits
	bucketSize := defaultDBBucketSize

	return &JSONDataConfig{
		options:      opt,
		DBRecMaxSize: opt.DBMaxRecordSize.int,
		DBBufferSize: opt.DBBufferSize.int,
		DBBucketSize: int(bucketSize),
		DBHashGrowth: opt.DBHashSize.int,
		DBHashedBits: uint(bits),
		DBNumBuckets: int(buckets),
	}, nil
}

// function marshal() marshals a configuration struct into a json string capable
// of being read from a file by the tiedot runtime.
func (c *JSONDataConfig) marshal(indent bool) ([]byte, *ReturnCode) {

	var js []byte
	var err error

	if indent {
		js, err = json.MarshalIndent(c, "", "  ")
	} else {
		js, err = json.Marshal(c)
	}
	if nil != err {
		return nil, rcInvalidJSONData.withInfof(
			"marshal(%s): cannot marshal struct into JSON object: %s", c, err)
	}
	return js, nil
}

// function unmarshal() unmarshals the tiedot-defined json configuration string
// into a JSONDataConfig{} struct.
func (c *JSONDataConfig) unmarshal(js []byte) *ReturnCode {

	if err := json.Unmarshal(js, c); nil != err {
		return rcInvalidJSONData.withInfof(
			"unmarshal(%q): cannot unmarshal JSON object into struct: %s",
			string(js), err)
	}
	return nil
}

// function equals() performs a field-by-field logical comparison of two
// JSONDataConfig{} structs returning true if and only if the fields are equal.
// a slice of strings containing the corresponding command-line option names of
// all unequal fields is returned. an empty slice is returned if all fields are
// equal or the argument references point to the same object.
func (c *JSONDataConfig) equals(jdc *JSONDataConfig) (bool, []string) {

	uneq := []string{}

	if c != jdc {
		// these fields are the only options the user can specify on the command
		// line. all other fields are calculated based on these.
		if c.DBRecMaxSize != jdc.DBRecMaxSize {
			uneq = append(uneq, c.options.DBMaxRecordSize.name)
		}
		if c.DBBufferSize != jdc.DBBufferSize {
			uneq = append(uneq, c.options.DBBufferSize.name)
		}
		if c.DBHashGrowth != jdc.DBHashGrowth {
			uneq = append(uneq, c.options.DBHashSize.name)
		}
	}
	return 0 == len(uneq), uneq
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

	// configure the database based on current Options struct -- this may be
	// user-provided values, default values, or a combination of the two; it
	// depends on whether or not the user overwrote the default values using
	// their command-line flags.
	jdc, ret := newJSONDataConfig(opt)
	if nil != ret {
		return nil, ret
	}

	userDefinedConfig, userOptions := opt.providedDBConfig()

	// check if a config file already exists; i.e. if a config file already
	// exists, then we assume this library's database has already been
	// configured (and the library possibly even scanned) sometime in the past.
	jsPath := filepath.Join(path, dataConfigFileName)
	if exists, _ := goutil.PathExists(jsPath); exists {

		// this is a known library, so its database configuration has already
		// been defined. verify the user isn't trying to change the database
		// configuration, because our db driver doesn't support reconfiguration
		// of a populated database.
		if userDefinedConfig {

			// the user has provided database configuration parameters via
			// command-line, so we need to compare them against the
			// configuration file on disk. read the file from disk.
			jdcPrev := &JSONDataConfig{}
			jsPrev, err := ioutil.ReadFile(jsPath)
			if nil != err {
				return nil, rcDatabaseError.withInfof(
					"newDatabase(%q, %q): ioutil.ReadFile(%q): %s",
					abs, dat, jsPath, err)
			}

			// now unmarshal the file's json string into a configuration struct.
			if ret := jdcPrev.unmarshal(jsPrev); nil != ret {
				return nil, ret
			}

			// construct a string of all of the user's actual command-line
			// arguments they provided to configure the database.
			given := make([]string, len(userOptions))
			for i, str := range userOptions {
				given[i] = "-" + str
			}
			csv := strings.Join(given, ", ")

			// verify the arguments are the same as the database configuration.
			// if they are not the same, bail out with an insanely long and
			// verbose reason and instructions to remedy the situation.
			// note that this is a limitation of the current database driver
			// "tiedot". if another database is used, be sure to revisit this.
			if equals, _ := jdc.equals(jdcPrev); !equals {
				errLog.logf(
					"you must delete the current database (%q) and rescan the "+
						"library to use a different database configuration. "+
						"otherwise, please remove one or more of the "+
						"following command-line options: %s", path, csv)
				return nil, rcDatabaseError.withInfof(
					"cannot reconfigure the storage/performance parameters " +
						"of an existing library database. one or more " +
						"command-line options provided are not compatible " +
						"with the current database configuration.")
			}

			// if we didn't die in the previous conditional, then the options
			// the user provided are the same as the current configuration.
			warnLog.verbosef(
				"database already configured, ignoring redundant "+
					"command-line options: %s", csv)
		}

	} else {

		// this is an unknown library. we are creating the database for the
		// first time and so need a database configuration file in json format
		// written to the database directory.

		// marshal the configuration struct into a json string for writing into
		// the config file which is read by and used by the tiedot runtime.
		js, ret := jdc.marshal(true)
		if nil != ret {
			return nil, ret
		}

		// flush the formatted json string to the config file on disk. this is
		// the permanent configuration used by the database runtime from now on
		// and cannot be changed.
		if err := ioutil.WriteFile(jsPath, js, dataConfigFilePerms); nil != err {
			return nil, rcDatabaseError.withInfof(
				"newDatabase(%q, %q): ioutil.WriteFile(%q, %s, %d): %s",
				abs, dat, jsPath, js, dataConfigFilePerms, err)
		}

		// notify the user if the database configuration written to file came
		// from the user's command-line options or the hard-coded defaults.
		if userDefinedConfig {
			infoLog.verbosef(
				"created database configuration file with user-defined options: %q (%s)",
				dataConfigFileName, sum)
		} else {
			infoLog.verbosef(
				"created database configuration file with default options: %q (%s)",
				dataConfigFileName, sum)
		}
	}

	// open the actual persistent data store if it exists; otherwise, create it.
	store, err := db.OpenDB(path)
	if nil != err {
		return nil, rcDatabaseError.withInfof(
			"newDatabase(%q, %q): db.OpenDB(%q): %s", abs, dat, path, err)
	}

	// initialize the new struct object.
	base := &Database{
		absPath: path,
		libPath: abs,
		name:    sum,
		dataDir: dat,
		store:   store,
	}

	// initialize the backing data store by creating the required collections;
	// returns to the caller any error it may have encountered.
	if ok, ret := base.initialize(); !ok {
		return nil, ret
	}

	// no errors caused an early return, so return the new struct object and a
	// nil ReturnCode to indicate success.
	return base, nil
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
			infoLog.tracef("created database collection: %q (%s)", name, d.name)
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
