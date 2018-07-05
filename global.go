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
	*log.Logger
	sync.Mutex
}

const (
	logFlags     = log.Ldate | log.Ltime
	logSeparator = '|'
)

const (
	rawLogID = iota
	infoLogID
	warnLogID
	errLogID
	consoleLogCount
)

var consoleLogPrefix = [consoleLogCount]map[bool]string{
	map[bool]string{false: "", true: ""},
	map[bool]string{false: " = ", true: " ✔ "},
	map[bool]string{false: " * ", true: " ⛔ "},
	map[bool]string{false: " ! ", true: " ✖ "},
}

var consoleLog = [consoleLogCount]*ConsoleLog{
	// RawLog:
	&ConsoleLog{
		isUTF8: false,
		prefix: consoleLogPrefix[rawLogID],
		writer: os.Stdout,
		Logger: log.New(os.Stdout, consoleLogPrefix[rawLogID][false], 0),
	},
	// InfoLog:
	&ConsoleLog{
		isUTF8: false,
		prefix: consoleLogPrefix[infoLogID],
		writer: os.Stdout,
		Logger: log.New(os.Stdout, consoleLogPrefix[infoLogID][false], logFlags),
	},
	// WarnLog:
	&ConsoleLog{
		isUTF8: false,
		prefix: consoleLogPrefix[warnLogID],
		writer: os.Stderr,
		Logger: log.New(os.Stderr, consoleLogPrefix[warnLogID][false], logFlags),
	},
	// ErrLog:
	&ConsoleLog{
		isUTF8: false,
		prefix: consoleLogPrefix[errLogID],
		writer: os.Stderr,
		Logger: log.New(os.Stderr, consoleLogPrefix[errLogID][false], logFlags),
	},
}

var (
	RawLog  *ConsoleLog = consoleLog[rawLogID]
	InfoLog *ConsoleLog = consoleLog[infoLogID]
	WarnLog *ConsoleLog = consoleLog[warnLogID]
	ErrLog  *ConsoleLog = consoleLog[errLogID]
)

func (l *ConsoleLog) SetUnicode(c bool) {
	if l.isUTF8 != c {
		l.Lock()
		l.Logger = log.New(l.writer, l.prefix[c], l.Flags())
		l.isUTF8 = c
		l.Unlock()
	}
}

func SetUnicodeLog(c bool) {
	for _, l := range consoleLog {
		l.SetUnicode(c)
	}
}

func (l *ConsoleLog) output(s string) {
	const (
		DO_OUTPUT = true
	)

	if DO_OUTPUT {
		if 0 != l.Logger.Flags() {
			//l.Printf("%c %s\n", logSeparator, s)
		} else {
			//l.Print(s)
		}
	}
}

func (l *ConsoleLog) Log(v ...interface{}) {
	s := fmt.Sprint(v...)
	l.output(s)
}

func (l *ConsoleLog) Logf(format string, v ...interface{}) {
	s := fmt.Sprintf(format, v...)
	l.output(s)
}

func (l *ConsoleLog) Logln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	l.output(s)
}

func (l *ConsoleLog) Die(c *ErrorCode) {
	if EUsage != c.ExitCode {
		s := fmt.Sprintf("%s", error(c))
		l.output(s)
	}
	os.Exit(c.Code)
}
