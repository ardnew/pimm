// =============================================================================
//  PROJ: pimmp
//  AUTH: ardnew
//  DATE: 02 Oct 2018
//  FILE: log.go
// -----------------------------------------------------------------------------
//
//  DESCRIPTION
//    provides a collection of types and functions for logging data to a file
//    or to an output stream such as STDOUT and STDERR.
//
// =============================================================================

package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"runtime/debug"
	"sync"
)

// type ConsoleLog represents an object that logs data to one of the output
// streams of the user's console. the different loggers use different streams
// and various prefixes to distinguish between benign and fatal messages.
type ConsoleLog struct {
	prefix string
	writer io.Writer
	*log.Logger
	sync.Mutex
}

// unexported constants
const (
	logFlags     = log.Ldate | log.Ltime // flags defining format of log.Logger
	logDelimiter = "| "                  // log detail fields delimiter
)

// type LogID is an enum identifying the different kinds of built-in loggers.
type LogID int

// unexported constant values of the LogID enum type.
const (
	liRaw LogID = iota
	liInfo
	liWarn
	liError
	liCOUNT
)

// Madmen toil surreptitiously in rituals to beckon the moon. Uncover their secrets.
var MoonPhase = [8]rune{'🌑', '🌒', '🌓', '🌔', '🌕', '🌖', '🌗', '🌘'}

// var consoleLogPrefix defines the substring prefixes included in log messages
// to help visually grep for anything you might find significant.
var (
	consoleLogPrefix = [liCOUNT]string{
		"",    // liRaw
		"   ", // liInfo
		" » ", // liWarn
		" × ", // liError
	}
)

// var consoleLog defines each of our loggers.
var consoleLog = [liCOUNT]*ConsoleLog{
	// rawLog:
	newConsoleLog(
		consoleLogPrefix[liRaw],
		os.Stdout,
		log.New(os.Stdout, consoleLogPrefix[liRaw], 0)),
	// infoLog:
	newConsoleLog(
		consoleLogPrefix[liInfo],
		os.Stdout,
		log.New(os.Stdout, consoleLogPrefix[liInfo], logFlags)),
	// warnLog:
	newConsoleLog(
		consoleLogPrefix[liWarn],
		os.Stderr,
		log.New(os.Stderr, consoleLogPrefix[liWarn], logFlags)),
	// errLog:
	newConsoleLog(
		consoleLogPrefix[liError],
		os.Stderr,
		log.New(os.Stderr, consoleLogPrefix[liError], logFlags)),
}

// single instantiation of each of the loggers for all goroutines to share
// indirectly through use of the exported subroutines below.
var (
	isVerboseLog bool // flag used by loggers -only- for determining verbosity

	rawLog  *ConsoleLog = consoleLog[liRaw]
	infoLog *ConsoleLog = consoleLog[liInfo]
	warnLog *ConsoleLog = consoleLog[liWarn]
	errLog  *ConsoleLog = consoleLog[liError]
)

// function newConsoleLog() creates a new ConsoleLog struct with the given
// args as fields and a new sync.Mutex semaphore all its very own.
func newConsoleLog(prefix string, writer io.Writer, logger *log.Logger) *ConsoleLog {
	return &ConsoleLog{prefix, writer, logger, *new(sync.Mutex)}
}

// function setWriter() changes the log writer to anything conforming to the
// io.Writer interface. this may be a file, I/O stream, ncurses panel, etc.
func (l *ConsoleLog) setWriter(w io.Writer) {
	if l.writer != w {
		l.Lock()
		l.writer = w
		l.Logger = log.New(w, l.prefix, l.Flags())
		l.Unlock()
	}
}

// function output() outputs a given string using the current properties of the
// target logger. this function is the final stop in the call stack for all of
// the logging subroutines exported by this unit, so it is possible to modify or
// simply toggle ON or OFF all of the output by changing this subroutine.
func (l *ConsoleLog) output(s string) {
	if false {
		return
	}
	if l != rawLog {
		l.Print(fmt.Sprintf("%s%s", logDelimiter, s))
	} else {
		l.Print(s)
	}
}

// function log() outputs a given string using the current properties of the
// logger and each of the variable-number-of arguments.
func (l *ConsoleLog) log(v ...interface{}) {
	s := fmt.Sprint(v...)
	l.output(s)
}

// function logf() outputs a given string using the current properties of the
// logger and any specified printf-style format string + arguments.
func (l *ConsoleLog) logf(format string, v ...interface{}) {
	s := fmt.Sprintf(format, v...)
	l.output(s)
}

// function vlog() outputs a given string using the current properties of the
// logger and each of the variable-number-of arguments. the string will only
// be output if the verbose flag was set.
func (l *ConsoleLog) vlog(v ...interface{}) {
	if isVerboseLog {
		l.log(v...)
	}
}

// function vlogf() outputs a given string using the current properties of the
// logger and any specified printf-style format string + arguments. the string
// will only be output if the verbose flag was set.
func (l *ConsoleLog) vlogf(format string, v ...interface{}) {
	if isVerboseLog {
		l.logf(format, v...)
	}
}

// function logStackTrace() prints the entire stack trace
func (l *ConsoleLog) logStackTrace() {
	byt := debug.Stack()
	str := string(byt[:])
	res := regexp.MustCompile("[\\r\\n]+").Split(str, -1)

	for n, s := range res[:len(res)-1] {
		l.logf("%d: %s", n, s)
	}
}

// function die() outputs the details of a given ReturnCode object, and then
// terminates program execution with the ReturnCode object's return value.
func (l *ConsoleLog) die(c *ReturnCode, trace bool) {
	if rcUsage != c {
		s := fmt.Sprintf("%s", error(c))
		l.output(s)
		if trace && isVerboseLog {
			l.logStackTrace()
		}
	}
	os.Exit(c.code)
}
