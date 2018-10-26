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

// type Options struct defines the collection of all command line options.
type Options struct {
	*flag.FlagSet
	UsageHelp Option // shows usage synopsis
	Verbose   Option // prints additional status information
	Trace     Option // prints very detailed status information
	Config    Option // defines path to config file
	LibData   Option // defines data directory path (where to store databases)

	DBMaxRecordSize Option // TBD
	DBBufferSize    Option // TBD
	DBHashSize      Option // TBD
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
				errLog.die(rcInvalidConfig.withInfof(
					"cannot create configuration directory: %q: %s", configDir, err), false)
			}
			warnLog.tracef("created configuration directory: %q", configDir)
		}

		// TODO: create configuration file
		warnLog.tracef("(TBD) -- created configuration: %q", config)
	}

	// if we haven't died yet, then config dir/file exists. load it.
	infoLog.tracef("(TBD) -- loading configuration: %q", config)

	// create the directory hierarchy that will store our libraries' backing
	// data stores permanently on disk.
	libData := options.LibData.string
	if exists, _ := goutil.PathExists(libData); !exists {
		if err := os.MkdirAll(libData, os.ModePerm); nil != err {
			errLog.die(rcInvalidLibrary.withInfof(
				"cannot create library data directory: %q: %s", libData, err), false)
		}
		warnLog.tracef("created library data directory: %q", libData)
	} else {
		infoLog.tracef("(TBD) -- loading data from library data directory: %q", libData)
	}

	// runtime environment defined, begin preparing the libs and databases.
	infoLog.log("initializing library databases ...")

	// remaining arguments are considered paths to libraries; verify the paths
	// before assuming valid ones exist for traversal.
	library := initLibrary(options)
	if 0 == len(library) {
		errLog.die(rcInvalidLibrary.withInfo("no libraries found"), false)
	}
	infoLog.log("initialization complete")

	// libraries ready, spool up the library scanners.
	populateLibrary(options, library)

	// keep process up and running
	for {
	}

	infoLog.die(rcOK.withInfo("have a nice day!"), false)
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
			err = rcInvalidArgs.withInfof("%s", recovered)
		}
	}()

	configPath := filepath.Join(configDir(nil), defaultConfigPath)
	libDataPath := filepath.Join(configDir(nil), defaultLibDataPath)

	// define the option properties that the command line parser recognizes.
	options = &Options{
		// PanicOnError gets trapped by the anon defer'd func() above. the
		// recover()'d  value will be set to flag.ErrHelp, which we want to
		// override by printing with our error logger.
		FlagSet: flag.NewFlagSet(identity, flag.PanicOnError),
		UsageHelp: Option{
			name:  "help",
			usage: "display this helpful usage synopsis!",
			bool:  false,
		},
		Verbose: Option{
			name:  "verbose",
			usage: "display additional status information",
			bool:  false,
		},
		Trace: Option{
			name:  "trace",
			usage: "display additional status information (maximum verbosity)",
			bool:  false,
		},
		Config: Option{
			name:   "config",
			usage:  "path to config file",
			string: configPath,
		},
		LibData: Option{
			name:   "libdata",
			usage:  "path to library data directory (database storage location)",
			string: libDataPath,
		},
		DBMaxRecordSize: Option{
			name:  "dbmaxrecordsize",
			usage: "TBD",
			int:   defaultDBRecMaxSize,
		},
		DBBufferSize: Option{
			name:  "dbbuffersize",
			usage: "TBD",
			int:   defaultDBBufferSize,
		},
		DBHashSize: Option{
			name:  "dbhashsize",
			usage: "TBD",
			int:   defaultDBHashGrowth,
		},
	}

	// register the command line options we want to handle.
	options.BoolVar(&options.UsageHelp.bool, options.UsageHelp.name, options.UsageHelp.bool, options.UsageHelp.usage)
	options.BoolVar(&options.Verbose.bool, options.Verbose.name, options.Verbose.bool, options.Verbose.usage)
	options.BoolVar(&options.Trace.bool, options.Trace.name, options.Trace.bool, options.Trace.usage)
	options.StringVar(&options.Config.string, options.Config.name, options.Config.string, options.Config.usage)
	options.StringVar(&options.LibData.string, options.LibData.name, options.LibData.string, options.LibData.usage)
	options.IntVar(&options.DBMaxRecordSize.int, options.DBMaxRecordSize.name, options.DBMaxRecordSize.int, options.DBMaxRecordSize.usage)
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

		// if we encounter an error, check the return code to determine if it is
		// fatal or not. special case logic is added for each non-fatal codes.
		// if no case logic exists, then it is assumed fatal by default.
		if nil != err {
			switch err.code {
			case rcDuplicateLibrary.code:
				warnLog.log(err)
			default:
				errLog.die(err, false)
			}
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

	scanHandler := &ScanHandler{

		// the scanner entered a subdirectory of the library's file system.
		handleEnter: func(l *Library, p string, v ...interface{}) {
			infoLog.tracef("entering: %#s", p)
			l.newDirectory <- newDiscovery(p, nil)
		},

		// the scanner exited a subdirectory of the library's file system.
		handleExit: func(l *Library, p string, v ...interface{}) {
			infoLog.tracef("exiting: %#s", p)
		},

		// the scanner identified some file in a subdirectory of the library's
		// file system as a media file.
		handleMedia: func(l *Library, p string, v ...interface{}) {
			m := v[0].(*Media)
			infoLog.tracef("discovered: %#s", m)
			l.newMedia <- newDiscovery(m, nil)
		},

		// the scanner identified some file in a subdirectory of the library's
		// file system as a supporting auxiliary file to a known or as-of-yet
		// unknown media file.
		handleAux: func(l *Library, p string, v ...interface{}) {
		},

		// the scanner identified some file in a subdirectory of the library's
		// file system as an undesirable piece of trash.
		handleOther: func(l *Library, p string, v ...interface{}) {
		},
	}

	// dispatch a single goroutine for each library that will listen on various
	// channels for content discovered by the concurrent scanners.
	for _, lib := range library {
		go watchLibrary(lib)
	}

	// and now dispatch the concurrent scanners, two per library: one reading
	// from the local database and one reading from the file system. as media or
	// other types of content are discovered, they will be fed into the receiver
	// goroutines dispatched above which will decide what to do with them.
	for _, lib := range library {

		// pulls all of the media already known to exist in the library from the
		// local database.
		go func(l *Library) {
			//err := l.Scan()
			//if nil != err {
			//	errLog.vlog(err)
			//}
		}(lib)

		// recursively walks a library's file system, notifying the library's
		// signal channels whenever any sort of content is found.
		go func(l *Library) {
			err := l.scan(scanHandler)
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
// handles new media as they are discovered.
func watchLibrary(lib *Library) {
	// continuously monitors a library's signal channels for new content, which
	// creates or processes the content accordingly.
	for {
		select {
		case <-lib.newDirectory:
		case <-lib.newMedia:
		}
	}
}
