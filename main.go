package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"sync"
)

// globals initialized in Makefile
var (
	VERSION   string
	REVISION  string
	BUILDTIME string
)

type Option struct {
	name  string
	usage string
	bool
	int
	uint
	float64
	string
}

type Options struct {
	*flag.FlagSet
	ColorLog Option
}

func initOptions() *Options {

	defer func() {
		if r := recover(); nil != r {
			if flag.ErrHelp == r {
				RawLog.Die(NewErrorCode(EUsage))
			}
			RawLog.Die(NewErrorCode(EArgs, fmt.Sprintf("%s", r)))
		}
	}()

	invokedName := path.Base(os.Args[0]) // using application name by default
	options := &Options{
		FlagSet: flag.NewFlagSet(invokedName, flag.PanicOnError),
		ColorLog: Option{
			name:  "log-color",
			usage: "colorize messages in output log",
			bool:  true},
	}

	options.BoolVar(&options.ColorLog.bool, options.ColorLog.name, options.ColorLog.bool, options.ColorLog.usage)

	options.SetOutput(ioutil.Discard)
	options.Usage = func() {
		options.SetOutput(os.Stderr)
		RawLog.Logf("%s version %s-r%s (%s)", invokedName, VERSION, REVISION, BUILDTIME)
		RawLog.Log()
		options.PrintDefaults()
		RawLog.Log()
	}

	options.Parse(os.Args[1:])

	return options
}

func initLibrary(options *Options) []*Library {

	var library []*Library
	var wg sync.WaitGroup

	fa := options.Args()
	lq := make(chan *Library, len(fa))

	for _, p := range fa {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			l, err := NewLibrary(p)
			if nil != err {
				ErrLog.Log(err.Reason)
				return
			}
			lq <- l
		}(p)
	}
	wg.Wait()
	close(lq)

	for l := range lq {
		library = append(library, l)
	}
	return library
}

func populateLibrary(library []*Library) {

	var wg sync.WaitGroup

	for _, l := range library {
		wg.Add(1)
		InfoLog.Logf("Scanning library: %s", l)
		go func(l *Library) {
			defer wg.Done()
			err := l.Scan()
			if nil != err {
				ErrLog.Log(err)
			}
		}(l)
	}
	wg.Wait()
}

func main() {

	options := initOptions()
	library := initLibrary(options)
	if 0 == len(library) {
		ErrLog.Die(NewErrorCode(EInvalidLibrary, "no libraries found"))
	}
	populateLibrary(library)
	InfoLog.Log("libraries ready")

	InfoLog.Die(NewErrorCode(EOK, "have a nice day!"))
}
