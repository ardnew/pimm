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

// type ReturnCode contains information describing the reason for program exit,
// including potential runtime errors with detailed diagnostic info.
type ReturnCode struct {
	code int    // value between 0 and 255 (inclusive) for portability
	desc string // built-in description of this general purpose return code
	info string // additional detail elaborating the return event
}

// private constants
const (
	errorOffset   = 100
	maxReturnCode = 255
)

var (
	// non-error return codes
	rcOK    = newReturnCode(0, "ok", "")    // no errors, normal return
	rcUsage = newReturnCode(1, "usage", "") // no errors, displays usage help

	// error return codes
	rcInvalidArgs     = newReturnCode(errorOffset+0, "invalid arguments", "")           // invalid command line args
	rcInvalidLibrary  = newReturnCode(errorOffset+1, "invalid library", "")             // invalid library
	rcLibraryBusy     = newReturnCode(errorOffset+2, "library busy", "")                // library busy with other tasks
	rcInvalidPath     = newReturnCode(errorOffset+3, "invalid path", "")                // invalid path
	rcInvalidStat     = newReturnCode(errorOffset+4, "error reading file stat", "")     // file stat error
	rcDirDepth        = newReturnCode(errorOffset+5, "search depth limit exceeded", "") // directory traversal depth limit exceeded
	rcDirOpen         = newReturnCode(errorOffset+6, "cannot open directory", "")       // cannot open directory for reading
	rcInvalidFile     = newReturnCode(errorOffset+7, "invalid file", "")                // some invalid type of file (symlink, FIFO, etc.)
	rcInvalidConfig   = newReturnCode(errorOffset+8, "invalid configuration", "")       // invalid configuration settings
	rcInvalidDatabase = newReturnCode(errorOffset+9, "invalid database", "")            // failed to create a media database
	rcDatabaseError   = newReturnCode(errorOffset+10, "database operation failed", "")  // failed to perform an operation on the database
	rcUnknown         = newReturnCode(maxReturnCode, "unknown error", "")               // unanticipated error encountered
)

// function newReturnCode() constructs a new ReturnCode object with a specified
// return code, description, and info.
func newReturnCode(code int, desc string, info string) *ReturnCode {
	return &ReturnCode{code, desc, info}
}

// function withInfo() replaces the info string of an existing ReturnCode object
// with the specified string and returns the updated ReturnCode object. the
// existing return code and description fields are left unchanged.
func (c *ReturnCode) withInfo(info string) *ReturnCode {
	c.info = info
	return c
}

// function withInfof() is a wrapper for function withInfo() that constructs the
// string using the specified printf-style format strings + arguments.
func (c *ReturnCode) withInfof(format string, v ...interface{}) *ReturnCode {
	s := fmt.Sprintf(format, v...)
	return c.withInfo(s)
}

// function isError() determines if the return code value of a ReturnCode object
// is defined as one of the error codes
func (c *ReturnCode) isError() bool {
	return c.code >= errorOffset
}

// function Error() constructs an error message using the current fields of a
// ReturnCode object. the Code and Desc fields are required, but Info is not. if
// info is an empty string or contains only whitespace, then it will not be
// included in the returned string.
func (c *ReturnCode) Error() string {
	s := fmt.Sprintf("[C%d] %s", c.code, c.desc)
	i := strings.TrimSpace(c.info)
	if len(i) > 0 {
		s = fmt.Sprintf("%s: %s", s, i)
	}
	return s
}
