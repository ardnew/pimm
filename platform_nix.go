// +build linux darwin

// =============================================================================
//  PROJ: pimmp
//  AUTH: ardnew
//  DATE: 26 Sept 2018
//  FILE: platform_nix.go
// -----------------------------------------------------------------------------
//
//  DESCRIPTION
//    all data related to system-dependent operations can be stored here
//    unconditionally. the Go compiler takes care of identifying which of the
//    symbols to export based on whichever platform we are targeting.
//
// =============================================================================

package main

import (
	"os"
)

const (
	newLine = "\n"
	pathSep = "/"
)

// function homeDir() returns the path to the user's home directory as defined
// by the user's current HOME environment variable.
func homeDir() string {
	return os.Getenv("HOME")
}
