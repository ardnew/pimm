package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"sync"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
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
	UTF8Log   Option
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

	// define the option properties that the command line parser shall recognize
	invokedName := path.Base(os.Args[0]) // using application name by default
	options = &Options{
		FlagSet: flag.NewFlagSet(invokedName, flag.PanicOnError),
		UsageHelp: Option{
			name:  "help",
			usage: "displays this helpful usage synopsis",
			bool:  false,
		},
		UTF8Log: Option{
			name:  "log-unicode",
			usage: "enable Unicode (UTF-8) encodings in output log",
			bool:  true,
		},
	}

	// register the command line options we wan't to handle
	options.BoolVar(&options.UsageHelp.bool, options.UsageHelp.name, options.UsageHelp.bool, options.UsageHelp.usage)
	options.BoolVar(&options.UTF8Log.bool, options.UTF8Log.name, options.UTF8Log.bool, options.UTF8Log.usage)

	// the output provided with -help or when a option parse error occurred
	options.SetOutput(ioutil.Discard)
	options.Usage = func() {
		options.SetOutput(os.Stderr)
		RawLog.Logf("%s v%s (%s) [%s]", invokedName, VERSION, REVISION, BUILDTIME)
		RawLog.Log()
		options.PrintDefaults()
		RawLog.Log()
	}

	// yeaaaaaaah, now we do it
	options.Parse(os.Args[1:])

	return options, nil
}

func initLibrary(options *Options) []*Library {

	var library []*Library
	var thrWait sync.WaitGroup

	libArgs := options.Args()
	libQueue := make(chan *Library, len(libArgs))

	for _, libPath := range libArgs {
		thrWait.Add(1)
		go func(libPath string) {
			defer thrWait.Done()
			lib, err := NewLibrary(libPath, make([]string, 0))
			if nil != err {
				ErrLog.Log(err.Reason)
				return
			}
			libQueue <- lib
		}(libPath)
	}
	thrWait.Wait()
	close(libQueue)

	// prep the library to start listening for media, adding them to the view
	for lib := range libQueue {
		library = append(library, lib)
	}
	return library
}

func populateLibrary(library []*Library, tree *tview.TreeView) {

	for _, lib := range library {
		libNode := tview.NewTreeNode(lib.Name())
		libNode.SetColor(tcell.ColorGreen)
		libNode.SetSelectable(true)
		libNode.SetExpanded(false)
		tree.GetRoot().AddChild(libNode)

		go func(lib *Library, libNode *tview.TreeNode) {
			for {
				media := <-lib.Media()
				InfoLog.Logf("received media: %s", media)
				mediaNode := tview.NewTreeNode(media.Name()).SetSelectable(true)
				libNode.AddChild(mediaNode)
			}
		}(lib, libNode)
	}

	for _, lib := range library {
		InfoLog.Logf("Scanning library: %s", lib)
		go func(lib *Library) {
			err := lib.Scan()
			if nil != err {
				ErrLog.Log(err)
			}
		}(lib)
	}
}

func main() {

	// parse options and command line arguments
	options, ok := initOptions()
	switch ok.(type) {
	case *ErrorCode:
		ErrLog.Die(ok.(*ErrorCode))
	default:
		ErrLog.Die(NewErrorCode(EArgs, fmt.Sprintf("%s", ok)))
	case nil:
	}

	// update program state for global optons
	if options.UsageHelp.bool {
		options.Usage()
		ErrLog.Die(NewErrorCode(EUsage))
	}
	SetUnicodeLog(options.UTF8Log.bool)

	// prep the treeview for displaying our media
	root := tview.NewTreeNode("Library").SetColor(tcell.ColorRed).SetSelectable(false)
	tree := tview.NewTreeView().SetRoot(root).SetCurrentNode(root)
	tree.SetSelectedFunc(
		func(node *tview.TreeNode) {
			if c := node.GetChildren(); len(c) > 0 {
				node.SetExpanded(!node.IsExpanded())
			}
		})

	// remaining arguments are considered paths to libraries; verify the paths
	// before assuming valid ones exist for traversal
	library := initLibrary(options)
	if 0 == len(library) {
		ErrLog.Die(NewErrorCode(EInvalidLibrary, "no libraries found"))
	}

	// provided libraries exist and are readable, begin scanning
	populateLibrary(library, tree)
	InfoLog.Log("libraries ready")

	// launch the terminal UI runtime
	if err := tview.NewApplication().SetRoot(tree, true).Run(); err != nil {
		InfoLog.Die(NewErrorCode(EUnknown, err))
	}

	InfoLog.Die(NewErrorCode(EOK, "have a nice day!"))
}
