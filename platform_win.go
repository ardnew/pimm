// +build windows

// =============================================================================
//  PROJ: pimmp
//  AUTH: ardnew
//  DATE: 26 Sept 2018
//  FILE: platform_win.go
// -----------------------------------------------------------------------------
//
//  DESCRIPTION
//    all data related to system-dependent operations can be stored here
//    unconditionally. the Go compiler takes care of identifying which of the
//    symbols to export based on whichever platform we are targeting.
//
// =============================================================================

package main

const (
	Newline = "\r\n"
	PathSep = "\\"
)
