package main

import (
	"flag"
	"sync"
)

// prepare runtime, args, and parse command linegodoc
func initLibrary() []*Library {

	var (
		wg      sync.WaitGroup
		library []*Library
	)

	// TODO: define command line options
	flag.Parse()

	fa := flag.Args()
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
	var (
		wg sync.WaitGroup
	)

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
