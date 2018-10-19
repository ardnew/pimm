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
	LibData   Option // defines data directory path (where to store databases)
}

// function configDir() constructs the full path to the default directory
// containing all of the program's supporting persistent data.
func configDir() string {
	return filepath.Join(homeDir(), fmt.Sprintf(".%s", identity))
}

// function main() is the program entry point, obviously.
func main() {

	// parse options and command line arguments.
	options, err := initOptions()
	if nil != err {
		errLog.die(err, false)
	}

	// configuration defined, begin preparing the runtime environment.
	infoLog.log("initializing library databases ...")

	// create the directory hierarchy that will store our configuration data
	// permanently on disk.
	config := configDir()
	if exists, _ := goutil.PathExists(config); !exists {
		if err := os.MkdirAll(config, os.ModePerm); nil != err {
			errLog.die(rcInvalidConfig.withInfof(
				"cannot create configuration directory: %q: %s", config, err), false)
		}
		infoLog.tracef("created configuration directory: %q", config)
	}

	// create the directory hierarchy that will store our libraries' backing
	// data stores permanently on disk.
	dbRoot := options.LibData.string
	if exists, _ := goutil.PathExists(dbRoot); !exists {
		if err := os.MkdirAll(dbRoot, os.ModePerm); nil != err {
			errLog.die(rcInvalidLibrary.withInfof(
				"cannot create data directory: %q: %s", dbRoot, err), false)
		}
		infoLog.tracef("created data directory: %q", dbRoot)
	}

	// remaining arguments are considered paths to libraries; verify the paths
	// before assuming valid ones exist for traversal.
	library := initLibrary(options)
	if 0 == len(library) {
		errLog.die(rcInvalidLibrary.withInfo("no libraries found"), false)
	}

	// libraries ready, spool up the library scanners.
	populateLibrary(options, library)

	infoLog.log("initialization complete")

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

	dataDir := filepath.Join(configDir(), defaultDataDir)

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
			usage: "display very detailed status information",
			bool:  false,
		},
		LibData: Option{
			name:   "libdata",
			usage:  "path used to store library databases",
			string: dataDir,
		},
	}

	// register the command line options we want to handle.
	options.BoolVar(&options.UsageHelp.bool, options.UsageHelp.name, options.UsageHelp.bool, options.UsageHelp.usage)
	options.BoolVar(&options.Verbose.bool, options.Verbose.name, options.Verbose.bool, options.Verbose.usage)
	options.BoolVar(&options.Trace.bool, options.Trace.name, options.Trace.bool, options.Trace.usage)
	options.StringVar(&options.LibData.string, options.LibData.name, options.LibData.string, options.LibData.usage)

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
			libPath, options.LibData.string, depthUnlimited, library)

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
		dirEnter: func(l *Library, p string, v ...interface{}) {
			infoLog.tracef("entering: %#s", p)
			l.newDirectory <- newDiscovery(p, nil)
		},

		// the scanner exited a subdirectory of the library's file system.
		dirExit: func(l *Library, p string, v ...interface{}) {
			infoLog.tracef("exiting: %#s", p)
		},

		// the scanner identified some file in a subdirectory of the library's
		// file system as a media file.
		fileMedia: func(l *Library, p string, v ...interface{}) {
			m := v[0].(*Media)
			infoLog.tracef("discovered: %#s", m)
			l.newMedia <- newDiscovery(m, nil)
		},

		// the scanner identified some file in a subdirectory of the library's
		// file system as a supporting auxiliary file to a known or as-of-yet
		// unknown media file.
		fileMediaAux: func(l *Library, p string, v ...interface{}) {
		},

		// the scanner identified some file in a subdirectory of the library's
		// file system as an undesirable piece of trash.
		fileOther: func(l *Library, p string, v ...interface{}) {
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
		// library scanner entered a subdirectory.
		//		case disco := <-lib.newDirectory:
		//			subdir := disco.string

		// library scanner discovered media.
		//		case disco := <-lib.newMedia:
		//			media := *disco.Media

		case <-lib.newDirectory:
		case <-lib.newMedia:
		}
	}
}
