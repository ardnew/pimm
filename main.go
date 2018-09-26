package main

import (
	"fmt"
)

func main() {

	var (
		kind MediaKind
		name string
	)

	kind, name = FileExtMediaKind("yuv")
	fmt.Printf("%d: %s%s", kind, name, Newline)

	kind, name = FileExtMediaKind("m4b")
	fmt.Printf("%d: %s%s", kind, name, Newline)

	kind, name = FileExtMediaKind("xxx")
	fmt.Printf("%d: %s%s", kind, name, Newline)
}
