// =============================================================================
//  PROJ: pimmp
//  AUTH: ardnew
//  DATE: 26 Sept 2018
//  FILE: main.go
// -----------------------------------------------------------------------------
//
//  DESCRIPTION
//    (TBD)
//
// =============================================================================

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
)

// versioning information defined by compiler switches in Makefile
var (
	version   string
	revision  string
	buildtime string
)

// type Option struct can contain any possible individual option configuration
// including its command line flag identifier and usage info
type Option struct {
	name  string
	usage string
	bool
	int
	uint
	float64
	string
}

// type Options struct defines the collection of all command line options
type Options struct {
	*flag.FlagSet
	UsageHelp Option
	Verbose   Option
}

// function main() is the program entry point, obviously
func main() {

	// parse options and command line arguments
	options, err := initOptions()
	if nil != err {
		errLog.Die(err, false)
	}

	// remaining arguments are considered paths to libraries; verify the paths
	// before assuming valid ones exist for traversal
	library := initLibrary(options)
	if 0 == len(library) {
		errLog.Die(rcInvalidLibrary.withInfo("no libraries found"), false)
	}

	// libraries ready, spool up the initial library scanners
	populateLibrary(options, library)

	infoLog.Log("ready")
	for {
	}

	infoLog.Die(rcOK.withInfo("have a nice day!"), false)
}

// function initOptions() parses all command line arguments and prepares the
// environment
func initOptions() (options *Options, err *ReturnCode) {

	// panic handler
	defer func() {
		if recovered := recover(); nil != recovered {
			options = nil
			if flag.ErrHelp == recovered {
				// hide the flag.flagSet's default output status message,
				// because we will print our own
				err = rcUsage
				return
			}
			// at this point we encountered an actual error, capture it and show
			// it with our error logger
			info := fmt.Sprintf("%s", recovered)
			// note this "err" is a named output paramater
			err = rcInvalidArgs.withInfo(info)
		}
	}()

	// define the option properties that the command line parser shall recognize
	selfName := path.Base(os.Args[0]) // using application name by default
	options = &Options{
		// PanicOnError gets trapped by the anon defer'd func() above. the
		// recover()'d  value will be set to flag.ErrHelp, which we want to
		// override or print with our error logger
		FlagSet: flag.NewFlagSet(selfName, flag.PanicOnError),
		UsageHelp: Option{
			name:  "help",
			usage: "display this helpful usage synopsis!",
			bool:  false,
		},
		Verbose: Option{
			name:  "verbose",
			usage: "display more detailed output",
			bool:  false,
		},
	}

	// register the command line options we want to handle
	options.BoolVar(&options.UsageHelp.bool, options.UsageHelp.name, options.UsageHelp.bool, options.UsageHelp.usage)
	options.BoolVar(&options.Verbose.bool, options.Verbose.name, options.Verbose.bool, options.Verbose.usage)

	// hide the flag.flagSet's default output error message, because we will
	// display our own
	options.SetOutput(ioutil.Discard)

	// the output provided with -help or when a option parse error occurred
	options.Usage = func() {
		rawLog.Logf("%s v%s (%s) [%s]", selfName, version, revision, buildtime)
		rawLog.Log()
		options.SetOutput(os.Stdout)
		options.PrintDefaults()
		rawLog.Log()
	}

	// yeaaaaaaah, now we do it
	options.Parse(os.Args[1:])

	var parseError *ReturnCode

	// update program state for global optons
	if options.UsageHelp.bool {
		options.Usage()
		parseError = rcUsage
	}

	// update the loggers' verbosity setting
	isVerboseLog = options.Verbose.bool

	return options, parseError
}

// function initLibrary() verifies all library paths provided, returning a list
// of the valid ones
func initLibrary(options *Options) []*Library {

	var library []*Library

	// any remaining args were not handled by the options parser. they are then
	// considered to be file paths of libraries to scan
	libArgs := options.Args()

	// dispatch a single goroutine per library to verify each concurrently
	for _, libPath := range libArgs {
		lib, err := NewLibrary(libPath, depthUnlimited)
		if nil != err {
			errLog.Die(err, true)
		}
		library = append(library, lib)
	}

	return library
}

// function populateLibrary() spawns goroutines to scan each library
// concurrently. it also spawns goroutines that listen via channels for new
// media discovered (see function watchLibrary() for handlers).
func populateLibrary(options *Options, library []*Library) {

	// all libraries are considered valid at this point. dispatch a single
	// goroutine for each library that will listen on various channels for
	// content discovered by the concurrent scanners
	for _, lib := range library {
		go watchLibrary(lib)
	}

	// and now dispatch the concurrent scanners -- one per library. as media or
	// other types of content are discovered, they will be fed into the receiver
	// goroutines dispatched above which will decide what to do with them.
	for _, lib := range library {
		go func(l *Library) {
			// recursively walks a library path, notifying the library's signal channels
			// whenever any sort of content is found. the scan time and details are
			// logged to the info log
			err := l.Scan()
			if nil != err {
				errLog.VLog(err)
			}
		}(lib)
	}

	// we don't wait for the scanning to finish. go ahead and launch the UI for
	// progress indicators and anything else the user can get away with while
	// they work.
}

// function watchLibrary() is the dispatched goroutine that listens for and
// handles new media as they are discovered
func watchLibrary(lib *Library) {
	// continuously monitors a library's signal channels for new content, which
	// creates or processes the content accordingly
	for {
		select {
		// library scanner discovered a subdirectory
		//case subdir := <-lib.newDirectory:
		//	infoLog.VLogf("entering: %q", subdir)
		case <-lib.newDirectory:

		// library scanner discovered media
		case media := <-lib.newMedia:
			infoLog.VLogf("processed: %q", media)
		}
	}
}
