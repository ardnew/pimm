project    = pimm
importpath = ardnew.com/$(project)

version    = 1.0
revision   = 7
buildtime  = $(shell date -u "+%FT%TZ")

ldflags    = -ldflags '-w -s -X "main.VERSION=$(version)" -X "main.REVISION=$(revision)" -X "main.BUILDTIME=$(buildtime)"'

.PHONY: build
build:
	go build $(ldflags) "$(importpath)"

.PHONY: install
install:
	go install $(ldflags) "$(importpath)"