// =============================================================================
//  PROJ: pimmp
//  AUTH: ardnew
//  DATE: 02 Oct 2018
//  FILE: error.go
// -----------------------------------------------------------------------------
//
//  DESCRIPTION
//    defines types and functions for describing return values and error codes
//    to the user
//
// =============================================================================

package main

import (
	"fmt"
	"strings"
)

// type ReturnCodeKind identifies the different kinds of return codes.
type ReturnCodeKind int

const (
	rkInfo ReturnCodeKind = iota
	rkWarn
	rkError
)

// type ReturnCode contains information describing the reason for program exit,
// including potential runtime errors with detailed diagnostic info.
type ReturnCode struct {
	kind ReturnCodeKind // type of return code; affects how the message is displayed
	code int            // value between 0 and 255 (inclusive) for portability
	desc string         // built-in description of this general purpose return code
	info string         // additional detail elaborating the return event
}

// private constants
const (
	errorOffset   = 100
	maxReturnCode = 255
)

var (
	// non-error return codes
	rcOK    = newReturnCode(rkInfo, 0, "ok", "")    // no errors, normal return
	rcUsage = newReturnCode(rkInfo, 1, "usage", "") // no errors, displays usage help

	// error return codes
	rcInvalidArgs      = newReturnCode(rkError, errorOffset+0, "invalid arguments", "")          // invalid command line args
	rcInvalidLibrary   = newReturnCode(rkWarn, errorOffset+1, "invalid library", "")             // invalid library
	rcLibraryBusy      = newReturnCode(rkWarn, errorOffset+2, "library busy", "")                // library busy with other tasks
	rcInvalidPath      = newReturnCode(rkWarn, errorOffset+3, "invalid path", "")                // invalid path
	rcInvalidStat      = newReturnCode(rkWarn, errorOffset+4, "error reading file stat", "")     // file stat error
	rcDirDepth         = newReturnCode(rkWarn, errorOffset+5, "search depth limit exceeded", "") // directory traversal depth limit exceeded
	rcDirOpen          = newReturnCode(rkWarn, errorOffset+6, "cannot open directory", "")       // cannot open directory for reading
	rcInvalidFile      = newReturnCode(rkWarn, errorOffset+7, "invalid file", "")                // some invalid type of file (symlink, FIFO, etc.)
	rcInvalidConfig    = newReturnCode(rkError, errorOffset+8, "invalid configuration", "")      // invalid configuration settings
	rcInvalidDatabase  = newReturnCode(rkWarn, errorOffset+9, "invalid database", "")            // failed to create a media database
	rcDatabaseError    = newReturnCode(rkWarn, errorOffset+10, "database operation failed", "")  // failed to perform an operation on the database
	rcDuplicateLibrary = newReturnCode(rkWarn, errorOffset+11, "duplicate library", "")          // duplicate; library path already being handled
	rcInvalidJSONData  = newReturnCode(rkWarn, errorOffset+12, "invalid JSON data", "")          // cannot handle some JSON-related data object
	rcUnknown          = newReturnCode(rkError, maxReturnCode, "unknown error", "")              // unanticipated error encountered
)

// function newReturnCode() constructs a new ReturnCode object with a specified
// return code, description, and info.
func newReturnCode(kind ReturnCodeKind, code int, desc string, info string) *ReturnCode {
	return &ReturnCode{kind, code, desc, info}
}

// function spec() replaces the info string of an existing ReturnCode object
// with the specified string and returns the updated ReturnCode object. the
// existing return code and description fields are left unchanged.
func (c *ReturnCode) spec(info string) *ReturnCode {
	c.info = info
	return c
}

// function specf() is a wrapper for function spec() that constructs the
// string using the specified printf-style format strings + arguments.
func (c *ReturnCode) specf(format string, v ...interface{}) *ReturnCode {
	s := fmt.Sprintf(format, v...)
	return c.spec(s)
}

// function kspecf() is a wrapper for function specf() that changes the kind of
// ReturnCode from the default.
func (c *ReturnCode) kspecf(kind ReturnCodeKind, format string, v ...interface{}) *ReturnCode {
	c.kind = kind
	return c.specf(format, v...)
}

// function Error() constructs an error message using the current fields of a
// ReturnCode object. the Code and Desc fields are required, but Info is not. if
// info is an empty string or contains only whitespace, then it will not be
// included in the returned string.
func (c *ReturnCode) Error() string {
	var pre string
	switch c.kind {
	case rkInfo:
		pre = ""
	case rkWarn:
		pre = fmt.Sprintf("[W%d] ", c.code)
	case rkError:
		pre = fmt.Sprintf("[E%d] ", c.code)
	}
	s := fmt.Sprintf("%s%s", pre, c.desc)
	i := strings.TrimSpace(c.info)
	if len(i) > 0 {
		s = fmt.Sprintf("%s: %s", s, i)
	}
	return s
}
