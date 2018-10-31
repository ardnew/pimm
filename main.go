// =============================================================================
//  PROJ: pimmp
//  AUTH: ardnew
//  DATE: 26 Sept 2018
//  FILE: main.go
// -----------------------------------------------------------------------------
//
//  DESCRIPTION
//    program entry-point and primary controller.
//
// =============================================================================

package main

import (
	"ardnew.com/goutil"

	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

// unexported constants
const (
	defaultConfigPath  = "config"
	defaultLibDataPath = "library.db"
)

// versioning information defined by compiler switches in Makefile.
var (
	identity  string
	version   string
	revision  string
	buildtime string
)

// type Option struct can contain any possible individual option configuration
// including its command line flag identifier and usage info..
type Option struct {
	name  string
	usage string
	bool
	int
	uint
	float64
	string
}

// type NamedOption is intended to map the name of an option to the actual
// *Option struct associated with it.
type NamedOption map[string]*Option

// type Options struct defines the collection of all command line options.
type Options struct {
	*flag.FlagSet // the builtin command-line parser

	Provided NamedOption // which options were provided by the user at runtime

	UsageHelp *Option // shows usage synopsis
	Verbose   *Option // prints additional status information
	Trace     *Option // prints very detailed status information
	Config    *Option // defines path to config file
	LibData   *Option // defines data directory path (where to store databases)

	DBBufferSize *Option // size (bytes) of each collection's pre-allocated buffers on disk. num buffers = num CPU cores
	DBHashSize   *Option // size (bytes) by which each hash table will grow once individual capacity is exceeded.
}

// function providedDBConfig() checks the "Provided" hash of the Options struct
// for any of the options related to initial database configuration. this is
// necessary to decide how to initialize the database. furthermore, a []string
// will be returned containing the name of each option the user provided.
func (o *Options) providedDBConfig() (bool, []string) {

	list := []string{}
	count := 0

	if d, ok := o.Provided[o.DBBufferSize.name]; ok {
		list = append(list, d.name)
		count++
	}
	if d, ok := o.Provided[o.DBHashSize.name]; ok {
		list = append(list, d.name)
		count++
	}

	return 0 != count, list
}

// function configDir() constructs the full path to the directory containing all
// of the program's supporting configuration data. if the user has defined a
// specific config file (via -config arg), then use the _logical_ parent
// directory of that file path; otherwise, use the default path "~/.<identity>".
func configDir(opt *Options) string {
	if nil == opt {
		return filepath.Join(homeDir(), fmt.Sprintf(".%s", identity))
	} else {
		return filepath.Dir(opt.Config.string)
	}
}

// function main() is the program entry point, obviously.
func main() {

	infoLog.verbose("parsing options ...")

	// parse options and command line arguments.
	options, err := initOptions()
	if nil != err {
		errLog.die(err, false)
	}

	// create the directory hierarchy that will store our configuration data
	// permanently on disk.
	configDir := configDir(options)
	config := options.Config.string
	if exists, _ := goutil.PathExists(config); !exists {
		if dirExists, _ := goutil.PathExists(configDir); !dirExists {
			if err := os.MkdirAll(configDir, os.ModePerm); nil != err {
				errLog.die(rcInvalidConfig.specf(
					"cannot create configuration directory: %q: %s", configDir, err), false)
			}
			infoLog.tracef("created configuration directory: %q", configDir)
		}

		// TODO: create configuration file
		infoLog.tracef("(TBD) -- created configuration: %q", config)
	}

	// if we haven't died yet, then config dir/file exists. load it.
	infoLog.tracef("(TBD) -- loading configuration: %q", config)

	// create the directory hierarchy that will store our libraries' backing
	// data stores permanently on disk.
	libData := options.LibData.string
	if exists, _ := goutil.PathExists(libData); !exists {
		if err := os.MkdirAll(libData, os.ModePerm); nil != err {
			errLog.die(rcInvalidConfig.specf(
				"cannot create library data directory: %q: %s", libData, err), false)
		}
		infoLog.tracef("created library data directory: %q", libData)
	} else {
		infoLog.tracef("(TBD) -- loading data from library data directory: %q", libData)
	}

	// runtime environment defined, begin preparing the libs and databases.
	infoLog.log("initializing library databases ...")

	// remaining arguments are considered paths to libraries; verify the paths
	// before assuming valid ones exist for traversal.
	library := initLibrary(options)
	if 0 == len(library) {
		errLog.die(rcInvalidConfig.spec("no valid libraries provided"), false)
	}
	infoLog.log("initialization complete")
	infoLog.log("scanning libraries ...")
	scanStart := time.Now()

	// libraries ready, spool up the library scanners.
	populateLibrary(options, library)

	for _, l := range library {
		<-l.scanComplete
	}
	scanElapsed := time.Since(scanStart)
	infoLog.logf("scan complete (%s)", scanElapsed.Round(time.Millisecond))

	infoLog.die(rcOK.spec("have a nice day! or night!"), false)
}

// function initOptions() parses all command line arguments and prepares the
// environment.
func initOptions() (options *Options, err *ReturnCode) {

	// panic handler
	defer func() {
		if recovered := recover(); nil != recovered {
			options = nil
			if flag.ErrHelp == recovered {
				// hide the flag.flagSet's default output status message,
				// because we will print our own.
				err = rcUsage
				return
			}
			// at this point we encountered an actual error, capture it and show
			// it with our error logger. (NOTE: this "err" is a named output
			// paramater of function initOptions()).
			err = rcInvalidArgs.specf("%s", recovered)
		}
	}()

	configPath := filepath.Join(configDir(nil), defaultConfigPath)
	libDataPath := filepath.Join(configDir(nil), defaultLibDataPath)

	// define the option properties that the command line parser recognizes.
	options = &Options{
		// PanicOnError gets trapped by the anon defer'd func() above. the
		// recover()'d  value will be set to flag.ErrHelp, which we want to
		// override by printing with our error logger.
		FlagSet:  flag.NewFlagSet(identity, flag.PanicOnError),
		Provided: NamedOption{},
		UsageHelp: &Option{
			name:  "help",
			usage: "display this helpful usage synopsis!",
			bool:  false,
		},
		Verbose: &Option{
			name:  "verbose",
			usage: "display additional status information",
			bool:  false,
		},
		Trace: &Option{
			name:  "trace",
			usage: "display additional status information (maximum verbosity)",
			bool:  false,
		},
		Config: &Option{
			name:   "config",
			usage:  "path to config file",
			string: configPath,
		},
		LibData: &Option{
			name:   "libdata",
			usage:  "path to library data directory (database storage location)",
			string: libDataPath,
		},
		DBBufferSize: &Option{
			name:  "dbbuffersize",
			usage: "size (in bytes) of each library's preallocated on-disk buffers (number of buffers = number of CPU cores)\n  (NOTE: this may not be changed after the corresponding library's database has been created)",
			int:   defaultDBBufferSize,
		},
		DBHashSize: &Option{
			name:  "dbhashsize",
			usage: "size (in bytes) by which each hash table will grow to make room once it reaches capacity\n  (NOTE: this may not be changed after the corresponding library's database has been created)",
			int:   defaultDBHashGrowth,
		},
	}
	// DBBufferSize    *Option // size (bytes) of each collection's pre-allocated buffers on disk. num buffers = num CPU cores
	// DBHashSize      *Option // size (bytes) by which each hash table will grow once individual capacity is exceeded.
	knownOptions := NamedOption{
		"help":         options.UsageHelp,
		"verbose":      options.Verbose,
		"trace":        options.Trace,
		"config":       options.Config,
		"libdata":      options.LibData,
		"dbbuffersize": options.DBBufferSize,
		"dbhashsize":   options.DBHashSize,
	}

	// register the command line options we want to handle.
	options.BoolVar(&options.UsageHelp.bool, options.UsageHelp.name, options.UsageHelp.bool, options.UsageHelp.usage)
	options.BoolVar(&options.Verbose.bool, options.Verbose.name, options.Verbose.bool, options.Verbose.usage)
	options.BoolVar(&options.Trace.bool, options.Trace.name, options.Trace.bool, options.Trace.usage)
	options.StringVar(&options.Config.string, options.Config.name, options.Config.string, options.Config.usage)
	options.StringVar(&options.LibData.string, options.LibData.name, options.LibData.string, options.LibData.usage)
	options.IntVar(&options.DBBufferSize.int, options.DBBufferSize.name, options.DBBufferSize.int, options.DBBufferSize.usage)
	options.IntVar(&options.DBHashSize.int, options.DBHashSize.name, options.DBHashSize.int, options.DBHashSize.usage)

	// hide the flag.flagSet's default output error message, because we will
	// display our own.
	options.SetOutput(ioutil.Discard)

	// the output provided with -help or when a option parse error occurred.
	options.Usage = func() {
		rawLog.logf("%s v%s (%s) [%s]", identity, version, revision, buildtime)
		rawLog.log()
		options.SetOutput(os.Stdout)
		options.PrintDefaults()
		rawLog.log()
	}

	// yeaaaaaaah, now we do it!
	options.Parse(os.Args[1:])
	options.Visit(
		func(f *flag.Flag) { options.Provided[f.Name] = knownOptions[f.Name] })

	var parseError *ReturnCode

	// update program state for global optons.
	if options.UsageHelp.bool {
		options.Usage()
		parseError = rcUsage
	}

	// update the loggers' verbosity settings.
	isVerboseLog = options.Verbose.bool
	isTraceLog = options.Trace.bool

	return options, parseError
}

// function initLibrary() validates all library paths provided, returning a list
// of the valid ones.
func initLibrary(options *Options) []*Library {

	var library []*Library

	// any remaining args were not handled by the options parser. they are then
	// considered to be file paths of libraries to scan.
	libArgs := options.Args()

	// dispatch a single goroutine per library to verify each concurrently.
	for _, libPath := range libArgs {
		lib, err := newLibrary(
			options, libPath, depthUnlimited, library)

		// if we encounter an error, issue a warning, do NOT add it to the list
		// of valid libraries, and continue. if it is truly a fatal error, then
		// all user-provided libraries will fail for the same reason; the list
		// of valid libraries will be empty on return, and the program will
		// terminate with error "no libraries found".
		if nil != err {
			warnLog.log(err)
		} else {
			// no error encountered, so the library is considered valid. add it
			// to the queue.
			infoLog.verbosef("using library: %s", lib)
			library = append(library, lib)
		}
	}

	return library
}

// function populateLibrary() spawns goroutines to scan each library
// concurrently. it also spawns goroutines that listen via channels for new
// media discovered (see function watchLibrary() for handlers).
func populateLibrary(options *Options, library []*Library) {

	// for each library, dispatch a few (3) goroutines in the following order:
	//   1. begin listening for content on the new-media and new-directory
	//       discovery channels, and decide what to do with them;
	//   2. dump all of the content from the library's database, verifying it
	//       and notifying the discovery channels;
	//   3. recursively traverse the library's filesystem, identifying which
	//       content is valid and desirable, then notify the discovery channels
	//       accordingly.
	for _, lib := range library {

		// 1. spool up the discovery channel polling, handling the content as
		//   it is received -- the media is considered valid if it has reached
		//   the channel.
		go watchLibrary(lib)

		// 2. pull all of the media already known to exist in the library from
		//  the local database, verify it still exists, then notify the channel.
		//go func(l *Library) {
		//
		//}(lib)

		// 3. recursively walks a library's file system, notifying the library's
		// signal channels whenever any sort of content is found.
		go func(l *Library) {
			err := l.scan(defaultScanHandler /* defined in library.go */)
			if nil != err {
				errLog.verbose(err)
			}
		}(lib)
	}

	// we don't wait for the scanning to finish. go ahead and launch the UI for
	// progress indicators and anything else the user can get away with while
	// they work.
}

// function watchLibrary() is the dispatched goroutine that listens for and
// handles new media as they are discovered. the media has been validated before
// it is written to the channel, so you can safely assume the media here exists
// and is desirable.
func watchLibrary(lib *Library) {
	// continuously monitors a library's signal channels for new content, which
	// creates or processes the content accordingly.
	for {
		select {
		case v := <-lib.newDirectory:
			switch v.data[0].(type) {
			case string:
				d := v.data[0].(string)
				infoLog.tracef("entered subdirectory: %s", d)
			default:
				warnLog.tracef("unknown object in directory channel: %t", v.data)
			}
		case v := <-lib.newMedia:
			switch v.data[0].(type) {
			case *AudioMedia:
				m := v.data[0].(*AudioMedia)
				infoLog.tracef("discovered media: %s", m)
			case *VideoMedia:
				m := v.data[0].(*VideoMedia)
				infoLog.tracef("discovered media: %s", m)
			default:
				warnLog.tracef("unknown object in media channel: %t", v.data)
			}
		}
	}
}
