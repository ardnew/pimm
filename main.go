package main

import (
	// non-standard
	"ardnew.com/util"

	// standard libs
	"flag"
	"os"
	"path"
	"path/filepath"
)

var (
	searchRoot []string
)

// prepare runtime, args, and parse command linegodoc
func init() {

	// TODO: define command line options
	flag.Parse()

	for _, arg := range flag.Args() {

		ok, nfo := util.PathExists(arg)
		if !ok || !nfo.IsDir() {
			ErrLog.Dief(1, "no such directory: %q", arg)
		}

		searchRoot = append(searchRoot, path.Clean(arg))
	}
}

func main() {

	walker := make(chan string, len(searchRoot))
	for _, p := range searchRoot {
		go func(p string) {
			InfoLog.Logf("searching: %q", p)
			walker <- p
			walk(p)
		}(p)
	}
	for i := range searchRoot {
		InfoLog.Logf("search paths remaining: %d", len(searchRoot)-i-1)
		<-walker
	}
}

func handleWalk(path string, info os.FileInfo, err error) error {
	if err != nil {
		InfoLog.Logf("error handling path: %q: %v", path, err)
		return err
	}
	if !info.IsDir() {
		InfoLog.Logf("    %q", path)
	}
	return nil
}

func walk(root string) {
	if err := filepath.Walk(root, handleWalk); err != nil {
		WarnLog.Logf("error walking path: %q: %v", root, err)
	}
}
