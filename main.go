package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
)

var (
	OptionColorLog bool
)

func initOptions() *flag.FlagSet {

	defer func() {
		if r := recover(); nil != r {
			if flag.ErrHelp == r {
				RawLog.Die(NewErrorCode(EUsage))
			}
			RawLog.Die(NewErrorCode(EArgs, fmt.Sprintf("%s", r)))
		}
	}()

	optionID := os.Args[0] // using application name by default
	option := flag.NewFlagSet(optionID, flag.PanicOnError)

	option.BoolVar(&OptionColorLog, "log-color", true, "colorize messages in output log")

	option.SetOutput(ioutil.Discard)
	option.Usage = func() {
		option.SetOutput(os.Stderr)
		RawLog.Logf("%s version ?", optionID)
		RawLog.Log()
		option.PrintDefaults()
		RawLog.Log()
	}

	option.Parse(os.Args[1:])

	return option
}

// prepare runtime, args, and parse command line
func initLibrary() []*Library {

	var (
		wg      sync.WaitGroup
		library []*Library
	)

	CmdLine := initOptions()
	fa := CmdLine.Args()
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
	library := initLibrary()
	if 0 == len(library) {
		ErrLog.Die(NewErrorCode(EInvalidLibrary, "no libraries found"))
	}
	populateLibrary(library)
	InfoLog.Log("libraries ready")

	InfoLog.Die(NewErrorCode(EOK, "have a nice day!"))
}
