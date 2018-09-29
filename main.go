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

import (
	"fmt"
)

func main() {

	var (
		kind MediaKind
		name string
	)

	kind, name = MediaKindOfFileExt("yuv")
	fmt.Printf("%d: %s%s", kind, name, Newline)

	kind, name = MediaKindOfFileExt("m4b")
	fmt.Printf("%d: %s%s", kind, name, Newline)

	kind, name = MediaKindOfFileExt("xxx")
	fmt.Printf("%d: %s%s", kind, name, Newline)
}
