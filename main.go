package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"sync"
	"time"
)

// READ-ONLY (plz)! globals initialized in Makefile
var (
	VERSION   string
	REVISION  string
	BUILDTIME string
)

// struct to contain any possible individual option
type Option struct {
	name  string
	usage string
	bool
	int
	uint
	float64
	string
}

// struct defining the collection of all command line options
type Options struct {
	*flag.FlagSet
	UsageHelp Option
	UTF8      Option
}

func isLocaleUTF8() bool {

	const LocaleEnvVar = "LANG"

	// first try to determine the char encoding using the LANG environment var
	localeLang := os.Getenv(LocaleEnvVar)
	// ending with the string "[...]UTF-8" is good enough for me
	isUTF8, err := regexp.MatchString("\\.UTF-8$", localeLang)
	if nil != err {
		// don't use UTF-8 if locale is screwy enough, seriously
		warnLog.Logf("failed to parse env: %q: %s", LocaleEnvVar, err)
		return false
	}
	return isUTF8
}

func initOptions() (options *Options, err error) {

	// panic handler
	defer func() {
		if recovered := recover(); nil != recovered {
			options = nil
			if flag.ErrHelp == recovered {
				err = NewErrorCode(EUsage)
				return
			}
			// note this "err" is a named output paramater
			err = NewErrorCode(EArgs, fmt.Sprintf("%s", recovered))
		}
	}()

	isUTF8 := isLocaleUTF8()

	// define the option properties that the command line parser shall recognize
	invokedName := path.Base(os.Args[0]) // using application name by default
	options = &Options{
		FlagSet: flag.NewFlagSet(invokedName, flag.PanicOnError),
		UsageHelp: Option{
			name:  "help",
			usage: "display this helpful usage synopsis!",
			bool:  false,
		},
		UTF8: Option{
			name: "unicode",
			usage: "override Unicode (UTF-8) support. otherwise, this is determined\n" +
				"automatically by checking your locale for a \"UTF-8\" tag in the\n" +
				"LANG environment variable.",
			bool: isUTF8,
		},
	}

	// register the command line options we wan't to handle
	options.BoolVar(&options.UsageHelp.bool, options.UsageHelp.name, options.UsageHelp.bool, options.UsageHelp.usage)
	options.BoolVar(&options.UTF8.bool, options.UTF8.name, options.UTF8.bool, options.UTF8.usage)

	// the output provided with -help or when a option parse error occurred
	options.SetOutput(ioutil.Discard)
	options.Usage = func() {
		options.SetOutput(os.Stderr)
		rawLog.Logf("%s v%s (%s) [%s]", invokedName, VERSION, REVISION, BUILDTIME)
		rawLog.Log()
		options.PrintDefaults()
		rawLog.Log()
	}

	// yeaaaaaaah, now we do it
	options.Parse(os.Args[1:])

	return options, nil
}

func initLibrary(options *Options) []*Library {

	var library []*Library
	var gateKeeper sync.WaitGroup

	// any remaining args were not handled by the options parser. they are then
	// considered to be file paths of libraries to scan
	libArgs := options.Args()
	numLibs := len(libArgs)
	libQueue := make(chan *Library, numLibs)

	// dispatch a single goroutine per library to verify each concurrently
	gateKeeper.Add(numLibs)
	for _, libPath := range libArgs {
		go func(libPath string) {
			defer gateKeeper.Done()
			lib, err := NewLibrary(libPath, make([]string, 0))
			if nil != err {
				errLog.Log(err.Reason)
				return // exit the goroutine, skip posting to the chan queue
			}
			libQueue <- lib
		}(libPath)
	}

	// wait until all goroutines have verified the libraries paths before
	// continuing on with scanning their content
	gateKeeper.Wait()
	close(libQueue)

	// any libraries that were added to the queue are considered valid
	for lib := range libQueue {
		library = append(library, lib)
	}
	return library
}

func watchLibrary(lib *Library, ui *UI) {
	// continuously monitors a library's signal channels for new content, which
	// creates or processes the content accordingly
	for {
		select {
		// library scanner discovered a subdirectory
		case subdir := <-lib.SigDir():
			ui.AddLibraryDirectory(lib, subdir)
		// library scanner discovered media
		case media := <-lib.SigMedia():
			ui.AddMedia(lib, media)
		}
	}
}

func scanLibrary(lib *Library) {
	// recursively walks a library path, notifying the library's signal channels
	// whenever any sort of content is found. the scan time and details are
	// logged to the info log
	start := time.Now()
	err := lib.Scan()
	delta := time.Since(start)
	if nil != err {
		errLog.Log(err)
	} else {
		infoLog.Logf("finished scan: %q (%s) %d files, %s",
			lib.Name(), delta.Round(time.Millisecond), lib.TotalMedia(), SizeStr(lib.TotalSize(), false))
	}
}

func populateLibrary(options *Options, library []*Library, ui *UI) {

	// all libraries are considered valid at this point. dispatch a single
	// goroutine for each library that will listen on various channels for
	// content discovered by the concurrent scanners
	for _, lib := range library {
		ui.AddLibrary(lib)
		go watchLibrary(lib, ui)
	}

	// and now dispatch the concurrent scanners -- one per library. as media or
	// other types of content are discovered, they will be fed into the receiver
	// goroutines dispatched above which will decide what to do with them.
	for _, lib := range library {
		infoLog.Logf("initiating scan: %q", lib.Name())
		go scanLibrary(lib)
	}

	// we don't wait for the scanning to finish. go ahead and launch the UI for
	// progress indicators and anything else the user can get away with while
	// they work.
}

func main() {

	// parse options and command line arguments
	options, err := initOptions()
	if nil != err {
		switch err.(type) {
		case *ErrorCode:
			errLog.Die(err.(*ErrorCode))
		default:
			errLog.Die(NewErrorCode(EArgs, fmt.Sprintf("%s", err)))
		}
	}

	// update program state for global optons
	if options.UsageHelp.bool {
		options.Usage()
		errLog.Die(NewErrorCode(EUsage))
	}
	setLogUnicode(options.UTF8.bool)

	// remaining arguments are considered paths to libraries; verify the paths
	// before assuming valid ones exist for traversal
	library := initLibrary(options)
	if 0 == len(library) {
		errLog.Die(NewErrorCode(EInvalidLibrary, "no libraries found"))
	}

	// initialize the UI components before population
	ui := NewUI(options)

	// libraries and UI ready, spool up the initial library scanners
	populateLibrary(options, library, ui)

	if err := ui.app.Run(); err != nil {
		errLog.Die(NewErrorCode(EUnknown, err))
	}

	infoLog.Die(NewErrorCode(EOK, "have a nice day!"))
}
