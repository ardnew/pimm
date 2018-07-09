package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"
)

type ExitCode struct {
	Code int
	Desc string
}

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

type ConsoleLog struct {
	isUTF8 bool
	prefix map[bool]string
	writer io.Writer
	update *chan bool
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

type UTF8OptionalString map[bool]string // false=ASCII, true=UTF8

var MoonPhase = [8]rune{'🌑', '🌒', '🌓', '🌔', '🌕', '🌖', '🌗', '🌘'}

var (
	treeNodePrefixExpanded = map[bool]UTF8OptionalString{
		false: {false: "-", true: "+"},
		true:  {false: "▶ ", true: "▼ "},
	}
	consoleLogPrefix = [consoleLogCount]UTF8OptionalString{
		{false: "", true: ""},
		{false: " = ", true: " [green]»[white] "},
		{false: " * ", true: " [yellow]»[white] "},
		{false: " ! ", true: " [red]×[white] "},
	}
)

var consoleLog = [consoleLogCount]*ConsoleLog{
	// RawLog:
	&ConsoleLog{
		isUTF8: false,
		prefix: consoleLogPrefix[rawLogID],
		writer: os.Stdout,
		update: nil,
		Logger: log.New(os.Stdout, consoleLogPrefix[rawLogID][false], 0),
	},
	// InfoLog:
	&ConsoleLog{
		isUTF8: false,
		prefix: consoleLogPrefix[infoLogID],
		writer: os.Stdout,
		update: nil,
		Logger: log.New(os.Stdout, consoleLogPrefix[infoLogID][false], logFlags),
	},
	// WarnLog:
	&ConsoleLog{
		isUTF8: false,
		prefix: consoleLogPrefix[warnLogID],
		writer: os.Stderr,
		update: nil,
		Logger: log.New(os.Stderr, consoleLogPrefix[warnLogID][false], logFlags),
	},
	// ErrLog:
	&ConsoleLog{
		isUTF8: false,
		prefix: consoleLogPrefix[errLogID],
		writer: os.Stderr,
		update: nil,
		Logger: log.New(os.Stderr, consoleLogPrefix[errLogID][false], logFlags),
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
		l.Logger = log.New(w, l.Prefix(), l.Flags())
		l.writer = w
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
		l.Logger = log.New(l.writer, l.prefix[c], l.Flags())
		l.isUTF8 = c
		l.Unlock()
	}
}

func setLogUnicode(c bool) {
	for _, l := range consoleLog {
		l.SetUnicode(c)
	}
}

func (l *ConsoleLog) setUpdate(u *chan bool) {
	if l.update != u {
		l.Lock()
		l.update = u
		l.Unlock()
	}
}

func setLogUpdate(u *chan bool) {
	for _, l := range consoleLog {
		l.setUpdate(u)
	}
}

func (l *ConsoleLog) Raw(s string) {
	if true {
		l.Print(s)
		// the following is causing serious input lag
		//if nil != l.update {
		//	*l.update <- true
		//}
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
