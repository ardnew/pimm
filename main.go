// =============================================================================
//  PROJ: pimmp
//  AUTH: ardnew
//  DATE: 26 Sept 2018
//  FILE: main.go
// -----------------------------------------------------------------------------
//
//  DESCRIPTION
//    (TBD)
//
// =============================================================================

package main

// import (
// 	"fmt"
// )

func main() {

	// var (
	// 	kind MediaKind
	// 	name string
	// )

	// kind, name = MediaKindOfFileExt("yuv")
	// rawLog.Logln("[Logln()] %d: %s%s", kind, name)

	// kind, name = MediaKindOfFileExt("m4b")
	// infoLog.Logln("[Logln()] %d: %s%s", kind, name)

	// kind, name = MediaKindOfFileExt("xxx")
	// warnLog.Logln("[Logln()] %d: %s%s", kind, name)

	// kind, name = MediaKindOfFileExt("dct")
	// errLog.Logln("[Logln()] %d: %s%s", kind, name)

	_, err := NewLibrary("..")
	if nil != err {
		errLog.Die(err, true)
	}

}
