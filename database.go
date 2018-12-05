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
	//"github.com/davecgh/go-spew/spew"

	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	dataConfigFileName  = "data-config.json"
	dataConfigFilePerms = 0644

	kibiBytes = 1024
	mebiBytes = 1048576
)

var (
	// see type JSONDataConfig for a description of these items
	defaultMaxRecordSize  = 64 * kibiBytes
	defaultDiskBufferSize = 4 * defaultMaxRecordSize / runtime.NumCPU()
	defaultHashBucketSize = 16
	defaultHashBufferSize = defaultDiskBufferSize / 4
	defaultHashedBitsSize = 13
	defaultNumHashBuckets = 8192
)

// type JSONDataConfig defines all of tiedot's configurable paramters for
// initial index and cache sizes
type JSONDataConfig struct {
	options        *Options // not stored in the json data
	MaxRecordSize  int      `json:"DocMaxRoom"`     // <=- maximum size of a single document that will ever be accepted into database.
	DiskBufferSize int      `json:"ColFileGrowth"`  // <=- size (in bytes) of each collection's pre-allocated files.
	HashBucketSize int      `json:"PerBucket"`      // number of entries pre-allocated to each hash table bucket.
	HashBufferSize int      `json:"HTFileGrowth"`   // size (in bytes) to grow hash table file to fit in more entries.
	HashedBitsSize uint     `json:"HashBits"`       // number of bits to consider for hashing indexed key, also determines the initial number of buckets in a hash table file.
	NumHashBuckets int      `json:"InitialBuckets"` // number of buckets initially allocated in a hash table file.
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
		return nil, rcInvalidJSONData.spec(
			"newJSONDataConfig(): cannot encode JSON object: &Options{} is nil")
	}

	bits := uint(math.Log2(float64(opt.HashBufferSize.int) / 512.0))
	buckets := 1 << bits
	recordSizeMax := defaultMaxRecordSize
	bucketSize := defaultHashBucketSize

	return &JSONDataConfig{
		options:        opt,
		MaxRecordSize:  int(recordSizeMax),
		DiskBufferSize: opt.DiskBufferSize.int,
		HashBucketSize: int(bucketSize),
		HashBufferSize: opt.HashBufferSize.int,
		HashedBitsSize: uint(bits),
		NumHashBuckets: int(buckets),
	}, nil
}

// function marshal() marshals a configuration struct into a json string capable
// of being read from a file by the tiedot runtime.
func (c *JSONDataConfig) marshal(indent bool) ([]byte, *ReturnCode) {

	var data []byte
	var err error

	if indent {
		data, err = json.MarshalIndent(c, "", "  ")
	} else {
		data, err = json.Marshal(c)
	}
	if nil != err {
		return nil, rcInvalidJSONData.specf(
			"marshal(%s): cannot marshal struct into JSON object: %s", c, err)
	}
	return data, nil
}

// function unmarshal() unmarshals the tiedot-defined json configuration string
// into a JSONDataConfig{} struct.
func (c *JSONDataConfig) unmarshal(data []byte) *ReturnCode {

	if err := json.Unmarshal(data, c); nil != err {
		return rcInvalidJSONData.specf(
			"unmarshal(%q): cannot unmarshal JSON object into struct: %s",
			string(data), err)
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
		if c.DiskBufferSize != jdc.DiskBufferSize {
			uneq = append(uneq, c.options.DiskBufferSize.name)
		}
		if c.HashBufferSize != jdc.HashBufferSize {
			uneq = append(uneq, c.options.HashBufferSize.name)
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

	store          *db.DB                  // interactive database object
	col            [ecCOUNT][]*db.Col      // db collections referenced by MediaKind
	colName        [ecCOUNT][]string       // name of each collection
	index          [ecCOUNT][]*EntityIndex // indices on each collection
	numRecordsLoad [ecCOUNT][]uint         // number of records in each media collection discovered by load()
	numRecordsScan [ecCOUNT][]uint         // number of records in each media collection discovered by scan()
	timeCreated    time.Time               // only set if the db was newly created, else IsZero() will return true
}

// type RecordID offers a tuple object storing any given type with an integer ID
// which tiedot uses as its primary (hash) key for locating records in any given
// collection.
type RecordID struct {
	id  int
	rec interface{}
}

// function newDatabase() creates a new high-level database object through
// which all of the persistent storage operations should be performed.
func newDatabase(opt *Options, abs string, dat string) (*Database, *ReturnCode) {

	// zeroized Time object is January 1, year 1, 00:00:00.000000000 UTC
	// calling time.IsZero() with this value will return true, alternatively,
	// calling our (*Database).isFirstAppearance() will also return true.
	timeCreated := time.Time{}

	// compute an identifying checksum from the absolute path to the library,
	// and use that to build a path to the database directory.
	sum := strings.ToLower(goutil.MD5(abs))
	path := filepath.Join(dat, sum)

	// verify or create the database directory if it doesn't exist.
	if exists, _ := goutil.PathExists(path); !exists {
		if err := os.MkdirAll(path, os.ModePerm); nil != err {
			return nil, rcInvalidDatabase.specf(
				"newDatabase(%q, %q): os.MkdirAll(%q): %s", abs, dat, path, err)
		}
		infoLog.verbosef("creating library database: %q (%s)", abs, sum)
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
	configPath := filepath.Join(path, dataConfigFileName)
	if exists, _ := goutil.PathExists(configPath); exists {

		// this is a known library, so its database configuration has already
		// been defined. verify the user isn't trying to change the database
		// configuration, because our db driver doesn't support reconfiguration
		// of a populated database.
		if userDefinedConfig {

			// the user has provided database configuration parameters via
			// command-line, so we need to compare them against the
			// configuration file on disk. read the file from disk.
			jdcPrev := &JSONDataConfig{}
			dataPrev, err := ioutil.ReadFile(configPath)
			if nil != err {
				return nil, rcDatabaseError.specf(
					"newDatabase(%q, %q): ioutil.ReadFile(%q): %s",
					abs, dat, configPath, err)
			}

			// now unmarshal the file's json string into a configuration struct.
			if ret := jdcPrev.unmarshal(dataPrev); nil != ret {
				return nil, ret
			}

			// construct a string of all of the user's actual command-line
			// arguments they provided to configure the database.
			args := make([]string, len(userOptions))
			for i, str := range userOptions {
				args[i] = "-" + str
			}
			csv := strings.Join(args, ", ")

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
				return nil, rcDatabaseError.specf(
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
		timeCreated = time.Now()

		// marshal the configuration struct into a json string for writing into
		// the config file which is read by and used by the tiedot runtime.
		data, ret := jdc.marshal(true)
		if nil != ret {
			return nil, ret
		}

		// flush the formatted json string to the config file on disk. this is
		// the permanent configuration used by the database runtime from now on
		// and cannot be changed.
		if err := ioutil.WriteFile(configPath, data, dataConfigFilePerms); nil != err {
			return nil, rcDatabaseError.specf(
				"newDatabase(%q, %q): ioutil.WriteFile(%q, %s, %d): %s",
				abs, dat, configPath, data, dataConfigFilePerms, err)
		}

		// notify the user if the database configuration written to file came
		// from the user's command-line options or the hard-coded defaults.
		if userDefinedConfig {
			infoLog.tracef(
				"created database configuration file with user-defined options: %q (%s)",
				dataConfigFileName, sum)
		} else {
			infoLog.tracef(
				"created database configuration file with default options: %q (%s)",
				dataConfigFileName, sum)
		}
	}

	// open the actual persistent data store if it exists; otherwise, create it.
	store, err := db.OpenDB(path)
	if nil != err {
		return nil, rcDatabaseError.specf(
			"newDatabase(%q, %q): db.OpenDB(%q): %s", abs, dat, path, err)
	}

	// initialize the new struct object.
	base := &Database{
		absPath:        path,
		libPath:        abs,
		name:           sum,
		dataDir:        dat,
		store:          store,
		col:            [ecCOUNT][]*db.Col{},
		colName:        [ecCOUNT][]string{},
		index:          [ecCOUNT][]*EntityIndex{},
		numRecordsLoad: [ecCOUNT][]uint{},
		numRecordsScan: [ecCOUNT][]uint{},
		timeCreated:    timeCreated,
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

// function String() creates a string representation of the Database for easy
// identification in logs.
func (d *Database) String() string {
	return fmt.Sprintf("{%q,%s}", d.dataDir, d.name)
}

// function totalRecordsString() constructs a human-readable string describing
// the total number of entity records (as indicated by the Database object's
// counter fields) of a given class c and kind k. if class and/or kind is a
// negative value, then include all classes and/or kinds, respectively.
// also returned is the total sum, indiscriminated by class or kind.
func (d *Database) totalRecordsString(m DiscoveryMethod, c int, k int) (uint, string) {

	var numRecords *[ecCOUNT][]uint
	switch m {
	case dmLoad:
		numRecords = &d.numRecordsLoad
	case dmScan:
		numRecords = &d.numRecordsScan
	default:
		return 0, ""
	}

	total := uint(0)
	desc := ""
	for class, count := range *numRecords {
		if !(int(c) == class || c < 0) {
			continue
		}
		for kind, name := range d.colName[class] {
			if !(int(k) == kind || k < 0) {
				continue
			}
			if count[kind] > 0 {
				total += count[kind]
				if len(desc) > 0 {
					desc = fmt.Sprintf("%s, ", desc)
				}
				desc = fmt.Sprintf("%s%d %s", desc, count[kind], strings.ToLower(name))
			}
		}
	}
	return total, desc
}

// function close() closes the backing data store. returns true on success, and
// returns false with a diagnostic ReturnCode on failure.
func (d *Database) close() (bool, *ReturnCode) {

	err := d.store.Close()
	if nil != err {
		return false, rcDatabaseError.specf("close(%s): %s", d, err)
	}
	return true, nil
}

// function isFirstAppearance() inspects this Database's timeCreated field to
// determine if the data store was just created for the first time during this
// invocation of the program. the timeCreated (time.Time) field remains its
// initial zero-value if the Database already existed on disk and we are loading
// from it before we begin to (potentially) populate it.
func (d *Database) isFirstAppearance() bool {
	return !d.timeCreated.IsZero()
}

// function initialize() creates the required collections in the backing data
// store. returns true on success, and returns false with a diagnostic
// ReturnCode on failure.
func (d *Database) initialize() (bool, *ReturnCode) {

	for class := EntityClass(0); class < ecCOUNT; class++ {

		// create each of the collection slices, copying items as needed.
		numCol := len(entityColName[class])
		d.col[class] = make([]*db.Col, numCol)
		d.colName[class] = make([]string, numCol)
		d.numRecordsLoad[class] = make([]uint, numCol)
		d.numRecordsScan[class] = make([]uint, numCol)
		copy(d.colName[class], entityColName[class])

		// create each of the index slices, copying items as needed.
		numIndex := len(entityIndex[class])
		d.index[class] = make([]*EntityIndex, numIndex)
		copy(d.index[class], entityIndex[class])

		// iterate over all required collection names
		for kind, name := range d.colName[class] {
			// verify it is available
			existed := d.store.ColExists(name)
			if !existed {
				// otherwise, collection doesn't exist -- create it
				if err := d.store.Create(name); nil != err {
					return false, rcDatabaseError.specf(
						"initialize(): %s: Create(%q): %s", d, name, err)
				}
				infoLog.tracef("created database collection: %q (%s)", name, d.name)
			}

			// keep a reference to the collection handler
			d.col[class][kind] = d.store.Use(name)

			// install all class indices if this is a newly created collection.
			if !existed {
				for _, idx := range d.index[class] {
					if err := d.col[class][kind].Index(*idx); nil != err {
						return false, rcDatabaseError.specf(
							"initialize(): %s: Index(%q): %s", d, name, err)
					}
				}
			}
		}
	}
	return true, nil
}

// function scrub() fixes corrupt records and defragments disk space used by the
// database -- performed on all collections in the database.
func (d *Database) scrub() {

	for class, col := range d.col {
		for kind, name := range d.colName[class] {
			if d.store.ColExists(name) {
				d.store.Scrub(name)
			}
			// after Scrub(), tiedot has potentially reallocated space elsewhere and
			// the reference is probably no longer valid.
			col[kind] = d.store.Use(name)
		}
	}
}
