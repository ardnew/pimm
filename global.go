package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"
)

// type indicating the general reason for application termination
type ExitCode struct {
	Code int
	Desc string
}

// type containing an explicit reason/message for using the contained ExitCode
type ErrorCode struct {
	Reason string
	*ExitCode
}

var (
	EOK             = &ExitCode{0, "ok"}
	EInvalidLibrary = &ExitCode{1, "invalid library"}
	EDirOpen        = &ExitCode{2, "cannot read directory"}
	EDirDepth       = &ExitCode{3, "directory depth limited"}
	EFileStat       = &ExitCode{4, "cannot stat file"}
	EInvalidFile    = &ExitCode{5, "invalid file"}
	EArgs           = &ExitCode{6, "invalid arguments"}
	EFileIgnore     = &ExitCode{7, "user-ignored file"}
	EInvalidOption  = &ExitCode{8, "invalid user option"}
	EFileHash       = &ExitCode{9, "failed to calculate hash"}
	ELibraryBusy    = &ExitCode{10, "library busy"}
	EUsage          = &ExitCode{99, "usage"}
	EUnknown        = &ExitCode{255, "unknown error"}
)

func (c *ExitCode) Error() string {
	return fmt.Sprintf("%s(%d)", c.Desc, c.Code)
}

func (c *ErrorCode) Error() string {
	return fmt.Sprintf("%s: %s", error(c.ExitCode), c.Reason)
}

func NewErrorCode(c *ExitCode, v ...interface{}) *ErrorCode {
	s := fmt.Sprint(v...)
	return &ErrorCode{s, c}
}

// outer map: false=ASCII, true=UTF8
// inner map: false=Plain, true=Colored
type StringContext map[bool]map[bool]string

type ConsoleLog struct {
	isUTF8  bool
	isColor bool
	prefix  StringContext
	writer  io.Writer
	*log.Logger
	sync.Mutex
}

const (
	logFlags     = log.Ldate | log.Ltime
	logSeparator = "| "
)

const (
	rawLogID = iota
	infoLogID
	warnLogID
	errLogID
	consoleLogCount
)

var MoonPhase = [8]rune{'🌑', '🌒', '🌓', '🌔', '🌕', '🌖', '🌗', '🌘'}

var (
	treeNodePrefixExpanded = map[bool]StringContext{
		// collapsed
		false: { // plain        colored
			false: {false: "+ ", true: "+ "}, // ASCII
			true:  {false: "▶ ", true: "▶ "}, // UTF-8
		},
		// expanded
		true: { //  plain        colored
			false: {false: "- ", true: "- "}, // ASCII
			true:  {false: "▼ ", true: "▼ "}, // UTF-8
		},
	}
	treeNodePrefixIncluded = map[bool]StringContext{
		// not included
		false: { // plain            colored
			false: {false: "[gray]", true: "[gray]"}, // ASCII
			true:  {false: "[gray]", true: "[gray]"}, // UTF-8
		},
		// included
		true: { //  plain            colored
			false: {false: "[blue]", true: "[blue]"}, // ASCII
			true:  {false: "[blue]", true: "[blue]"}, // UTF-8
		},
	}
	consoleLogPrefix = [consoleLogCount]StringContext{
		{ // rawLogID
			false: {false: "", true: ""}, // ASCII
			true:  {false: "", true: ""}, // UTF-8
		},
		{ // infoLogID
			false: {false: " = ", true: " [green]=[white] "}, // ASCII
			true:  {false: " » ", true: " [green]»[white] "}, // UTF-8
		},
		{ // warnLogID
			false: {false: " * ", true: " [yellow]*[white] "}, // ASCII
			true:  {false: " » ", true: " [yellow]»[white] "}, // UTF-8
		},
		{ // errLogID
			false: {false: " ! ", true: " [red]![white] "}, // ASCII
			true:  {false: " × ", true: " [red]×[white] "}, // UTF-8
		},
	}
)

var consoleLog = [consoleLogCount]*ConsoleLog{
	// RawLog:
	&ConsoleLog{
		isUTF8:  false,
		isColor: false,
		prefix:  consoleLogPrefix[rawLogID],
		writer:  os.Stdout,
		Logger:  log.New(os.Stdout, consoleLogPrefix[rawLogID][false][false], 0),
	},
	// InfoLog:
	&ConsoleLog{
		isUTF8:  false,
		isColor: false,
		prefix:  consoleLogPrefix[infoLogID],
		writer:  os.Stdout,
		Logger:  log.New(os.Stdout, consoleLogPrefix[infoLogID][false][false], logFlags),
	},
	// WarnLog:
	&ConsoleLog{
		isUTF8:  false,
		isColor: false,
		prefix:  consoleLogPrefix[warnLogID],
		writer:  os.Stderr,
		Logger:  log.New(os.Stderr, consoleLogPrefix[warnLogID][false][false], logFlags),
	},
	// ErrLog:
	&ConsoleLog{
		isUTF8:  false,
		isColor: false,
		prefix:  consoleLogPrefix[errLogID],
		writer:  os.Stderr,
		Logger:  log.New(os.Stderr, consoleLogPrefix[errLogID][false][false], logFlags),
	},
}

var (
	rawLog  *ConsoleLog = consoleLog[rawLogID]
	infoLog *ConsoleLog = consoleLog[infoLogID]
	warnLog *ConsoleLog = consoleLog[warnLogID]
	errLog  *ConsoleLog = consoleLog[errLogID]
)

func (l *ConsoleLog) SetWriter(w io.Writer) {
	if l.writer != w {
		l.Lock()
		l.writer = w
		l.isColor = !(os.Stdout == w || os.Stderr == w)
		l.Logger = log.New(w, l.prefix[l.isUTF8][l.isColor], l.Flags())
		l.Unlock()
	}
}

func setLogWriter(w io.Writer) {
	for _, l := range consoleLog {
		l.SetWriter(w)
	}
}

func (l *ConsoleLog) SetUnicode(c bool) {
	if l.isUTF8 != c {
		l.Lock()
		l.isUTF8 = c
		l.Logger = log.New(l.writer, l.prefix[c][l.isColor], l.Flags())
		l.Unlock()
	}
}

func setLogUnicode(c bool) {
	for _, l := range consoleLog {
		l.SetUnicode(c)
	}
}

func (l *ConsoleLog) Raw(s string) {
	if true {
		l.Print(s)
	}
}

func (l *ConsoleLog) Output(s string) {
	if l != rawLog {
		l.Raw(fmt.Sprintf("%s%s", logSeparator, s))
	} else {
		l.Raw(s)
	}
}

func (l *ConsoleLog) Log(v ...interface{}) {
	s := fmt.Sprint(v...)
	l.Output(s)
}

func (l *ConsoleLog) Logf(format string, v ...interface{}) {
	s := fmt.Sprintf(format, v...)
	l.Output(s)
}

func (l *ConsoleLog) Logln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	l.Output(s)
}

func (l *ConsoleLog) Die(c *ErrorCode) {
	if EUsage != c.ExitCode {
		s := fmt.Sprintf("%s", error(c))
		l.Output(s)
	}
	os.Exit(c.Code)
}

func SizeStr(bytes int64, showBytes bool) string {

	kb := float32(bytes) / 1024.0
	mb := float32(kb) / 1024.0
	gb := float32(mb) / 1024.0
	tb := float32(gb) / 1024.0
	ss := ""

	switch {
	case uint64(tb) > 0:
		ss = fmt.Sprintf("%.3g TiB", tb)
	case uint64(gb) > 0:
		ss = fmt.Sprintf("%.3g GiB", gb)
	case uint64(mb) > 0:
		ss = fmt.Sprintf("%.3g MiB", mb)
	case uint64(kb) > 0:
		ss = fmt.Sprintf("%.3g KiB", kb)
	default:
		ss = fmt.Sprintf("%d B", bytes)
		showBytes = false
	}

	if showBytes {
		ss = fmt.Sprintf("%s (%d B)", ss, bytes)
	}
	return ss
}
