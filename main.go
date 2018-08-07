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
	localeLang := os.Getenv(LocaleEnvVar)
	isUTF8, err := regexp.MatchString("\\.UTF-8$", localeLang)
	if nil != err {
		// don't use UTF-8 if locale is screwy enough to break this regex, seriously
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

	libArgs := options.Args()
	libQueue := make(chan *Library, len(libArgs))

	for _, libPath := range libArgs {
		gateKeeper.Add(1)
		go func(libPath string) {
			defer gateKeeper.Done()
			lib, err := NewLibrary(libPath, make([]string, 0))
			if nil != err {
				errLog.Log(err.Reason)
				return
			}
			libQueue <- lib
		}(libPath)
	}
	gateKeeper.Wait()
	close(libQueue)

	// prep the library to start listening for media, adding them to the view
	for lib := range libQueue {
		library = append(library, lib)
	}
	return library
}

func populateLibrary(options *Options, library []*Library, ui *UI) {

	for _, lib := range library {
		ui.AddLibrary(lib)

		go func(lib *Library, ui *UI) {
			for {
				select {
				case subdir := <-lib.SigDir():
					ui.AddLibraryDirectory(lib, subdir)
				case media := <-lib.SigMedia():
					ui.AddMedia(lib, media)
				}
			}
		}(lib, ui)
	}

	for _, lib := range library {
		infoLog.Logf("initiating scan: %q", lib.Name())
		go func(lib *Library) {
			start := time.Now()
			err := lib.Scan()
			delta := time.Since(start)
			if nil != err {
				errLog.Log(err)
			} else {
				infoLog.Logf("finished scan: %q (%s) %d files, %s",
					lib.Name(), delta.Round(time.Millisecond), lib.TotalMedia(), SizeStr(lib.TotalSize(), false))
			}
		}(lib)
	}
}

func main() {

	// parse options and command line arguments
	options, ok := initOptions()
	if nil != ok {
		switch ok.(type) {
		case *ErrorCode:
			errLog.Die(ok.(*ErrorCode))
		default:
			errLog.Die(NewErrorCode(EArgs, fmt.Sprintf("%s", ok)))
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
