// =============================================================================
//  PROJ: pimmp
//  AUTH: ardnew
//  DATE: 02 Oct 2018
//  FILE: error.go
// -----------------------------------------------------------------------------
//
//  DESCRIPTION
//    defines types and functions for describing errors to the user
//
// =============================================================================

package main

import (
	"fmt"
	"strings"
)

// type ErrorCode contains potential program runtime errors with detailed
// diagnostic info.
type ErrorCode struct {
	Code int    // value of error code
	Desc string // built-in description of this general purpose error code
	Info string // additional detail elaborating the error event
}

var (
	EOK             = &ErrorCode{0, "ok", ""}               // no errors, normal return
	EInvalidLibrary = &ErrorCode{10, "invalid library", ""} // invalid library
	EUnknown        = &ErrorCode{254, "unknown error", ""}  // unanticipated error encountered
	EUsage          = &ErrorCode{255, "usage", ""}          // no errors, displays usage help
)

// function NewErrorCode() constructs a new ErrorCode object with a specified
// error code, description, and info.
func NewErrorCode(code int, desc string, info string) *ErrorCode {
	return &ErrorCode{code, desc, info}
}

// function WithInfo() replaces the info string of an existing ErrorCode object
// with the specified string and returns the new ErrorCode object. the existing
// error code and description fields are left unchanged.
func (c *ErrorCode) WithInfo(info string) *ErrorCode {
	c.Info = info
	return c
}

// function Error() constructs an error message using the current fields of an
// ErrorCode object. the Code and Desc fields are required, but Info is not. if
// info is an empty string or contains only whitespace, then it will not be
// included in the returned string.
func (c *ErrorCode) Error() string {
	s := fmt.Sprintf("%s (%d)", c.Desc, c.Code)
	i := strings.TrimSpace(c.Info)
	if len(i) > 0 {
		s = fmt.Sprintf("%s: %s", s, i)
	}
	return s
}
