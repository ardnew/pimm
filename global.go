package main

import (
	"fmt"
	"log"
	"os"
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
	EDirDepth       = &ExitCode{3, "exceeded directory depth limit"}
	EFileStat       = &ExitCode{4, "cannot stat file"}
	EInvalidFile    = &ExitCode{5, "invalid file"}
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
	*log.Logger
}

const (
	logFlags = log.Ldate | log.Ltime
)

var (
	InfoLog ConsoleLog
	WarnLog ConsoleLog
	ErrLog  ConsoleLog
)

func init() {
	InfoLog = ConsoleLog{Logger: log.New(os.Stdout, "[ ] ", logFlags)}
	WarnLog = ConsoleLog{Logger: log.New(os.Stderr, "[*] ", logFlags)}
	ErrLog = ConsoleLog{Logger: log.New(os.Stderr, "[!] ", logFlags)}
}

func (l *ConsoleLog) Log(v ...interface{}) {
	s := fmt.Sprint(v...)
	l.Printf("| %s", s)
}

func (l *ConsoleLog) Logf(format string, v ...interface{}) {
	s := fmt.Sprintf(format, v...)
	l.Printf("| %s", s)
}

func (l *ConsoleLog) Logln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	l.Printf("| %s", s)
}

func (l *ConsoleLog) Die(c *ErrorCode) {
	l.Printf("| %s", error(c))
	os.Exit(c.Code)
}
