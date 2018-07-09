package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"sync"
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

// singleton struct containing current values of all defined command line options
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

func populateLibrary(library []*Library, layout *Layout) {

	for _, lib := range library {
		layout.AddLibrary(lib)

		go func(lib *Library, layout *Layout) {
			for {
				select {
				case subdir := <-lib.Subdir():
					infoLog.Logf("entering: %q", subdir)
					layout.AddLibrarySubdir(lib, subdir)
				case media := <-lib.MediaChan():
					infoLog.Logf("discovered: %s", media)
					layout.AddMedia(lib, media)
				}
			}
		}(lib, layout)
	}

	for _, lib := range library {
		infoLog.Logf("scanning library: %s", lib)
		go func(lib *Library) {
			err := lib.Scan()
			if nil != err {
				errLog.Log(err)
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

	// prep the UI components for population
	layout := initUI(options)

	// provided libraries exist and are readable, begin scanning
	populateLibrary(library, layout)
	infoLog.Log("libraries ready")

	// launch the terminal UI runtime
	if err := layout.app.Run(); err != nil {
		infoLog.Die(NewErrorCode(EUnknown, err))
	}

	infoLog.Die(NewErrorCode(EOK, "have a nice day!"))
}
