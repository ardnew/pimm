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
//    if you add anything to one of these platform-specific definitions files
//    (platform_nix.go, platform_win.go) that is used by any of the client
//    application (pimmp) code, then be sure to implement it in ~ALL~ of the
//    platform-specific definitions files.
//
// =============================================================================

package main

import (
	"os"
)

const (
	newLine = "\r\n"
	pathSep = "\\"
	currDir = "."
)

// function homeDir() returns the path to the user's home directory as defined
// by several of the user's current environment variables.
func homeDir() string {
	home := os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
	if home == "" {
		home = os.Getenv("USERPROFILE")
	}
	return home
}
